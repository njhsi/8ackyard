package backyard

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/meta"
	"github.com/photoprism/photoprism/pkg/fs"
)

type IndexOptions struct {
	Path       string
	BackupPath string
	CachePath  string
	Hostname   string
	NumWorkers int
	Rescan     bool
	Convert    bool
	Stack      bool
}

type IndexJob struct {
	FileName string
	IndexOpt IndexOptions
	Ind      *Index
	ChDB     chan *File8
}

func IndexWorker(jobs <-chan IndexJob, et *exiftool.Exiftool) {
	for job := range jobs {
		log.Infof("IndexWorker:                           fileName=%s", job.FileName)
		mainIndex(job.FileName, job.Ind, job.IndexOpt, et, job.ChDB)

	}
}

func mainIndex(fileName string, ind *Index, opt IndexOptions, exifTool *exiftool.Exiftool, chDB chan *File8) {
	//	log.Infof("mainIndex: entering, %v , %v", fileName, exifTool)

	sizeLimit := config.OriginalsLimit()

	err, fi := NewFileIndex(fileName)
	if err != nil || fi == nil || fi.Size <= 0 || fi.Size > sizeLimit {
		log.Errorf("mainIndex: NewFileIndex - wrong of file size of %v,  err=%v, fi=%v", fileName, err, fi)
		return
	}
	if len(opt.Hostname) > 0 {
		fi.Hostname = opt.Hostname
	}

	exif := &meta.Data{}
	idStr := Int64ToString(fi.Id)
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
		if err = exif.Exiftool(jbuf.Bytes(), ""); err != nil { //TODO: exif.JSON(exifJson,"")
			log.Errorf("mainIndex: exif.DataFromExiftool %v %v", exifJson, err)
		}
	} else {
		if jbuf, err := buildExifJson(fileName, exifTool); err == nil {
			if err := exif.Exiftool(jbuf, ""); err != nil {
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
		takeAt := exif.TakenAt
		if len(exif.TimeZone) == 0 {
			timeLoc, _ := time.LoadLocation("Asia/Chongqing")
			if len(exif.OffsetTimeOriginal) > 0 {
				layouts := [4]string{"+08:00", "+0800", "-08:00", "-0800"}
				for _, layout := range layouts {
					if tos, err := time.Parse(layout, exif.OffsetTimeOriginal); err == nil {
						timeLoc = tos.Location()
						log.Infof("mainIndex: lookup timezone by layout=%v, got loc=%v", layout, timeLoc)
						break
					}
				}

			}
			tDuration := time.Date(takeAt.Year(), takeAt.Month(), takeAt.Day(), 0, 0, 0, 0, timeLoc).Sub(time.Date(takeAt.Year(), takeAt.Month(), takeAt.Day(), 0, 0, 0, 0, takeAt.Location()))
			takeAt = takeAt.Add(tDuration)
			takeAt = takeAt.In(timeLoc)
			log.Infof("mainIndex: exif has no TimeZone, did adjust.  exif.takenat=%v,  duration=%v", exif.TakenAt, tDuration)
		}
		fi.TimeBorn, fi.TimeBornSrc = takeAt.Unix(), TimeBornSrcMeta //TODO: exif.TimeZone
		log.Infof("mainIndex: exif.takenat=%v, fi.timeborn=%v", exif.TakenAt, fi.TimeBorn)
	}

	chDB <- fi

	log.Infof("mainIndex:  DONE(%v) - fi=%+v | exif.takenat=%v,tz=%v, timeoffset=%v | err=%v", fileName, fi, exif.TakenAt, exif.TimeZone, exif.OffsetTimeOriginal, err)
}
