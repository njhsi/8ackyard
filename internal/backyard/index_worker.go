package backyard

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"

	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/fs"
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
	//	log.Infof("mainIndex: entering, %v , %v", fileName, exifTool)

	sizeLimit := config.OriginalsLimit()

	err, fi := NewFileIndex(fileName)
	if err != nil || fi == nil || fi.Size <= 0 || fi.Size > sizeLimit {
		log.Errorf("mainIndex: NewFileIndex - wrong of file size of %v,  err=%v, fi=%v", fileName, err, fi)
		return
	}

	exif := &ExifData{}
	idStr := Uint64ToString(fi.ID)
	exifJson, err := CacheName(idStr, "json", "exiftool.json")
	if err != nil {
		log.Fatalf("mainIndex: CacheName - %v %v", fileName, err)
	}
	if fs.FileExists(exifJson) {
		log.Infof("mainIndex: json %v existed ..", exifJson)
		jsonFile, err := os.Open(exifJson)
		if err != nil {
			log.Fatalf("mainIndex: Open - %v %v", fileName, err)
		}
		defer jsonFile.Close()
		var jbuf bytes.Buffer
		jbuf.ReadFrom(jsonFile)
		if err = exif.DataFromExiftool(jbuf.Bytes()); err != nil {
			log.Errorf("mainIndex: exif.DataFromExiftool %v %v", exifJson, err)
		}
	} else {
		if jbuf, err := buildExifJson(fileName, exifTool); err == nil {
			if err := exif.DataFromExiftool(jbuf); err != nil {
				log.Errorf("mainIndex: DataFromExiftool %v - err=%v, exif=%v", fileName, err, exif)
			}
			if exif.TakenAt.Year() > 1900 {
				ioutil.WriteFile(exifJson, jbuf, 0644)
			}
		}
	}

	if len(fi.MIMEType) == 0 && len(exif.MIMEType) > 0 {
		mts := strings.Split(exif.MIMEType, "/")
		fi.MIMEType, fi.MIMESubtype = mts[0], mts[1]
	}
	if exif.TakenAt.Year() > 1900 {
		fi.TimeBorn, fi.TimeBornSrc = exif.TakenAt, TimeBornSrcMeta //TODO: exif.TimeZone
	}

	chDB <- fi

	log.Infof("mainIndex:  DONE(%v) - fi=%v | exif=%v |  err=%v", fileName, fi, exif, err)
}
