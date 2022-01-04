package backyard

import (
	"path"
	"sync"
	"time"
)

type FileMap map[string]int64

// Files represents a list of already indexed file names and their unix modification timestamps.
type Files struct {
	count int
	files FileMap
	mutex sync.RWMutex
}

// NewFiles returns a new Files instance.
func NewFiles() *Files {
	m := &Files{
		files: make(FileMap),
	}

	return m
}

// Init fetches the list from the database once.
func (m *Files) Init() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.files) > 0 {
		m.count = len(m.files)
		return nil
	}

	//	files, err := query.IndexedFiles()
	files := make(FileMap)

	m.files = files
	m.count = len(files)
	return nil

}

// Done should be called after all files have been processed.
func (m *Files) Done() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
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