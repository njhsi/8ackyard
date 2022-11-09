package backyard

type FileBacked struct {
	ID      uint64           //xxh3 hash of file content
	Size    int64            // file size
	Path    string           //full path, regular file existed on device
	Duplica map[string]int64 //fullpath:modtime
	Type    string
	Info    string
}

type BackupOptions struct {
	OriginalsPath string
	BackupPath    string
	CachePath     string
	NumWorkers    int
	Rescan        bool
}

type BackupJob struct {
	BackupOpt BackupOptions
	File      *FileIndexed
	ChDB      chan *FileBacked
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:                           mfs=%d", len(job.File.Path))
		if fb := mainBackup(job.File, job.BackupOpt); fb != nil {
			job.ChDB <- fb
		}
	}
}

func mainBackup(file *FileIndexed, opt BackupOptions) *FileBacked {
	//	timeLoc, _ := time.LoadLocation("Asia/Chongqing")
	//	baseName, fullPath, mType, hash := path.Base(file.Path), file.Path, file.Info, file.Id
	//	takenAt, takenAtSrc := time.Unix(file.TimeBorn.Unix(), 0).In(timeLoc), file.TimeBornSrc
	return nil

}
