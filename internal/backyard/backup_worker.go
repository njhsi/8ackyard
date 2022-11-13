package backyard

import (
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/photoprism/photoprism/pkg/fs"
)

type BackupOptions struct {
	OriginalsPath string
	BackupPath    string
	CachePath     string
	NumWorkers    int
	Rescan        bool
}

type BackupJob struct {
	Id        int64
	BackupOpt BackupOptions
	Files     []*File8
	BackFile  *File8 //existed in db
	ChDB      chan *File8
}

func BackupWorker(jobs <-chan *BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:     got a job[%v %v, Backfile=%v], %v files. uniq=%v ",
			job.Files[0].Id, job.Files[0].Name, job.BackFile, len(job.Files), len(job.Files) == 1)

		f0 := job.Files[0]
		if job.BackFile != nil {
			f0 = job.BackFile
		}

		if f0.MIMEType != "video" && f0.MIMEType != "audio" && f0.MIMEType != "image" {
			fb := &File8{Id: f0.Id, Size: 0} //must send back to count on
			log.Warnf("BackupWorker: ignore this mime[%v]..... %+v", f0.MIMEType, f0)
			job.ChDB <- fb
			continue
		}

		fb := *f0 //clone
		fb.backup_ = job.BackFile
		fb_basename := filepath.Base(fb.Name)

		for _, f := range job.Files {
			f_basename := filepath.Base(f.Name)

			if f.Size != fb.Size || (f.TimeBornSrc == TimeBornSrcMeta && f.TimeBorn != fb.TimeBorn) {
				log.Fatalf("BackupWorker: conflicted files(size, or birth) - %+v, size=%v, born=%v", f, fb.Size, fb.TimeBorn)
			}
			if f.TimeModified < fb.TimeModified {
				fb.TimeModified = f.TimeModified
			}
			if f_basename != fb_basename {
				log.Warnf("BackupWorker: id=%v with another name %v", f.Id, f_basename)
			}
			if len(f_basename) < len(fb_basename) { //TODO: other names could be symlink to the prefered name in backup folder
				fb_basename = f_basename //prefer short name
			}
			if f.TimeBorn < fb.TimeBorn {
				fb.TimeBorn = f.TimeBorn
			}
		}
		//make the destination to backup
		birth := time.Unix(fb.TimeBorn, 0).Local()
		dest := job.BackupOpt.BackupPath + "/" + f0.MIMEType + "/" + birth.Format("2006/01/02") + "/" + fb_basename
		dest = path.Clean(dest)

		//do backup on disk: 1)check if existed on disk
		path_final := ""                                             // if backup confirmed finished on disk
		if job.BackFile != nil && fs.FileExists(job.BackFile.Name) { //TODO: hostname check
			id_fb_ondisk := int64(fileXXH3(job.BackFile.Name))
			if id_fb_ondisk == job.BackFile.Id {
				//return after confirm naming
				log.Infof("BackupWorker: job.BackFile(%v) existed on disk with same id(%v), might do rename to dest=%v ", job.BackFile.Name, id_fb_ondisk, dest)
				path_final = dest
				if dest != job.BackFile.Name {
					if err := os.Rename(job.BackFile.Name, dest); err != nil {
						log.Warnf("BackupWorker: existed on disk with same id, but os.Rename failed %v -> %v", job.BackFile.Name, dest)
						path_final = "" //reset
					}
				}
			} else {
				log.Warnf("BackupWorker: rotten bits or normal names duplicated... fi=%+v, id_fb_ondisk=%v", job.BackFile, id_fb_ondisk)
			}
		}
		for len(path_final) == 0 && fs.FileExists(dest) {
			//id_f_ondisk := fb.Id
			id_f_ondisk := int64(fileXXH3(dest)) //TODO: stat check to speed up..
			if fb.Id == id_f_ondisk {
				path_final = dest
				log.Infof("BackupWorker: dest=%v existed on disk with same id of fb=%+v", dest, fb)
				//TODO: confirm stats
				break
			} else {
				log.Warnf("BackupWorker: dest=%v existed on disk with different id[%v], fb=%+v", dest, id_f_ondisk, fb)
				dest = dest + "-" + Int64ToString(fb.Id) + "_XXH3"
				if len(dest) > 256 {
					log.Fatalf("BackupWorker: can not choose dest(%v) at all, fb=%+v", dest, fb)
				}
			}
		}
		if len(path_final) == 0 && len(dest) > 0 {
			for _, f := range job.Files {
				if err, mtime, size := fileStat(f.Name); err == nil &&
					size == f.Size && mtime.Unix() == f.TimeModified {
					log.Infof("BackupWorker: going to do copy on disk %v->%v, id=%v ", f.Name, dest, f.Id)
					if err := CopyWithStat(f.Name, dest); err == nil { //!!TODO: stat
						path_final = dest
						break
					} else {
						log.Warnf("BackupWorker: failed to do copy on disk %v->%v, id=%v . err=%v ", f.Name, dest, f.Id, err)
					}
				}
			}
		}

		//update fb
		fb.Name = path_final

		job.ChDB <- &fb

		log.Infof("BackupWorker:     choose birth=%v dest=%v \n  fi=%+v", birth, dest, f0)
	}
}
