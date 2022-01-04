package backyard

import (
	"fmt"
	"path/filepath"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/sanitize"
)

type IndexJob struct {
	FileName string
	IndexOpt IndexOptions
	Ind      *Index
}

func IndexWorker(jobs <-chan IndexJob) {
	for job := range jobs {
		log.Infof("IndexWorker:                           %s", job.FileName)
		index_main(job.FileName, job.Ind, job.IndexOpt)
	}
}

func index_main(fileName string, ind *Index, opt IndexOptions) (result IndexResult) {
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

	if f.NeedsExifToolJson() {
		if jsonName, err := f.ToJson(); err != nil {
			log.Debugf("index: %s in %s (extract metadata)", sanitize.Log(err.Error()), sanitize.Log(f.BaseName()))
		} else {
			log.Debugf("index: created %s", filepath.Base(jsonName))
			f.ReadExifToolJson()
		}
	}

	//	result = ind.MediaFile(f, opt, "")
	result.Status = IndexAdded
	takenAt, src := f.TakenAt()
	ind.files.Add(f)

	log.Infof("index: %s ma!n %s file %s %s.%s", result, f.FileType(), sanitize.Log(f.RelName(ind.originalsPath())), takenAt, src)

	return result
}
