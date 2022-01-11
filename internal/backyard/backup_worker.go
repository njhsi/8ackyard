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
	Name    string
	PathMap map[string]int64 //fullpath:modtime
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
	Ind       *Index
	MFiles    MediaFiles
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:                           mfs=%d", len(job.MFiles))
		new_backup_main(job.MFiles, job.Ind, job.BackupOpt)
	}
}

func new_backup_main(mFiles MediaFiles, ind *Index, opt BackupOptions) {
	mapMfiles := map[string]MediaFiles{}
	store := ind.storeBackup
	for _, mf := range mFiles {
		mapMfiles[mf.Hash()] = append(mapMfiles[mf.Hash()], mf)
		log.Infof("backup: mf=%s size=%d  sha1=%s", mf.FileName(), mf.FileSize(), mf.Hash())
	}

	// mfiles are of the same size, so that no need to consider store concurrency lock for updating ..
	timeLoc, _ := time.LoadLocation("Asia/Chongqing")
	for _, mfs := range mapMfiles {
		for _, mf := range mfs {
			fullPath, mtime, hash := mf.FileName(), mf.modTime.Unix(), mf.Hash()
			takenAt, takenAtSrc := mf.TakenAt()
			takenAt = takenAt.In(timeLoc)
			mType := "unknown"
			if mf.IsPhoto() {
				mType = "photo"
			} else if mf.IsVideo() {
				mType = "video"
			} else if mf.IsAudio() {
				mType = "audio"
			}

			backupTo := opt.BackupPath + "/" + mType + "/" + takenAt.Format("2006/01/02") + "/" + mf.BaseName()
			backupTo = path.Clean(backupTo)
			for fs.FileExists(backupTo) && fs.Hash(backupTo) != hash {
				log.Warnf("backup: same name but diff hash: %s ->%s", backupTo, mf.FileName())
				backupTo = backupTo + "_" + hash
			}
			fi := FileBacked{
				ID:      hash,
				Name:    backupTo,
				Size:    int(mf.FileSize()),
				Hash:    hash,
				PathMap: map[string]int64{backupTo: mtime},
			}

			err := store.Insert(fi.ID, &fi)
			if err == badgerhold.ErrKeyExists {
				log.Infof("backup: Insert failed, key=%s existed for mf=%s, need to update..", fi.ID, fullPath)
				if err = store.FindOne(&fi, badgerhold.Where("ID").Eq(hash)); err == nil {
					mtime2, bExisted := fi.PathMap[backupTo]
					if bExisted == true && mtime != mtime2 {
						//TODO: choose a better one to update?
						log.Warnf("backup: Update? key=%s and mf=%s existed, but time %v->%v", fi.ID, fullPath, mtime, mtime2)
					}
					if bExisted == false || mtime != mtime2 {
						log.Infof("backup: Update key=%s existed,mf=%s, time same %s, fiName existed %s", fi.ID, bExisted, mtime == mtime2, fs.FileExists(fi.Name))
						if fs.FileExists(fi.Name) == false {
							fi.Name = backupTo
						}
						fi.PathMap[backupTo] = mf.ModTime().Unix()
						store.Update(fi.ID, &fi)
					}
				}
			} else if err != nil {
				log.Errorf("backup: Insert error %v %s", err, fi.Name)
			}

			if fs.FileExists(fi.Name) == false {
				// copy it
				log.Infof("backup: DO COPY for mf=%s, -> %s , backupTo=%s", fullPath, fi.Name, backupTo)
				mf.Copy(fi.Name)
			} else if fs.FileExists(fi.Name) == true && backupTo != fi.Name {
				//link it
				log.Infof("backup: DO LINK for mf=%s, %s -> %s", fullPath, fi.Name, backupTo)
				os.Symlink(fi.Name, backupTo)
			} else {
				log.Infof("backup: Do nothing, mf=%s existed in backup,fi=%s, backupTo=%s", fullPath, fi.Name, backupTo)
			}

			log.Infof("backup: DONE mf=%s key=%s fiName=%s backupTo=%s %s %s, map=%v", fullPath, fi.ID, fi.Name, backupTo, takenAt, takenAtSrc, fi.PathMap)
		}
	}

}

func backup_main(mFiles MediaFiles, ind *Index, opt BackupOptions) (result IndexResult) {
	sumMfiles := map[string]MediaFiles{}
	if len(mFiles) == 1 { // no need to do hash
		sumMfiles[""] = mFiles
		log.Infof("backup: mf=%s size=%d  sha1=%s", mFiles[0].FileName(), mFiles[0].FileSize(), mFiles[0].Hash())
	} else {
		for _, mf := range mFiles {
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
			mType := "unknown"
			if mfBest.IsPhoto() {
				mType = "photo"
			} else if mfBest.IsVideo() {
				mType = "video"
			} else if mfBest.IsAudio() {
				mType = "audio"
			}

			backupTo := opt.BackupPath + "/" + mType + "/" + takenAt.Format("2006/01/02") + "/" + mfBest.BaseName()
			for fs.FileExists(backupTo) && fs.Hash(backupTo) != mfBest.Hash() {
				log.Warnf("backup: same name but diff hash: %s ->%s", backupTo, mfBest.FileName())
				backupTo = backupTo + "_" + mfBest.Hash()
			}
			log.Infof("backup: DO!!! [ %s => %s ], %s %s", mfBest.FileName(), backupTo, takenAt, src)
			mfBest.Copy(backupTo)
		}
	}

	result.Status = IndexAdded
	return result
}
