package backyard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/sanitize"
)

type IndexJob struct {
	FileName string
	IndexOpt IndexOptions
	Ind      *Index
}

func IndexWorker(jobs <-chan IndexJob, et *exiftool.Exiftool) {
	for job := range jobs {
		log.Infof("IndexWorker:                           fileName=%s", job.FileName)
		index_main(job.FileName, job.Ind, job.IndexOpt, et)
	}
}

func index_main(fileName string, ind *Index, opt IndexOptions, exifTool *exiftool.Exiftool) (result IndexResult) {
	f, err := NewMediaFile(fileName)
	if err != nil {
		result.Err = fmt.Errorf("index: found no  mediafile for %s", sanitize.Log(fileName))
		result.Status = IndexFailed
		return result
	}

	sizeLimit := config.OriginalsLimit()

	// Enforce file size limit for originals.
	if sizeLimit > 0 && f.FileSize() > sizeLimit {
		result.Err = fmt.Errorf("index: %s (%d/%dM)", sanitize.Log(f.BaseName()), f.FileSize()/(1024*1024), sizeLimit/(1024*1024))
		result.Status = IndexFailed
		return result
	}

	if exifTool != nil && f.NeedsExifToolJson() {
		fileInfos := exifTool.ExtractMetadata(fileName)
		for _, fileInfo := range fileInfos {
			if fileInfo.Err != nil {
				log.Errorf("index: Error in exiftool %v: %v\n", fileInfo.File, fileInfo.Err)
				continue
			}
			jsonName, err1 := f.ExifToolJsonName()
			jsonFile, err2 := json.MarshalIndent(fileInfo.Fields, "", "")
			if err1 == nil && err2 == nil {
				ioutil.WriteFile(jsonName, jsonFile, 0644)
			} else {
				log.Errorf("index: exifTool on %s %s,%s,%s|%s", fileInfo.File, f.Hash(), jsonName, err1, err2)

			}

		}
	}

	if f.NeedsExifToolJson() {
		if jsonName, err := f.ToJson(); err != nil {
			log.Debugf("index: %s in %s (extract metadata)", sanitize.Log(err.Error()), sanitize.Log(f.BaseName()))
		} else {
			log.Debugf("index: created %s (extract metadata)", filepath.Base(jsonName))
			f.ReadExifToolJson()
		}
	}

	//	result = ind.MediaFile(f, opt, "")
	result.Status = IndexAdded
	takenAt, src := f.TakenAt()
	ind.files.Add(f)

	log.Infof("index: %s ma!n %s file %s %s.%s", result, f.FileType(), sanitize.Log(f.RelName(opt.Path)), takenAt, src)

	return result
}
