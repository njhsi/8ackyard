package backyard

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
	ChDB      chan *File8
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:                           id=%d", job.Id)
	}
}
