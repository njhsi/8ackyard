package backyard

import (
	"sync"
)

type PhotoMap map[string]uint

// Photos represents photo id lookup table, sorted by date and S2 cell id.
type Photos struct {
	count  int
	photos PhotoMap
	mutex  sync.RWMutex
}

// NewPhotos returns a new Photos instance.
func NewPhotos() *Photos {
	m := &Photos{
		photos: make(PhotoMap),
	}

	return m
}

// Init fetches the list from the database once.
func (m *Photos) Init() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.photos) > 0 {
		m.count = len(m.photos)
		return nil
	}

	photos := make(PhotoMap)

	m.photos = photos
	m.count = len(photos)
	return nil

}
