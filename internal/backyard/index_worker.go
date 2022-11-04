package backyard

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/sanitize"
	"github.com/timshannon/badgerhold/v4"
)

type FileIndexed struct {
	ID       string `badgerholdIndex:"ID"` //xxhash of file content
	Path     string //full path
	TimeBorn int64  //unix seconds
	TimeSrc  string //meta, name, auto
	Size     int
	Hash     string `badgerhold:"unique"`
	Format   string // extension etc..
	Mime     string
	Duplica  map[string]int64 //fullpath:modtime
	Info     string
}

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
}

func IndexWorker(jobs <-chan IndexJob, et *exiftool.Exiftool) {
	for job := range jobs {
		log.Infof("IndexWorker:                           fileName=%s", job.FileName)
		mainIndex(job.FileName, job.Ind, job.IndexOpt, et)
	}
}

func mainIndex(fileName string, ind *Index, opt IndexOptions, exifTool *exiftool.Exiftool) error {
	f, err := NewMediaFile(fileName)
	if err != nil {
		log.Errorf("index_main: found no  mediafile for %s", sanitize.Log(fileName))
		return err
	}

	sizeLimit := config.OriginalsLimit()

	// Enforce file size limit for originals.
	if sizeLimit > 0 && f.FileSize() > sizeLimit {
		log.Errorf("index_main: file size xx%s (%d/%dM)", f.FileName(), f.FileSize()/(1024*1024), sizeLimit/(1024*1024))
		return nil
	}

	if exifTool != nil && f.NeedsExifToolJson() {
		fileInfos := exifTool.ExtractMetadata(fileName)
		for _, fileInfo := range fileInfos {
			if fileInfo.Err != nil {
				log.Errorf("index_main: Error in exiftool %v: %v\n", fileInfo.File, fileInfo.Err)
				continue
			}
			jsonName, err1 := f.ExifToolJsonName()
			jsonFile, err2 := json.MarshalIndent(fileInfo.Fields, "", "")
			if err1 == nil && err2 == nil {
				ioutil.WriteFile(jsonName, jsonFile, 0644)
			} else {
				log.Errorf("index_main: exifTool on %s %s,%s,%s|%s", fileInfo.File, f.Hash(), jsonName, err1, err2)

			}

		}
	}

	if f.NeedsExifToolJson() {
		if jsonName, err := f.ToJson(); err != nil {
			log.Debugf("index_main: %s in %s (extract metadata)", sanitize.Log(err.Error()), sanitize.Log(f.BaseName()))
		} else {
			log.Debugf("index_main: created %s (extract metadata)", filepath.Base(jsonName))
			f.ReadExifToolJson()
		}
	}

	//	result = ind.MediaFile(f, opt, "")
	takenAt, src := f.TakenAt()
	add(ind, f)

	log.Infof("index_main: DONE mf=%s(%), %s %s.%s", f.FileName(), f.FileType(), f.Hash(), takenAt, src)

	return nil
}

func add(ind *Index, mf *MediaFile) {
	ind.mutex.Lock()
	defer ind.mutex.Unlock()

	store := ind.storeIndex
	fullPath, mtime := mf.FileName(), mf.modTime.Unix()
	takenAt, takenAtSrc := mf.TakenAt()
	info := "ukn"
	switch {
	case mf.IsPhoto():
		info = "image"
	case mf.IsVideo():
		info = "video"
	case mf.IsAudio():
		info = "audio"
	}

	fi := FileIndexed{
		ID:       mf.Hash(),
		Path:     mf.FileName(),
		TimeBorn: takenAt.Unix(),
		TimeSrc:  takenAtSrc,
		Size:     int(mf.FileSize()),
		Hash:     mf.Hash(),
		Format:   string(mf.FileType()),
		Mime:     mf.MimeType(),
		Duplica:  map[string]int64{fullPath: mtime},
		Info:     info,
	}

	err := store.Insert(fi.ID, &fi)
	if err == badgerhold.ErrKeyExists {
		log.Infof("index: - add - Insert key=%s existed for %s, updating ..", fi.ID, fullPath)
		if err = store.FindOne(&fi, badgerhold.Where("ID").Eq(mf.Hash())); err == nil {
			mtime2, bExisted := fi.Duplica[fullPath]
			if bExisted == true && mtime != mtime2 {
				//TODO: choose a better one to update?
				log.Warnf("index: - add - Insert file=%s existed. time %v-> %v", fullPath, mtime, mtime2)
			}
			if bExisted == false || mtime < mtime2 {
				fi.Duplica[fullPath] = mf.ModTime().Unix()
				store.Update(fi.ID, &fi)
			}
		}
	} else if err != nil {
		log.Errorf("index: - add - Insert error %v %s", err, fi.Path)
	}
	log.Infof("index: - add - DONE %s %s %s %s, paths=%v", fi.ID, fi.Path, takenAt, takenAtSrc, fi.Duplica)
}
