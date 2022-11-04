package backyard

import (
	"os"
	"path"
	"time"

	"github.com/njhsi/8ackyard/pkg/fs"
	"github.com/timshannon/badgerhold/v4"
)

type FileBacked struct {
	ID      string `badgerholdIndex:"ID"` //xxhash of file content
	Hash    string `badgerhold:"unique"`
	Size    int
	Path    string           //full path, regular file existed on device
	Duplica map[string]int64 //fullpath:modtime
	Type    string
	Info    string
}

type BackupOptions struct {
	Store         *badgerhold.Store
	OriginalsPath string
	BackupPath    string
	CachePath     string
	NumWorkers    int
	Rescan        bool
}

type BackupJob struct {
	BackupOpt BackupOptions
	Store     *badgerhold.Store
	File      *FileIndexed
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:                           mfs=%d", len(job.File.Path))
		mainBackup(job.File, job.Store, job.BackupOpt)
	}
}

func mainBackup(file *FileIndexed, store *badgerhold.Store, opt BackupOptions) {
	timeLoc, _ := time.LoadLocation("Asia/Chongqing")

	baseName, fullPath, hash, mType := path.Base(file.Path), file.Path, file.Hash, file.Info
	takenAt, takenAtSrc := time.Unix(file.TimeBorn, 0).In(timeLoc), file.TimeSrc
	mtime := fs.BirthTime(fullPath).Unix() //TODO: birthtime not works?

	backupTo := opt.BackupPath + "/" + mType + "/" + takenAt.Format("2006/01/02") + "/" + baseName
	backupTo = path.Clean(backupTo)
	for fs.FileExists(backupTo) && fs.HashXXH3_64(backupTo) != hash {
		log.Warnf("backup: same name but diff hash: %s ->%s", backupTo, file.Path)
		backupTo = backupTo + "_" + hash + "XXH3"
	}
	log.Infof("backup: STARTing file=%s, %s -> %s , %s, %v(%s)", baseName, fullPath, backupTo, hash, takenAt, takenAtSrc)

	fb := FileBacked{
		ID:      hash,
		Path:    backupTo,
		Size:    file.Size,
		Hash:    hash,
		Duplica: map[string]int64{backupTo: mtime},
	}

	err := store.Insert(fb.ID, &fb)
	if err == badgerhold.ErrKeyExists {
		log.Infof("backup: Insert failed, key=%s existed for mf=%s, need to update..", fb.ID, fullPath)
		if err = store.FindOne(&fb, badgerhold.Where("ID").Eq(hash)); err == nil {
			mtime2, bExisted := fb.Duplica[backupTo]
			if bExisted == true && mtime != mtime2 {
				//TODO: choose a better one to update?
				log.Warnf("backup: Update? key=%s and mf=%s existed, but time %v->%v", fb.ID, fullPath, mtime, mtime2)
			}
			if bExisted == false || mtime != mtime2 {
				log.Infof("backup: Update key=%s existed,mf=%s, time same %s, fiName existed %s", fb.ID, bExisted, mtime == mtime2, fs.FileExists(fb.Path))
				if fs.FileExists(fb.Path) == false {
					fb.Path = backupTo
				}
				fb.Duplica[backupTo] = file.TimeBorn
				store.Update(fb.ID, &fb)
			}
		}
	} else if err != nil {
		log.Errorf("backup: Insert error %v %s", err, fb.Path)
	}

	f_pathx := func(apath string) string {
		// copy realted files like .AAE
		related := apath
		for i := len(apath) - 1; i >= 0 && apath[i] != '/'; i-- {
			if apath[i] == '.' {
				related = apath[:i]
				break
			}
		}
		return related
	}
	related := f_pathx(fullPath) + ".AAE"
	related_ := f_pathx(fb.Path) + ".AAE"
	if fs.FileExists(related) == true && fs.FileExists(related_) == false {
		log.Infof("backup: DO COPY (realted) for mf=%s, -> %s , backupTo=%s", related, related_, backupTo)
		fs.CopyWithStat(related, related_)
	}

	if fs.FileExists(fb.Path) == false { // copy it
		log.Infof("backup: DO COPY for mf=%s, -> %s , backupTo=%s", fullPath, fb.Path, backupTo)
		fs.CopyWithStat(fullPath, fb.Path)
	} else {
		if backupTo != fb.Path {
			//link it
			log.Infof("backup: DO LINK for mf=%s,(backupTo->fiName) %s -> %s", fullPath, backupTo, fb.Path)
			os.Symlink(fb.Path, backupTo)

		} else { //TODO: is it really the file we want?
			log.Infof("backup: DO NOTHING for mf=%s, already existed in backup,fi=%s, backupTo=%s", fullPath, fb.Path, backupTo)
		}
	}

	log.Infof("backup: DONE mf=%s key=%s fiName=%s backupTo=%s %s %s, map=%v", fullPath, fb.ID, fb.Path, backupTo, takenAt, takenAtSrc, fb.Duplica)

}
