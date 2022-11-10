package backyard

import (
	"path/filepath"
	"time"
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
	BackFile  *File8
	ChDB      chan *File8
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		f0 := job.Files[0]
		log.Infof("BackupWorker:     got a job[%v %v], %v files. uniq=%v ", f0.Id, f0.Name, len(job.Files), len(job.Files) == 1)
		if f0.MIMEType != "video" && f0.MIMEType != "audio" && f0.MIMEType != "image" {
			log.Warnf("BackupWorker: ignore this mime[%v]..... %+v", f0.MIMEType, f0)
			continue
		}

		fb := *f0 //clone
		fb.Name = filepath.Base(fb.Name)

		for _, f := range job.Files {
			f_name := filepath.Base(f.Name)

			if f.Size != fb.Size || (f.TimeBornSrc == TimeBornSrcMeta && f.TimeBorn != fb.TimeBorn) {
				log.Fatalf("BackupWorker: conflicted files(size, or birth) - %+v, size=%v, born=%v", f, fb.Size, fb.TimeBorn)
			}
			if f.TimeModified < fb.TimeModified {
				fb.TimeModified = f.TimeModified
			}
			if f_name != fb.Name {
				log.Warnf("BackupWorker: id=%v with another name %v", f.Id, f_name)
			}
			if len(f_name) < len(fb.Name) {
				fb.Name = f_name //prefer short name
			}
			if f.TimeBorn < fb.TimeBorn {
				fb.TimeBorn = f.TimeBorn
			}
		}
		//make the destination to backup
		birth := time.Unix(fb.TimeBorn, 0).Local()
		dest := "/" + f0.MIMEType + "/" + birth.Format("2006/01/02") + "/" + fb.Name
		//update fb

		//do backup on disk: 1)check if existed on disk

		job.ChDB <- &fb

		log.Infof("BackupWorker: choose birth=%v dest=%v - fi=%+v", birth, dest, f0)
	}
}
