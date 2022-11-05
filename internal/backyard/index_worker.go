package backyard

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/barasher/go-exiftool"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/sanitize"
)

type FileIndexed struct {
	ID       uint64 //xxh3 of file content
	Path     string //full path
	TimeBorn int64  //unix seconds
	TimeSrc  string //meta, name, auto
	Size     uint64
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
	ChDB     chan *FileIndexed
}

func IndexWorker(jobs <-chan IndexJob, et *exiftool.Exiftool) {
	for job := range jobs {
		log.Infof("IndexWorker:                           fileName=%s", job.FileName)
		if _, mf := mainIndex(job.FileName, job.Ind, job.IndexOpt, et); mf != nil {
			add(job.ChDB, mf)
		}
	}
}

func mainIndex(fileName string, ind *Index, opt IndexOptions, exifTool *exiftool.Exiftool) (error, *MediaFile) {
	log.Infof("mainIndex: entering, %v , %v", fileName, exifTool)
	//	return nil, nil
	f, err := NewMediaFile(fileName)
	if err != nil {
		log.Errorf("index_main: found no  mediafile for %s", sanitize.Log(fileName))
		return err, nil
	}

	sizeLimit := config.OriginalsLimit()
	log.Infof("mainIndex: entering 1, %v , %v", fileName, exifTool)

	// Enforce file size limit for originals.
	if sizeLimit > 0 && f.FileSize() > sizeLimit {
		log.Errorf("index_main: file size xx%s (%d/%dM)", f.FileName(), f.FileSize()/(1024*1024), sizeLimit/(1024*1024))
		return nil, nil
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

	log.Infof("index_main: DONE mf=%s(%), %s %s.%s", f.FileName(), f.FileType(), f.Hash(), takenAt, src)

	return nil, f
}

func add(chDb chan *FileIndexed, mf *MediaFile) {
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

	fi := &FileIndexed{
		ID:       mf.Hash(),
		Path:     mf.FileName(),
		TimeBorn: takenAt.Unix(),
		TimeSrc:  takenAtSrc,
		Size:     uint64(mf.FileSize()),
		Format:   string(mf.FileType()),
		Mime:     mf.MimeType(),
		Duplica:  map[string]int64{fullPath: mtime},
		Info:     info,
	}

	chDb <- fi

	log.Infof("index: - add - DONE %s %s %s %s, paths=%v", fi.ID, fi.Path, takenAt, takenAtSrc, fi.Duplica)
}
