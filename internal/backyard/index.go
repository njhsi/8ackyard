package backyard

import (
	"errors"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/barasher/go-exiftool"
	"github.com/karrick/godirwalk"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/mutex"
	"github.com/njhsi/8ackyard/pkg/fs"
)

// Index represents an indexer that indexes files in the originals directory.
type Index struct {
	//	convert *Convert
	files  *Files
	photos *Photos
}

// NewIndex returns a new indexer and expects its dependencies as arguments.
func NewIndex(files *Files, photos *Photos) *Index {

	i := &Index{
		files:  files,
		photos: photos,
	}

	return i
}

// Cancel stops the current indexing operation.
func (ind *Index) Cancel() {
	mutex.MainWorker.Cancel()
}

// Start indexes media files in the "originals" folder.
func (ind *Index) Start(opt IndexOptions) fs.Done {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("index: %s (panic)\nstack: %s", r, debug.Stack())
		}
	}()

	done := make(fs.Done)

	originalsPath := opt.Path
	optionsPath := opt.Path

	if !fs.PathExists(optionsPath) {
		log.Errorf("index: %s does not exist", optionsPath)
		return done
	}

	if err := mutex.MainWorker.Start(); err != nil {
		log.Errorf("index: %s", err.Error())
		return done
	}

	defer mutex.MainWorker.Stop()

	jobs := make(chan IndexJob)

	// Start a fixed number of goroutines to index files.
	var wg sync.WaitGroup
	var numWorkers = opt.NumWorkers
	if numWorkers == 0 {
		numWorkers = 4
	}
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			et, err := exiftool.NewExiftool()
			if err != nil {
				et = nil
				log.Warnf("index: error when intializing exiftool: %v\n", err)
			} else {
				defer et.Close()
			}
			IndexWorker(jobs, et) // HLc
			wg.Done()
		}()

	}

	if err := ind.files.Init(opt.CachePath); err != nil {
		log.Errorf("index: %s", err)
	}
	config.CacheDir = opt.CachePath

	defer ind.files.Done()

	filesIndexed := 0
	ignore := fs.NewIgnoreList(fs.IgnoreFile, true, false)

	if err := ignore.Dir(originalsPath); err != nil {
		log.Infof("index: %s", err)
	}

	ignore.Log = func(fileName string) {
		log.Infof(`index: ignored "%s"`, fs.RelName(fileName, originalsPath))
	}

	err := godirwalk.Walk(optionsPath, &godirwalk.Options{
		ErrorCallback: func(fileName string, err error) godirwalk.ErrorAction {
			log.Errorf("index: %s", strings.Replace(err.Error(), originalsPath, "", 1))
			return godirwalk.SkipNode
		},
		Callback: func(fileName string, info *godirwalk.Dirent) error {
			if mutex.MainWorker.Canceled() {
				return errors.New("indexing canceled")
			}

			isDir := info.IsDir()
			isSymlink := info.IsSymlink()
			relName := fs.RelName(fileName, originalsPath)

			if skip, result := fs.SkipWalk(fileName, isDir, isSymlink, done, ignore); skip {
				if (isSymlink || isDir) && result != filepath.SkipDir {
					log.Infof("index: added folder /%s", fileName)
				}

				if isDir {
					log.Infof("index.folder filePath /%s", relName)
				}

				return result
			}

			done[fileName] = fs.Found

			if !fs.IsMedia(fileName) {
				log.Infof("index: not media file /%s", fileName)
				return nil
			}

			mf, err := NewMediaFile(fileName)

			if err != nil {
				log.Error(err)
				return nil
			}

			if mf.FileSize() == 0 {
				log.Infof("index: skipped empty file %s", mf.BaseName())
				return nil
			}

			if ind.files.Indexed(relName, "/", mf.modTime, opt.Rescan) {
				return nil
			}

			done[fileName] = fs.Processed

			jobs <- IndexJob{
				FileName: mf.FileName(),
				IndexOpt: opt,
				Ind:      ind,
			}

			return nil
		},
		Unsorted:            false,
		FollowSymbolicLinks: true,
	})

	close(jobs)
	wg.Wait()
	log.Infof("index completed .. wg.Wait done")

	if err != nil {
		log.Error(err.Error())
	}

	if filesIndexed > 0 {

		log.Infof("index.updating /%d", filesIndexed)

		// Update precalculated photo and file counts.
	} else {
		log.Infof("index: found no new or modified files")
	}

	runtime.GC()

	// BACKUP to destine
	if opt.BackupPath != "" {
		jobs2 := make(chan BackupJob)
		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func() {
				BackupWorker(jobs2)
				wg.Done()
			}()
		}
		for kFileSize, vMfiles := range ind.files.mfiles {
			log.Infof("backup: size=%d, %d mfs", kFileSize, len(vMfiles))
			jobs2 <- BackupJob{
				IndexOpt: opt,
				Ind:      ind,
				MFiles:   vMfiles,
			}
		}
		close(jobs2)
		wg.Wait()
		runtime.GC()
	}

	return done
}
