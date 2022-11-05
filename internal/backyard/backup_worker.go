package backyard

import (
	"os"
	"path"
	"time"

	"github.com/njhsi/8ackyard/pkg/fs"
)

type FileBacked struct {
	ID      uint64           //xxh3 hash of file content
	Size    uint64           // file size
	Path    string           //full path, regular file existed on device
	Duplica map[string]int64 //fullpath:modtime
	Type    string
	Info    string
}

type BackupOptions struct {
	OriginalsPath string
	BackupPath    string
	CachePath     string
	NumWorkers    int
	Rescan        bool
}

type BackupJob struct {
	BackupOpt BackupOptions
	File      *FileIndexed
	ChDB      chan *FileBacked
}

func BackupWorker(jobs <-chan BackupJob) {
	for job := range jobs {
		log.Infof("BackupWorker:                           mfs=%d", len(job.File.Path))
		if fb := mainBackup(job.File, job.BackupOpt); fb != nil {
			job.ChDB <- fb
		}
	}
}

func mainBackup(file *FileIndexed, opt BackupOptions) *FileBacked {
	timeLoc, _ := time.LoadLocation("Asia/Chongqing")

	baseName, fullPath, mType, hash := path.Base(file.Path), file.Path, file.Info, file.ID
	takenAt, takenAtSrc := time.Unix(file.TimeBorn, 0).In(timeLoc), file.TimeSrc
	mtime := fs.BirthTime(fullPath).Unix() //TODO: birthtime not works?

	backupTo := opt.BackupPath + "/" + mType + "/" + takenAt.Format("2006/01/02") + "/" + baseName
	backupTo = path.Clean(backupTo)
	for fs.FileExists(backupTo) && fs.XXHash3(backupTo) != hash {
		log.Warnf("backup: same name but diff hash: %s ->%s", backupTo, file.Path)
		backupTo = backupTo + "_" + fs.Uint64ToString(hash) + "_XXH3"
	}
	log.Infof("backup: STARTing file=%s, %s -> %s , %s, %v(%s)", baseName, fullPath, backupTo, hash, takenAt, takenAtSrc)

	fb := FileBacked{
		ID:      file.ID,
		Path:    backupTo,
		Size:    file.Size,
		Duplica: map[string]int64{backupTo: mtime},
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
	return &fb

}
