package backyard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/fs"
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
			jsonFile, err2 := json.Marshal(fileInfo.Fields)
			if err1 == nil && err2 == nil {
				log.Infof("index: exifTool.ExtractMetadata on %s %s, -> %s", fileInfo.File, f.Hash(), jsonName)
				ioutil.WriteFile(jsonName, jsonFile, 0644)
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

	log.Infof("index: %s ma!n %s file %s %s.%s", result, f.FileType(), sanitize.Log(f.RelName(ind.originalsPath())), takenAt, src)

	return result
}

type BackupJob struct {
	IndexOpt IndexOptions
	Ind      *Index
	MFiles   MediaFiles
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:                           mfs=%d", len(job.MFiles))
		backup_main(job.MFiles, job.Ind, job.IndexOpt)
	}
}

func backup_main(mFiles MediaFiles, ind *Index, opt IndexOptions) (result IndexResult) {
	sumMfiles := map[string]MediaFiles{}
	if len(mFiles) == 1 { // no need to do hash
		sumMfiles[""] = mFiles
		log.Infof("backup: mf=%s size=%d  sha1=%s", mFiles[0].FileName(), mFiles[0].FileSize(), mFiles[0].Hash())
	} else {
		for _, mf := range mFiles[1:] {
			sumMfiles[mf.Hash()] = append(sumMfiles[mf.Hash()], mf)
			log.Infof("backup: mf=%s size=%d  sha1=%s", mf.FileName(), mf.FileSize(), mf.Hash())
		}
	}
	for _, mfs := range sumMfiles { //TODO: job the vMfiles of each Hash
		var mfBest *MediaFile = nil
		for _, mf := range mfs {
			//TODO: save dups info into a txt file, in case ..
			takenAt, src := mf.TakenAt()
			if src == "meta" {
				mfBest = mf
				break
			} else {
				if mfBest == nil {
					mfBest = mf
				} else {
					takenAtBest, _ := mfBest.TakenAt()
					if takenAt.Before(takenAtBest) {
						mfBest = mf
					}
				}
			}
		}
		//do!
		if mfBest != nil {
			loc, _ := time.LoadLocation("Asia/Chongqing")
			takenAt, src := mfBest.TakenAt()
			takenAt = takenAt.In(loc)
			backupTo := opt.BackupPath + "/" + takenAt.Format("2006/01/02") + "/" + mfBest.BaseName()
			for fs.FileExists(backupTo) {
				backupTo = backupTo + "_8"
			}
			log.Infof("backup: DO!!! [ %s => %s ], %s %s", mfBest.FileName(), backupTo, takenAt, src)
			mfBest.Copy(backupTo)
		}
	}

	result.Status = IndexAdded
	return result
}
