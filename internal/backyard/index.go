package backyard

import (
	"database/sql"
	"errors"
	iofs "io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	"github.com/barasher/go-exiftool"
	"github.com/karrick/godirwalk"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/mutex"
	"github.com/njhsi/8ackyard/pkg/fs"
)

// Index represents an indexer that indexes files in the originals directory.
type Index struct {
	//	convert *Convert
	mutex sync.RWMutex // storeIndex

}

// NewIndex returns a new indexer and expects its dependencies as arguments.
func NewIndex() *Index {

	i := &Index{}

	return i
}

// Cancel stops the current indexing operation.
func (ind *Index) Cancel() {
	mutex.MainWorker.Cancel()
}

func (ind *Index) initStoreIndex(storePath string) error {

	return nil
}

func (ind *Index) initStoreBacup(storePath string) error {
	return nil
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

	dbName := opt.CachePath + "/indexed.db"
	dbExisted := true
	if _, err := os.Stat(dbName); errors.Is(err, iofs.ErrNotExist) {
		dbExisted = false
	}
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Errorf("db failed: Open %v", err)
		return done
	}
	defer db.Close()
	if !dbExisted {
		sqlStmt := `
               create table file (idxxh3 integer not null primary key, size integer not null, birth integer, type text, name text);
               create table path (path text not null primary key, idxxh3 integer not null, mtime integer);
               delete from file;
               delete from path;
               `
		_, err = db.Exec(sqlStmt)
		if err != nil {
			log.Errorf("db failed: Exec %q: %s", err, sqlStmt)
			return done
		}
	}

	if err := mutex.MainWorker.Start(); err != nil {
		log.Errorf("index: %s", err.Error())
		return done
	}

	defer mutex.MainWorker.Stop()

	jobs := make(chan IndexJob)
	chDb := make(chan *FileIndexed, 50)

	// Start a fixed number of goroutines to index files.
	var wg sync.WaitGroup
	var numWorkers = opt.NumWorkers
	if numWorkers == 0 {
		numWorkers = 3
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
	chDbWait := make(chan bool)
	go func() {
		//db
		var fcount int
		var dbtx *sql.Tx
		var stmt *sql.Stmt

		for fi := range chDb {
			log.Infof("index db:%v, fi:%v, fcount:%v", db, fi, fcount)

			if fcount == 0 {
				dbtx, _ = db.Begin()
				stmt, _ = dbtx.Prepare("insert into path(path, idxxh3, mtime) values(?, ?, ?)")
			}
			fcount = fcount + 1
			if _, errx := stmt.Exec(fi.Path, int64(fi.ID), fi.TimeBorn); errx != nil {
				log.Warn(errx)
			}
			if fcount == 100 {
				fcount = 0
				dbtx.Commit()
				stmt.Close()
			}
		}

		if fcount > 0 {
			dbtx.Commit()
			stmt.Close()
		}
		log.Infof("index db: exit %v", db)
		chDbWait <- true
	}()

	config.CacheDir = opt.CachePath
	config.FileRoot = opt.Path

	filesIndexed := 0
	ignore := fs.NewIgnoreList(fs.IgnoreFile, true, false)

	if err := ignore.Dir(originalsPath); err != nil {
		log.Infof("index: %s", err)
	}

	ignore.Log = func(fileName string) {
		log.Infof(`index: ignored "%s"`, fs.RelName(fileName, originalsPath))
	}

	err = godirwalk.Walk(optionsPath, &godirwalk.Options{
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

			jobs <- IndexJob{
				FileName: fileName,
				IndexOpt: opt,
				Ind:      ind,
				ChDB:     chDb,
			}

			return nil
		},
		Unsorted:            false,
		FollowSymbolicLinks: true,
	})

	close(jobs)

	wg.Wait()
	close(chDb)
	<-chDbWait
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
		if err := ind.initStoreBacup(opt.CachePath); err != nil {
			log.Errorf("index: ind.initStoreBacup failed %s", err)
			return done
		}

		backupOpt := BackupOptions{
			OriginalsPath: opt.Path,
			BackupPath:    opt.BackupPath,
			CachePath:     opt.CachePath,
			NumWorkers:    opt.NumWorkers,
			//			Store:         ind.storeBackup,
		}
		log.Infof(backupOpt.BackupPath)

		jobs2 := make(chan BackupJob)
		chDb2 := make(chan *FileBacked, 50)

		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func() {
				BackupWorker(jobs2)
				wg.Done()
			}()
		}
		go func() {
			//db
			var fcount int
			var dbtx *sql.Tx
			var stmt *sql.Stmt
			for fb := range chDb2 {
				if fcount == 0 {
					dbtx, _ = db.Begin()
					stmt, _ = dbtx.Prepare("insert into file(idxxh3,  size, birth, type, name) values(?, ?, ?, ?, ?, ?)")
				}
				fcount = fcount + 1

				if _, errx := stmt.Exec(fb); errx != nil {
					log.Fatal(errx)
				}
				if fcount == 100 {
					fcount = 0
					dbtx.Commit()
					stmt.Close()
				}
			}
			if fcount > 0 {
				dbtx.Commit()
				stmt.Close()
			}
			chDbWait <- true
		}()

		ind.mutex.RLock()
		defer ind.mutex.RUnlock()

		close(jobs2)
		wg.Wait()
		close(chDb2)
		<-chDbWait

		runtime.GC()
	}

	log.Infof("index: Start() finished.. mainworker canceld %v", mutex.MainWorker.Canceled())
	return done
}
