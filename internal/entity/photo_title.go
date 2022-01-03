package entity

import (
	"path/filepath"
	"strings"

	"github.com/njhsi/8ackyard/pkg/fs"
)

// HasTitle checks if the photo has a title.
func (m *Photo) HasTitle() bool {
	return m.PhotoTitle != ""
}

// NoTitle checks if the photo has no Title
func (m *Photo) NoTitle() bool {
	return m.PhotoTitle == ""
}

// SetTitle changes the photo title and clips it to 300 characters.
func (m *Photo) SetTitle(title, source string) {

	if title == "" {
		return
	}

	if (SrcPriority[source] < SrcPriority[m.TitleSrc]) && m.HasTitle() {
		return
	}

	m.PhotoTitle = title
	m.TitleSrc = source
}

// UpdateAndSaveTitle updates the photo title and saves it.

// FileTitle returns a photo title based on the file name and/or path.
func (m *Photo) FileTitle() string {
	// Generate title based on photo name, if not generated:
	if !fs.IsGenerated(m.PhotoName) {
		if title := m.PhotoName; title != "" {
			return title
		}
	}

	// Generate title based on original file name, if any:
	if m.OriginalName != "" {
		if title := m.OriginalName; !fs.IsGenerated(m.OriginalName) && title != "" {
			return title
		} else if title := filepath.Dir(m.OriginalName); title != "" {
			return title
		}
	}

	// Generate title based on photo path, if any:
	if m.PhotoPath != "" && !fs.IsGenerated(m.PhotoPath) {
		return m.PhotoPath
	}

	return ""
}

// SubjectNames returns all known subject names.
func (m *Photo) SubjectNames() []string {
	if f, err := m.PrimaryFile(); err == nil {
		return f.SubjectNames()
	}

	return nil
}

// SubjectKeywords returns keywords for all known subject names.
func (m *Photo) SubjectKeywords() []string {
	return strings.Join(m.SubjectNames(), " ")
}
