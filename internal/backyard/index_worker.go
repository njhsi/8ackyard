package backyard

import (
	"io/ioutil"

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

	if jbuf, err := buildExifJson(fileName, exifTool); err == nil {
		exif := &ExifData{}
		if err := exif.DataFromExiftool(jbuf); err != nil || exif.TakenAt.Year() < 1980 {
			log.Errorf("mainIndex: DataFromExiftool $v - err=%v, exif=%v", fileName, err, exif)
			//			return
		} else {
			log.Infof("mainIndex: exif(%v) -  %v", fileName, exif)
			fi.TimeBorn, fi.TimeBornSrc = exif.TakenAt, TimeBornSrcMeta //TODO: exif.TimeZone
			ids := Uint64ToString(fi.ID)
			if exifJson, err := CacheName(ids, "json", "exiftool.json"); err == nil {
				ioutil.WriteFile(exifJson, jbuf, 0644)
			}
		}
	}

	chDB <- fi

	log.Infof("mainIndex:  DONE(%v) - %v - %v", fileName, fi, err)
}
