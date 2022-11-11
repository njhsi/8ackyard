package backyard

import (
	"database/sql"
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/barasher/go-exiftool"
	"github.com/karrick/godirwalk"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/mutex"
	"github.com/photoprism/photoprism/pkg/fs"
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

	if len(opt.Hostname) == 0 {
		opt.Hostname, _ = os.Hostname()
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
		// id: xxhash h3 64bit. INT rather than INTEGER of sqlite, constraints non-auto-incremental as primary key needs.
		sqlStmt := `
               create table filez (id int not null, name text not null, hostname text,
                                   size integer not null, timemodified integer, timeborn integer, timebornsrc text,
                                   mimetype text, mimesubtype text, info text,
                                   primary key(id));
               create table files (name text not null, hostname text not null, id int not null,
                                   size integer not null, timemodified integer, timeborn integer, timebornsrc text,
                                   mimetype text, mimesubtype text, info text,
                                   primary key(name, hostname));
               delete from filez;
               delete from files;
               `
		_, err = db.Exec(sqlStmt)
		if err != nil {
			log.Fatalf("db failed: Exec %q: %s", err, sqlStmt)
			return done
		}
	}

	mapFiles := make(map[string]*File8) //TODO: instead, query db when neccessary
	dbtx, err := db.Begin()
	dbrows, err := dbtx.Query("select name,size,timemodified,id from files where hostname=?", opt.Hostname)
	for dbrows.Next() {
		fi := &File8{}
		if err := dbrows.Scan(&fi.Name, &fi.Size, &fi.TimeModified, &fi.Id); err != nil {
			log.Fatalf("dbrows.Scan :%v", err)
		}
		mapFiles[fi.Name] = fi
	}
	dbtx.Commit()
	dbrows.Close()
	log.Infof("index: loaded %v files of host[%v] in db", len(mapFiles), opt.Hostname)

	if err := mutex.MainWorker.Start(); err != nil {
		log.Errorf("index: %s", err.Error())
		return done
	}
	defer mutex.MainWorker.Stop()

	jobs := make(chan IndexJob)
	chDb := make(chan *File8, 50)

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
		sqlQuery := `select id, size, hostname, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info from files where name=? and hostname=?`
		sqlInsert := `insert into files(name, id, size, hostname, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		sqlDelete := `delete from files where name=?`
		var dbtx *sql.Tx
		var sInsert *sql.Stmt
		var sDelete *sql.Stmt

		var fcount int
		for fi := range chDb {
			if dbtx == nil {
				dbtx, _ = db.Begin()
			}
			if f, ok := mapFiles[fi.Name]; ok {
				log.Warnf("index db: conflicted path=%v, updating in db with id=%v to id=%v", fi.Name, f.Id, fi.Id)
				fiOld := File8{}
				dbRow := dbtx.QueryRow(sqlQuery, fi.Name, fi.Hostname)
				if err := dbRow.Scan(&fiOld.Id, &fiOld.Size, &fiOld.Hostname, &fiOld.TimeModified, &fiOld.TimeBorn, &fiOld.TimeBornSrc,
					&fiOld.MIMEType, &fiOld.MIMESubtype, &fiOld.Info); err == nil {
					fi.Info = fmt.Sprintf("\fi=%+v NOW=%v", fiOld, time.Now()) + fi.Info //checkpoint
				}
				if sDelete == nil {
					sDelete, _ = dbtx.Prepare(sqlDelete)
				}
				if _, err := sDelete.Exec(fi.Name); err != nil {
					log.Warnf("index db: sDelete.Exec err=%v, fi=%v", err, fi)
				}
			}

			if sInsert == nil {
				sInsert, _ = dbtx.Prepare(sqlInsert)
			}
			if _, err := sInsert.Exec(fi.Name, fi.Id, fi.Size, fi.Hostname,
				fi.TimeModified, fi.TimeBorn, fi.TimeBornSrc,
				fi.MIMEType, fi.MIMESubtype, fi.Info); err != nil {
				log.Warnf("index db: sInsert.Exec err=%v, fi=%v", err, fi)
			}

			fcount = fcount + 1
			if fcount%100 == 0 {
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

		if dbtx != nil {
			dbtx.Commit()
			dbtx = nil
		}
		log.Infof("index db: exit fcount=%v", fcount)
		chDbWait <- true
	}()

	config.CacheDir = opt.CachePath
	config.FileRoot = opt.Path

	filesIndexed := 0
	ignore := fs.NewIgnoreList(fs.IgnoreFile, false, false) //!! do not ignore hidden files

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
			skip, result := fs.SkipWalk(fileName, isDir, isSymlink, done, ignore)

			log.Infof("index: Walk got a file(skip=%v) - %v", skip, fileName)
			if skip {
				if (isSymlink || isDir) && result != filepath.SkipDir {
					log.Infof("index: added folder /%s", fileName)
				}

				if isDir {
					log.Infof("index.folder filePath /%s", relName)
				}

				return result
			}

			done[fileName] = fs.Found

			if fi, ok := mapFiles[fileName]; ok == true {
				if err, mtime, size := fileStat(fileName); err == nil {
					mtime_ts := mtime.Unix()
					if fi.Size == size && fi.TimeModified == mtime_ts { //TODO: strict option to check ID
						done[fileName] = fs.Processed
						log.Infof("index: Walk - file=[%v] with id=[%v] was in db, not processing..", fileName, fi.Id)
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
		//collect of distinct files to backup
		ids := make([]int64, 0)
		dbtx, _ := db.Begin()
		dbrows, _ := dbtx.Query("select distinct id from files where hostname=?", opt.Hostname)
		for dbrows.Next() {
			var id int64
			if err := dbrows.Scan(&id); err == nil {
				ids = append(ids, id)
			}
		}
		log.Infof("index: backup starts, %v distinct files in db", len(ids))

		//collect of files back'd up and existed in db ?

		backupOpt := BackupOptions{
			OriginalsPath: opt.Path,
			BackupPath:    opt.BackupPath,
			CachePath:     opt.CachePath,
			NumWorkers:    opt.NumWorkers,
		}

		jobs2 := make(chan BackupJob)
		chDb2 := make(chan *File8, 50)

		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func() {
				BackupWorker(jobs2)
				wg.Done()
			}()
		}
		go func() {
			//db
			sqlInsert := `insert into filez(name, id, size, hostname, timemodified, timeborn, timebornsrc,
                                       mimetype, mimesubtype, info) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
			sqlDelete := `delete from filez where id=?`
			var dbtx *sql.Tx
			var sInsert *sql.Stmt
			var sDelete *sql.Stmt

			var fcount int
			for fb := range chDb2 {
				log.Infof("db backup: f_count=%v, name=%v          \n  save=%+v, \n  ori=%+v", fcount, fb.Name, fb, fb.backup_)
				if len(fb.Name) == 0 {
					log.Warnf("db backup: empty name that not back'd up onto disk, fb=%+v", fb)
				}

				if dbtx == nil {
					dbtx, _ = db.Begin()
				}

				if fb.backup_ != nil {
					if sDelete == nil {
						sDelete, _ = dbtx.Prepare(sqlDelete)
					}
					if _, err := sDelete.Exec(fb.Id); err != nil {
						log.Warnf("backup db: sDelete.Exec err=%v, fi=%+v", err, fb)
					}
				}

				if sInsert == nil {
					sInsert, _ = dbtx.Prepare(sqlInsert)
				}
				if _, err := sInsert.Exec(fb.Name, fb.Id, fb.Size, fb.Hostname, fb.TimeModified, fb.TimeBorn, fb.TimeBornSrc,
					fb.MIMEType, fb.MIMESubtype, fb.Info); err != nil {
					log.Fatal(err)
				}

				fcount = fcount + 1
				if fcount%100 == 0 {
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

			if dbtx != nil {
				dbtx.Commit()
				dbtx = nil
			}

			log.Infof("backup db: exit fcount=%v", fcount)
			chDbWait <- true
		}()

		//load the backup jobs
		queryIndexed := `select name, hostname, size, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info from files where id=? and hostname=?`
		queryBacked := `select name, hostname, size, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info from filez where id=?` //existed backup
		for _, id := range ids {
			job := BackupJob{
				Id:        id,
				BackupOpt: backupOpt,
				ChDB:      chDb2,
			}
			rows, _ := dbtx.Query(queryIndexed, id, opt.Hostname)
			for rows.Next() {
				fi := &File8{Id: id}
				if err := rows.Scan(&fi.Name, &fi.Hostname, &fi.Size, &fi.TimeModified, &fi.TimeBorn, &fi.TimeBornSrc,
					&fi.MIMEType, &fi.MIMESubtype, &fi.Info); err == nil {
					job.Files = append(job.Files, fi)
				}
			}

			fb := &File8{Id: id} //back'd up
			row := dbtx.QueryRow(queryBacked, id)
			if err := row.Scan(&fb.Name, &fb.Hostname, &fb.Size, &fb.TimeModified, &fb.TimeBorn, &fb.TimeBornSrc,
				&fb.MIMEType, &fb.MIMESubtype, &fb.Info); err == nil {
				job.BackFile = fb
			}

			jobs2 <- job
		}
		dbtx.Commit()

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
