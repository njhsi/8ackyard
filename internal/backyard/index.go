package backyard

import (
	"errors"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

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

func (ind *Index) originalsPath() string {
	return config.OriginalsPath()
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

	originalsPath := ind.originalsPath()
	optionsPath := filepath.Join(originalsPath, opt.Path)

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
	var numWorkers = 5
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			IndexWorker(jobs) // HLc
			wg.Done()
		}()
	}

	if err := ind.files.Init(); err != nil {
		log.Errorf("index: %s", err)
	}

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

	// copy to destine
	for kFileSize, vMfiles := range ind.files.mfiles {
		log.Infof("backup: size=%d, %d mfs", kFileSize, len(vMfiles))
		sumMfiles := map[string]MediaFiles{}
		for _, mf := range vMfiles {
			sumMfiles[mf.Md5sum()] = append(sumMfiles[mf.Md5sum()], mf)
		}
		for _, mfs := range sumMfiles { //TODO: job the vMfiles of each md5sum
			var mfBest *MediaFile = nil
			for _, mf := range mfs {
				//TODO: save dups info into a txt file, in case ..
				takenAt, src := mf.TakenAt()
				log.Infof("backup: mf=%s md5=%s takenat=%s src=%s", mf.FileName(), mf.Md5sum(), takenAt, src)
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
				folder := takenAt.Format("2006/01/02")
				log.Infof("backup: DO!! %s=>%s %s %s", mfBest.FileName(), folder, takenAt, src)
				mfBest.Copy("/tmp/Backup/" + folder + "/" + mfBest.BaseName())
			}
		}
	}

	return done
}
