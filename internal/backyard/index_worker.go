package backyard

import (
	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
)

type IndexOptions struct {
	Path       string
	BackupPath string
	CachePath  string
	NumWorkers int
	Rescan     bool
	Convert    bool
	Stack      bool
}

type IndexJob struct {
	FileName string
	IndexOpt IndexOptions
	Ind      *Index
	ChDB     chan *FileIndexed
}

func IndexWorker(jobs <-chan IndexJob, et *exiftool.Exiftool) {
	for job := range jobs {
		log.Infof("IndexWorker:                           fileName=%s", job.FileName)
		mainIndex(job.FileName, job.Ind, job.IndexOpt, et, job.ChDB)

	}
}

func mainIndex(fileName string, ind *Index, opt IndexOptions, exifTool *exiftool.Exiftool, chDB chan *FileIndexed) {
	log.Infof("mainIndex: entering, %v , %v", fileName, exifTool)

	sizeLimit := config.OriginalsLimit()

	err, fi := NewFileIndex(fileName)

	if err != nil || fi == nil || fi.Size <= 0 || fi.Size > sizeLimit {
		log.Errorf("mainIndex: NewFileIndex - wrong of file size of  err=%v, fi=%v", err, fi)
		return
	}

	fi.TryExif(exifTool)

	//	result = ind.MediaFile(f, opt, "")
	//	takenAt, src := f.TakenAt()

	//	log.Infof("index_main: DONE mf=%s(%), %s %s.%s", f.FileName(), f.FileType(), f.Hash(), takenAt, src)

	chDB <- fi

	//	log.Infof("mainIndex:  DONE %s %s %s %s, paths=%v", fi.ID, fi.Path, takenAt, takenAtSrc, fi.Duplica)
}
