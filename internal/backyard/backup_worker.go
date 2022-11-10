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

		name, timemodified, timeborn, size := filepath.Base(f0.Name), f0.TimeModified, f0.TimeBorn, f0.Size
		for _, f := range job.Files {
			f_name := filepath.Base(f.Name)

			if f.Size != size || (f.TimeBornSrc == TimeBornSrcMeta && f.TimeBorn != timeborn) {
				log.Fatalf("BackupWorker: conflicted files(size, or birth) - %+v, size=%v, born=%v", f, size, timeborn)
			}
			if f.TimeModified < timemodified {
				timemodified = f.TimeModified
			}
			if f_name != name {
				log.Warnf("BackupWorker: id=%v with another name %v", f.Id, f_name)
			}
			if len(f_name) < len(name) {
				name = f_name //prefer short name
			}
			if f.TimeBorn < timeborn {
				timeborn = f.TimeBorn
			}
		}
		//make the destination to backup
		birth := time.Unix(timeborn, 0).Local()
		dest := "/" + f0.MIMEType + "/" + birth.Format("2006/01/02") + "/" + name
		log.Infof("BackupWorker: choose birth=%v dest=%v - fi=%+v", birth, dest, f0)
	}
}
