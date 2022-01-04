package backyard

import (
	"fmt"
	"path/filepath"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/meta"
	"github.com/njhsi/8ackyard/pkg/fs"
)

// ExifToolJsonName returns the cached ExifTool metadata file name.
func (m *MediaFile) ExifToolJsonName() (string, error) {

	return CacheName(m.Hash(), "json", "exiftool.json")
}

// NeedsExifToolJson tests if an ExifTool JSON file needs to be created.
func (m *MediaFile) NeedsExifToolJson() bool {
	if !m.IsMedia() {
		return false
	}

	jsonName, err := m.ExifToolJsonName()

	if err != nil {
		return false
	}

	return !fs.FileExists(jsonName)
}

// ReadExifToolJson reads metadata from a cached ExifTool JSON file.
func (m *MediaFile) ReadExifToolJson() error {
	jsonName, err := m.ExifToolJsonName()

	if err != nil {
		return err
	}

	return m.metaData.JSON(jsonName, "")
}

// MetaData returns exif meta data of a media file.
func (m *MediaFile) MetaData() (result meta.Data) {
	m.metaDataOnce.Do(func() {
		var err error

		if m.ExifSupported() {
			err = m.metaData.Exif(m.FileName(), m.FileType())
		} else {
			err = fmt.Errorf("exif not supported")
		}

		// Parse regular JSON sidecar files ("img_1234.json")
		if !m.IsSidecar() {
			if jsonFiles := fs.FormatJson.FindAll(m.FileName(), []string{config.SidecarPath(), fs.HiddenPath}, config.OriginalsPath(), false); len(jsonFiles) == 0 {
				log.Tracef("metadata: found no additional sidecar file for %s", filepath.Base(m.FileName()))
			} else {
				for _, jsonFile := range jsonFiles {
					jsonErr := m.metaData.JSON(jsonFile, m.BaseName())

					if jsonErr != nil {
						log.Debug(jsonErr)
					} else {
						err = nil
					}
				}
			}

			if jsonErr := m.ReadExifToolJson(); jsonErr != nil {
				log.Debug(jsonErr)
			} else {
				err = nil
			}
		}

		if err != nil {
			m.metaData.Error = err
			log.Debugf("metadata: %s in %s", err, m.BaseName())
		}
	})

	return m.metaData
}
