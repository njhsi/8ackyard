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
	dbtx = nil
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
		var dbtx1 *sql.Tx
		var sInsert *sql.Stmt
		var sDelete *sql.Stmt

		var fcount int
		for fi := range chDb {
			if dbtx1 == nil {
				dbtx1, _ = db.Begin()
			}
			if f, ok := mapFiles[fi.Name]; ok {
				log.Warnf("index db: conflicted path=%v, updating in db with id=%v to id=%v", fi.Name, f.Id, fi.Id)
				fiOld := File8{}
				dbRow := dbtx1.QueryRow(sqlQuery, fi.Name, fi.Hostname)
				if err := dbRow.Scan(&fiOld.Id, &fiOld.Size, &fiOld.Hostname, &fiOld.TimeModified, &fiOld.TimeBorn, &fiOld.TimeBornSrc,
					&fiOld.MIMEType, &fiOld.MIMESubtype, &fiOld.Info); err == nil {
					fi.Info = fmt.Sprintf("\fi=%+v NOW=%v", fiOld, time.Now()) + fi.Info //checkpoint
				}
				if sDelete == nil {
					sDelete, _ = dbtx1.Prepare(sqlDelete)
				}
				if _, err := sDelete.Exec(fi.Name); err != nil {
					log.Warnf("index db: sDelete.Exec err=%v, fi=%v", err, fi)
				}
			}

			if sInsert == nil {
				sInsert, _ = dbtx1.Prepare(sqlInsert)
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
				dbtx1.Commit()
				dbtx1 = nil
			}
		}

		if dbtx1 != nil {
			dbtx1.Commit()
			dbtx1 = nil
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

	// BACKUP to destine
	if opt.BackupPath != "" {
		backup_start(opt, db)
	}

	ind.mutex.RLock()
	defer ind.mutex.RUnlock()
	runtime.GC()
	log.Infof("index: Start() finished.. mainworker canceld %v", mutex.MainWorker.Canceled())
	return done
}

func backup_start(opt IndexOptions, db *sql.DB) {
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
	dbtx.Commit()
	dbtx = nil
	log.Infof("index: backup starts, %v distinct files in db", len(ids))

	//collect of files back'd up and existed in db ?

	backupOpt := BackupOptions{
		OriginalsPath: opt.Path,
		BackupPath:    opt.BackupPath,
		CachePath:     opt.CachePath,
		NumWorkers:    opt.NumWorkers,
	}

	jobs := make(chan *BackupJob)
	chDb := make(chan *File8, 50)

	var wg sync.WaitGroup
	numWorkers := 3
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			BackupWorker(jobs)
			wg.Done()
		}()
	}

	//load the backup jobs
	sqlQueryFiles := `select name, hostname, size, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info from files where id=? and hostname=?`
	sqlQueryFilez := `select name, hostname, size, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info from filez where id=?` //existed backup
	sqlInsertFilez := `insert into filez(name, id, size, hostname, timemodified, timeborn, timebornsrc, mimetype, mimesubtype, info) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	sqlDeleteFilez := `delete from filez where id=?`
	var sInsertFilez, sDeleteFilez *sql.Stmt

	var bcount, jcount int
	var job *BackupJob
	bfm := NewBackupFsMutex()
	for bcount < len(ids) {
		if dbtx == nil {
			dbtx, _ = db.Begin()
		}

		if job == nil && jcount < len(ids) {
			id := ids[jcount]
			job = &BackupJob{
				Id:        id,
				BackupOpt: backupOpt,
				ChDB:      chDb,
				Bfm:       bfm,
			}
			rows, _ := dbtx.Query(sqlQueryFiles, id, opt.Hostname)
			for rows.Next() {
				fi := &File8{Id: id}
				if err := rows.Scan(&fi.Name, &fi.Hostname, &fi.Size, &fi.TimeModified, &fi.TimeBorn, &fi.TimeBornSrc,
					&fi.MIMEType, &fi.MIMESubtype, &fi.Info); err == nil {
					job.Files = append(job.Files, fi)
				}
			}

			f8 := &File8{Id: id} //back'd up
			row := dbtx.QueryRow(sqlQueryFilez, id)
			if err := row.Scan(&f8.Name, &f8.Hostname, &f8.Size, &f8.TimeModified, &f8.TimeBorn, &f8.TimeBornSrc,
				&f8.MIMEType, &f8.MIMESubtype, &f8.Info); err == nil {
				job.BackFile = f8
			} else {
				//				log.Warnf("index: Backup : query for job.BackFile(id=%v) failed - %v", id, err)
			}
		}

		var fb *File8
		if job != nil {
			select {
			case jobs <- job:
				log.Infof("backup: select sent job.id=%v, b=%v,j=%v", job.Id, bcount, jcount)
				jcount = jcount + 1
				job = nil
			case fb = <-chDb:
				log.Infof("backup: select got fb %+v, b=%v,j=%v", fb, bcount, jcount)
			}
		}

		if jcount == len(ids) && fb == nil {
			log.Infof("backup: select-no got fb %+v, b=%v,j=%v", fb, bcount, jcount)
			fb = <-chDb
		}
		if fb != nil {
			bcount = bcount + 1
			log.Infof("backup: got fb %+v, bcount=%v", fb, bcount)
			if fb.Size == 0 { // non-backup, just count it on
				continue
			}
			if fb.backup_ != nil {
				if sDeleteFilez == nil {
					sDeleteFilez, _ = dbtx.Prepare(sqlDeleteFilez)
				}
				if _, err := sDeleteFilez.Exec(fb.Id); err != nil {
					log.Warnf("backup db: sDelete.Exec err=%v, fi=%+v", err, fb)
				}
			}
			if sInsertFilez == nil {
				sInsertFilez, _ = dbtx.Prepare(sqlInsertFilez)

			}
			if _, err := sInsertFilez.Exec(fb.Name, fb.Id, fb.Size, fb.Hostname, fb.TimeModified, fb.TimeBorn, fb.TimeBornSrc,
				fb.MIMEType, fb.MIMESubtype, fb.Info); err != nil {
				log.Warnf("backup db: sInsert.Exec err=%v, fi=%v", err, fb)
			}

		}

		if bcount%100 == 0 {
			if err := dbtx.Commit(); err != nil {
				log.Fatalf("backup db: Commit failed %v", err)
			}
			dbtx = nil
			sDeleteFilez = nil
			sInsertFilez = nil
		}
	} //for

	if dbtx != nil { // must be out of for loop above, in case of fb=nil break
		dbtx.Commit()
	}

	//	ind.mutex.RLock()
	//	defer ind.mutex.RUnlock()

	close(jobs)
	wg.Wait()
	close(chDb)
}
