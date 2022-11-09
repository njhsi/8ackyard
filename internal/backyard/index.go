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
		// id: xxhash h3 64bit
		sqlStmt := `
               create table filez (id integer not null primary key, size integer not null, birth integer, type text, name text);
               create table files (path text not null primary key, id integer not null, size integer not null, hostname text, timemodified integer, timeborn integer, timebornsrc text, mimetype text, mimesubtype text, info text);
               delete from filez;
               delete from files;
               `
		_, err = db.Exec(sqlStmt)
		if err != nil {
			log.Fatalf("db failed: Exec %q: %s", err, sqlStmt)
			return done
		}
	}

	type FileInDB struct {
		Size  int64
		Mtime int64
		Id    int64
	}
	mapFiles := make(map[string]FileInDB)
	dbtx, err := db.Begin()
	dbrows, err := dbtx.Query("select path,size,timemodified,id from files;")
	for dbrows.Next() {
		fidPath, fid := "", FileInDB{}
		if err := dbrows.Scan(&fidPath, &fid.Size, &fid.Mtime, &fid.Id); err != nil {
			log.Fatalf("dbrows.Scan :%v", err)
		}
		mapFiles[fidPath] = fid
	}
	dbtx.Commit()
	dbrows.Close()
	log.Infof("index: loaded %v files in db", len(mapFiles))

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
	go func() { //db
		sqlInsert := `insert into files(path, id, size, hostname, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		sqlDelete := `delete from files where path=?`
		var dbtx *sql.Tx
		var sInsert *sql.Stmt
		var sDelete *sql.Stmt

		var fcount int
		for fi := range chDb {
			fcount = fcount + 1

			if dbtx == nil {
				dbtx, _ = db.Begin()
			}
			if fid, ok := mapFiles[fi.Path]; ok {
				log.Warnf("index db: conflicted path=%v, updating in db with id=%v to id=%v", fi.Path, fid.Id, fi.Id)
				if sDelete == nil {
					sDelete, _ = dbtx.Prepare(sqlDelete)
				}
				if _, err := sDelete.Exec(fi.Path); err != nil {
					log.Warnf("index db: sDelete.Exec err=%v, fi=%v", err, fi)
				}
			}

			if sInsert == nil {
				sInsert, _ = dbtx.Prepare(sqlInsert)
			}
			if _, err := sInsert.Exec(fi.Path, int64(fi.Id), fi.Size, fi.Hostname,
				fi.Mtime.Unix(), fi.TimeBorn.Unix(), fi.TimeBornSrc,
				fi.MIMEType, fi.MIMESubtype, fi.Info); err != nil {
				log.Warnf("index db: sInsert.Exec err=%v, fi=%v", err, fi)
			}

			if fcount == 100 {
				fcount = 0
				if sDelete != nil {
					sDelete.Close()
					sDelete = nil
				}
				if sInsert != nil {
					sInsert.Close()
					sInsert = nil
				}
				dbtx.Commit()
				dbtx = nil
			}
		}

		if fcount > 0 {
			dbtx.Commit()
		}
		log.Infof("index db: exit fcount=%v", fcount)
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
			log.Errorf("index: Walk an error=%s, @%v ", err, strings.Replace(err.Error(), originalsPath, "", 1))
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

				log.Infof("index: SkipWalk result=%v", result)
				return result
			}

			done[fileName] = fs.Found
			log.Infof("index: Walk got file - %v", fileName)
			if fid, ok := mapFiles[fileName]; ok == true {
				if err, mtime, size := fileStat(fileName); err == nil {
					mtime_ts := mtime.Unix()
					if fid.Size == size && fid.Mtime == mtime_ts { //TODO: strict option to check ID
						done[fileName] = fs.Processed
						log.Infof("index: Walk - file=[%v] with id=[%v] was in db, not processing..", fileName, fid.Id)
					}
				}
			}
			if done[fileName] != fs.Processed {
				jobs <- IndexJob{
					FileName: fileName,
					IndexOpt: opt,
					Ind:      ind,
					ChDB:     chDb,
				}
			}
			done[fileName] = fs.Processed

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
