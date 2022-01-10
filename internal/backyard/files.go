package backyard

import (
	"path"
	"sync"
	"time"

	"github.com/timshannon/badgerhold/v4"
)

type FileMap map[string]int64
type MediafilesMap map[int64]MediaFiles //filesize:MFs

// Files represents a list of already indexed file names and their unix modification timestamps.
type Files struct {
	count  int
	files  FileMap
	mfiles MediafilesMap
	mutex  sync.RWMutex
	store  *badgerhold.Store
}

type FileInStore struct {
	ID      string `badgerholdIndex:"ID"` //xxhash of file content
	Hash    string `badgerhold:"unique"`
	Size    int
	Name    string
	PathMap map[string]int64 //fullpath:modtime
	Type    string
	Info    string
}

// NewFiles returns a new Files instance.
func NewFiles() *Files {
	m := &Files{
		files:  make(FileMap),
		mfiles: make(MediafilesMap),
	}

	return m
}

// Init fetches the list from the database once.
func (m *Files) Init(storePath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.files) > 0 {
		m.count = len(m.files)
		return nil
	}

	options := badgerhold.DefaultOptions
	options.Dir = storePath + "/db"
	options.ValueDir = storePath + "/db"

	store, err := badgerhold.Open(options)
	if err != nil {
		log.Errorf("bolt open failed %s", storePath)
		return err
	}
	m.store = store

	//	files, err := query.IndexedFiles()
	files := make(FileMap)
	fis := []FileInStore{}
	if err := store.Find(&fis, nil); err != nil {
		log.Errorf("bolt find failed %s %v", storePath, err)
	}
	log.Infof("files: init - number of files in store %d", len(fis))

	m.mfiles = make(MediafilesMap)
	m.files = files
	m.count = len(files)
	return nil

}

// Done should be called after all files have been processed.
func (m *Files) Done() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.store != nil {
		m.store.Close()
	}
	if (len(m.files) - m.count) == 0 {
		return
	}

	m.count = 0
	m.files = make(FileMap)
}

// Remove a file from the lookup table.
func (m *Files) Remove(fileName, fileRoot string) {
	key := path.Join(fileRoot, fileName)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.files, key)
	m.store.DeleteMatching(&FileInStore{}, badgerhold.Where("Name").Eq(key))
}

func (m *Files) Add(mf *MediaFile) {
	if mf == nil {
		return
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fileSize := mf.FileSize()
	mfs := m.mfiles[fileSize]
	mfs = append(mfs, mf)
	m.mfiles[fileSize] = mfs

	fullPath, mtime := mf.FileName(), mf.modTime.Unix()
	takenAt, takenAtSrc := mf.TakenAt()
	fi := FileInStore{
		ID:      mf.Hash(),
		Name:    mf.BaseName(),
		Size:    int(mf.FileSize()),
		Hash:    mf.Hash(),
		PathMap: map[string]int64{fullPath: mtime},
	}

	err := m.store.Insert(fi.ID, &fi)
	if err == badgerhold.ErrKeyExists {
		log.Infof("files: store.Insert key=%s existed for %s, update", fi.ID, fullPath)
		if err = m.store.FindOne(&fi, badgerhold.Where("ID").Eq(mf.Hash())); err == nil {
			mtime2, bExisted := fi.PathMap[fullPath]
			if bExisted == true && mtime != mtime2 {
				//TODO: choose a better one to update?
				log.Warnf("files: store.Insert file=%s existed. time %v-> %v", fullPath, mtime, mtime2)
			}
			if bExisted == false || mtime != mtime2 {
				fi.PathMap[fullPath] = mf.ModTime().Unix()
				m.store.Update(fi.ID, &fi)
			}
		}
	} else if err != nil {
		log.Errorf("files: store.Insert error %v %s", err, fi.Name)
	}
	log.Infof("store.Upsert %s %s %s %s, paths=%v", fi.ID, fi.Name, takenAt, takenAtSrc, fi.PathMap)

}

// Ignore tests of a file requires indexing, file name must be relative to the originals path.
func (m *Files) Ignore(fileName, fileRoot string, modTime time.Time, rescan bool) bool {
	timestamp := modTime.Unix()
	key := path.Join(fileRoot, fileName)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if rescan {
		m.files[key] = timestamp
		return false
	}

	mod, ok := m.files[key]

	if ok && mod == timestamp {
		return true
	} else {
		m.files[key] = timestamp
		return false
	}
}

// Indexed tests of a file was already indexed without modifying the files map.
func (m *Files) Indexed(fileName, fileRoot string, modTime time.Time, rescan bool) bool {
	if rescan {
		return false
	}

	timestamp := modTime.Unix()
	key := path.Join(fileRoot, fileName)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	mod, ok := m.files[key]

	if ok && mod == timestamp {
		return true
	} else {
		return false
	}
}
