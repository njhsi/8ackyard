package backyard

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/djherbis/times"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/meta"
	"github.com/njhsi/8ackyard/pkg/fs"
	"github.com/njhsi/8ackyard/pkg/sanitize"
	"github.com/njhsi/8ackyard/pkg/txt"
)

// MediaFile represents a single photo, video or sidecar file.
type MediaFile struct {
	fileName     string
	fileRoot     string
	statErr      error
	modTime      time.Time
	fileSize     int64
	fileType     fs.FileFormat
	mimeType     string
	takenAt      time.Time
	takenAtSrc   string
	hash         string
	checksum     string
	md5sum       string
	hasJpeg      bool
	width        int
	height       int
	metaData     meta.Data
	metaDataOnce sync.Once
}

// NewMediaFile returns a new media file.
func NewMediaFile(fileName string) (*MediaFile, error) {
	m := &MediaFile{
		fileName: fileName,
		fileRoot: config.FileRoot,
		fileType: fs.FormatOther,
		metaData: meta.NewData(),
		width:    -1,
		height:   -1,
	}

	if _, _, err := m.Stat(); err != nil {
		return m, fmt.Errorf("media: %s not found", sanitize.Log(m.BaseName()))
	}

	return m, nil
}

// Stat returns the media file size and modification time rounded to seconds
func (m *MediaFile) Stat() (size int64, mod time.Time, err error) {
	if m.fileSize > 0 {
		return m.fileSize, m.modTime, m.statErr
	}

	if s, err := os.Stat(m.FileName()); err != nil {
		m.statErr = err
		m.modTime = time.Time{}
		m.fileSize = -1
	} else {
		m.statErr = nil
		m.modTime = s.ModTime().Round(time.Second)
		m.fileSize = s.Size()
	}

	return m.fileSize, m.modTime, m.statErr
}

// ModTime returns the file modification time.
func (m *MediaFile) ModTime() time.Time {
	_, modTime, _ := m.Stat()

	return modTime
}

// FileSize returns the file size in bytes.
func (m *MediaFile) FileSize() int64 {
	fileSize, _, _ := m.Stat()

	return fileSize
}

// DateCreated returns only the date on which the media file was probably taken in UTC.
func (m *MediaFile) DateCreated() time.Time {
	takenAt, _ := m.TakenAt()

	return takenAt
}

// TakenAt returns the date on which the media file was taken in UTC and the source of this information.
func (m *MediaFile) TakenAt() (time.Time, string) {
	if !m.takenAt.IsZero() {
		return m.takenAt, m.takenAtSrc
	}

	m.takenAt, m.takenAtSrc = time.Now().UTC(), "auto"
	tAuto, tName := m.takenAt, m.takenAt

	data := m.MetaData()
	if data.Error == nil && !data.TakenAt.IsZero() && data.TakenAt.Year() > 1000 {
		m.takenAt, m.takenAtSrc = data.TakenAt.UTC(), "meta"
		//log.Infof("media: %s was taken at %s (%s)", filepath.Base(m.fileName), tMeta.String(), "meta")
		return m.takenAt, m.takenAtSrc
	}

	if nameTime := txt.Time(m.fileName); !nameTime.IsZero() {
		tName = nameTime
		if nameTime := txt.Time(m.BaseName()); !nameTime.IsZero() && nameTime.Before(tName) {
			tName = nameTime
		}
		//log.Infof("media: %s was taken at %s (%s)", filepath.Base(m.fileName), tName.String(), "name")
	}

	fileInfo, err := times.Stat(m.FileName())
	if err == nil {
		if fileInfo.HasBirthTime() && fileInfo.BirthTime().Before(fileInfo.ModTime()) {
			tAuto = fileInfo.BirthTime().UTC()
			//log.Infof("media: %s was taken at %s ( birth time)", filepath.Base(m.fileName), tAuto.String(), "auto")
		} else {
			tAuto = fileInfo.ModTime().UTC()
			//log.Infof("media: %s was taken at %s ( mod time)", filepath.Base(m.fileName), tAuto.String(), "auto")
		}
	} else {
		log.Warnf("media: %s (file stat)", err.Error())
		log.Infof("media: %s was taken at %s (now)", filepath.Base(m.fileName), m.takenAt.String())
	}

	if tName.Before(tAuto) {
		m.takenAt, m.takenAtSrc = tName, "name"
	} else {
		m.takenAt, m.takenAtSrc = tAuto, "auto"
	}

	return m.takenAt, m.takenAtSrc
}

func (m *MediaFile) HasTimeAndPlace() bool {
	data := m.MetaData()

	result := !data.TakenAt.IsZero() && data.Lat != 0 && data.Lng != 0

	return result
}

// CameraModel returns the camera model with which the media file was created.
func (m *MediaFile) CameraModel() string {
	data := m.MetaData()

	return data.CameraModel
}

// CameraMake returns the make of the camera with which the file was created.
func (m *MediaFile) CameraMake() string {
	data := m.MetaData()

	return data.CameraMake
}

// LensModel returns the lens model of a media file.
func (m *MediaFile) LensModel() string {
	data := m.MetaData()

	return data.LensModel
}

// LensMake returns the make of the Lens.
func (m *MediaFile) LensMake() string {
	data := m.MetaData()

	return data.LensMake
}

// FocalLength return the length of the focal for a file.
func (m *MediaFile) FocalLength() int {
	data := m.MetaData()

	return data.FocalLength
}

// FNumber returns the F number with which the media file was created.
func (m *MediaFile) FNumber() float32 {
	data := m.MetaData()

	return data.FNumber
}

// Iso returns the iso rating as int.
func (m *MediaFile) Iso() int {
	data := m.MetaData()

	return data.Iso
}

// Exposure returns the exposure time as string.
func (m *MediaFile) Exposure() string {
	data := m.MetaData()

	return data.Exposure
}

// CanonicalName returns the canonical name of a media file.
func (m *MediaFile) CanonicalName() string {
	return fs.CanonicalName(m.DateCreated(), m.Checksum())
}

// CanonicalNameFromFile returns the canonical name of a file derived from the image name.
func (m *MediaFile) CanonicalNameFromFile() string {
	basename := filepath.Base(m.FileName())

	if end := strings.Index(basename, "."); end != -1 {
		return basename[:end] // Length of canonical name: 16 + 12
	}

	return basename
}

// CanonicalNameFromFileWithDirectory gets the canonical name for a MediaFile
// including the directory.
func (m *MediaFile) CanonicalNameFromFileWithDirectory() string {
	return m.Dir() + string(os.PathSeparator) + m.CanonicalNameFromFile()
}

func (m *MediaFile) Md5sum() string {
	if len(m.md5sum) == 0 {
		m.md5sum = fs.Md5sum(m.FileName())
	}

	return m.md5sum
}

// Hash returns the SHA1 hash of a media file.
func (m *MediaFile) Hash() string {
	if len(m.hash) == 0 {
		m.hash = fs.Hash(m.FileName())
	}

	return m.hash
}

// Checksum returns the CRC32 checksum of a media file.
func (m *MediaFile) Checksum() string {
	if len(m.checksum) == 0 {
		m.checksum = fs.Checksum(m.FileName())
	}

	return m.checksum
}

// EditedName returns the corresponding edited image file name as used by Apple (e.g. IMG_E12345.JPG).
func (m *MediaFile) EditedName() string {
	basename := filepath.Base(m.fileName)

	if strings.ToUpper(basename[:4]) == "IMG_" && strings.ToUpper(basename[:5]) != "IMG_E" {
		if filename := filepath.Dir(m.fileName) + string(os.PathSeparator) + basename[:4] + "E" + basename[4:]; fs.FileExists(filename) {
			return filename
		}
	}

	return ""
}

// FileName returns the filename.
func (m *MediaFile) FileName() string {
	return m.fileName
}

// BaseName returns the filename without path.
func (m *MediaFile) BaseName() string {
	return filepath.Base(m.fileName)
}

// SetFileName sets the filename to the given string.
func (m *MediaFile) SetFileName(fileName string) {
	m.fileName = fileName
	//	m.fileRoot = entity.RootUnknown
	m.fileRoot = "UNKNOWN" //TODO
}

// RelName returns the relative filename.
func (m *MediaFile) RelName(directory string) string {
	return fs.RelName(m.fileName, directory)
}

// RelPath returns the relative path without filename.
func (m *MediaFile) RelPath(directory string) string {
	pathname := m.fileName

	if i := strings.Index(pathname, directory); i == 0 {
		if i := strings.LastIndex(directory, string(os.PathSeparator)); i == len(directory)-1 {
			pathname = pathname[len(directory):]
		} else if i := strings.LastIndex(directory, string(os.PathSeparator)); i != len(directory) {
			pathname = pathname[len(directory)+1:]
		}
	}

	if end := strings.LastIndex(pathname, string(os.PathSeparator)); end != -1 {
		pathname = pathname[:end]
	} else if end := strings.LastIndex(pathname, string(os.PathSeparator)); end == -1 {
		pathname = ""
	}

	// Remove hidden sub directory if exists.
	if path.Base(pathname) == fs.HiddenPath {
		pathname = path.Dir(pathname)
	}

	// Use empty string for current / root directory.
	if pathname == "." || pathname == "/" || pathname == "\\" {
		pathname = ""
	}

	return pathname
}

// RelPrefix returns the relative path and file name prefix.
func (m *MediaFile) RelPrefix(directory string, stripSequence bool) string {
	if relativePath := m.RelPath(directory); relativePath != "" {
		return filepath.Join(relativePath, m.BasePrefix(stripSequence))
	}

	return m.BasePrefix(stripSequence)
}

// Dir returns the file path.
func (m *MediaFile) Dir() string {
	return filepath.Dir(m.fileName)
}

// SubDir returns a sub directory name.
func (m *MediaFile) SubDir(dir string) string {
	return filepath.Join(filepath.Dir(m.fileName), dir)
}

// BasePrefix returns the filename base without any extensions and path.
func (m *MediaFile) BasePrefix(stripSequence bool) string {
	return fs.BasePrefix(m.FileName(), stripSequence)
}

// AbsPrefix returns the directory and base filename without any extensions.
func (m *MediaFile) AbsPrefix(stripSequence bool) string {
	return fs.AbsPrefix(m.FileName(), stripSequence)
}

// MimeType returns the mime type.
func (m *MediaFile) MimeType() string {
	if m.mimeType != "" {
		return m.mimeType
	}

	m.mimeType = fs.MimeType(m.FileName())

	return m.mimeType
}

func (m *MediaFile) openFile() (*os.File, error) {
	handle, err := os.Open(m.fileName)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	return handle, nil
}

// Exists checks if a media file exists by filename.
func (m *MediaFile) Exists() bool {
	return fs.FileExists(m.FileName())
}

// Remove permanently removes a media file.
func (m *MediaFile) Remove() error {
	return os.Remove(m.FileName())
}

// HasSameName compares a media file with another media file and returns if
// their filenames are matching or not.
func (m *MediaFile) HasSameName(f *MediaFile) bool {
	if f == nil {
		return false
	}

	return m.FileName() == f.FileName()
}

// Move file to a new destination with the filename provided in parameter.
func (m *MediaFile) Move(dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	if err := os.Rename(m.fileName, dest); err != nil {
		log.Debugf("failed renaming file, fallback to copy and delete: %s", err.Error())
	} else {
		m.SetFileName(dest)

		return nil
	}

	if err := m.Copy(dest); err != nil {
		return err
	}

	if err := os.Remove(m.fileName); err != nil {
		return err
	}

	m.SetFileName(dest)

	return nil
}

// Copy a MediaFile to another file by destinationFilename.
func (m *MediaFile) Copy(dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	thisFile, err := m.openFile()

	if err != nil {
		log.Error(err.Error())
		return err
	}

	defer thisFile.Close()

	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, os.ModePerm)

	if err != nil {
		log.Error(err.Error())
		return err
	}

	defer destFile.Close()

	if _, err = io.Copy(destFile, thisFile); err != nil {
		log.Error(err.Error())
		return err
	}

	si, err := os.Lstat(m.fileName)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	err = os.Chmod(dest, si.Mode())
	err = os.Chtimes(dest, si.ModTime(), si.ModTime())

	return nil
}

// Extension returns the filename extension of this media file.
func (m *MediaFile) Extension() string {
	return strings.ToLower(filepath.Ext(m.fileName))
}

// IsJpeg return true if this media file is a JPEG image.
func (m *MediaFile) IsJpeg() bool {
	// Don't import/use existing thumbnail files (we create our own)
	if m.Extension() == ".thm" {
		return false
	}

	return m.MimeType() == fs.MimeTypeJpeg
}

// IsPng returns true if this is a PNG file.
func (m *MediaFile) IsPng() bool {
	return m.MimeType() == fs.MimeTypePng
}

// IsGif returns true if this is a GIF file.
func (m *MediaFile) IsGif() bool {
	return m.MimeType() == fs.MimeTypeGif
}

// IsTiff returns true if this is a TIFF file.
func (m *MediaFile) IsTiff() bool {
	return m.HasFileType(fs.FormatTiff) && m.MimeType() == fs.MimeTypeTiff
}

// IsHEIF returns true if this is a High Efficiency Image File Format file.
func (m *MediaFile) IsHEIF() bool {
	return m.MimeType() == fs.MimeTypeHEIF
}

// IsBitmap returns true if this is a bitmap file.
func (m *MediaFile) IsBitmap() bool {
	return m.MimeType() == fs.MimeTypeBitmap
}

// IsVideo returns true if this is a video file.
func (m *MediaFile) IsVideo() bool {
	return strings.HasPrefix(m.MimeType(), "video/") || m.MediaType() == fs.MediaVideo
}

func (m *MediaFile) IsAudio() bool {
	return strings.HasPrefix(m.MimeType(), "audio/") || m.MediaType() == fs.MediaAudio
}

// IsJson return true if this media file is a json sidecar file.
func (m *MediaFile) IsJson() bool {
	return m.HasFileType(fs.FormatJson)
}

// FileType returns the file type (jpg, gif, tiff,...).
func (m *MediaFile) FileType() fs.FileFormat {
	switch {
	case m.IsJpeg():
		return fs.FormatJpeg
	case m.IsPng():
		return fs.FormatPng
	case m.IsGif():
		return fs.FormatGif
	case m.IsHEIF():
		return fs.FormatHEIF
	case m.IsBitmap():
		return fs.FormatBitmap
	default:
		return fs.GetFileFormat(m.fileName)
	}
}

// MediaType returns the media type (video, image, raw, sidecar,...).
func (m *MediaFile) MediaType() fs.MediaType {
	return fs.GetMediaType(m.fileName)
}

// HasFileType returns true if this is the given type.
func (m *MediaFile) HasFileType(fileType fs.FileFormat) bool {
	if fileType == fs.FormatJpeg {
		return m.IsJpeg()
	}

	return m.FileType() == fileType
}

// IsRaw returns true if this is a RAW file.
func (m *MediaFile) IsRaw() bool {
	return m.HasFileType(fs.FormatRaw)
}

// IsImageOther returns true if this is a PNG, GIF, BMP or TIFF file.
func (m *MediaFile) IsImageOther() bool {
	switch {
	case m.IsPng(), m.IsGif(), m.IsTiff(), m.IsBitmap():
		return true
	default:
		return false
	}
}

// IsXMP returns true if this is a XMP sidecar file.
func (m *MediaFile) IsXMP() bool {
	return m.FileType() == fs.FormatXMP
}

// IsSidecar returns true if this is a sidecar file (containing metadata).
func (m *MediaFile) IsSidecar() bool {
	return m.MediaType() == fs.MediaSidecar
}

// IsPlayableVideo returns true if this is a supported video file format.
func (m *MediaFile) IsPlayableVideo() bool {
	return m.IsVideo() && (m.HasFileType(fs.FormatMp4) || m.HasFileType(fs.FormatAvc))
}

// IsPhoto returns true if this file is a photo / image.
func (m *MediaFile) IsPhoto() bool {
	return m.IsJpeg() || m.IsRaw() || m.IsHEIF() || m.IsImageOther()
}

// ExifSupported returns true if parsing exif metadata is supported for the media file type.
func (m *MediaFile) ExifSupported() bool {
	return m.IsJpeg() || m.IsRaw() || m.IsHEIF() || m.IsPng() || m.IsTiff()
}

// IsMedia returns true if this is a media file (photo or video, not sidecar or other).
func (m *MediaFile) IsMedia() bool {
	return m.IsJpeg() || m.IsVideo() || m.IsRaw() || m.IsHEIF() || m.IsImageOther() || m.IsAudio()
}

func (m *MediaFile) decodeDimensions() error {
	if !m.IsMedia() {
		return fmt.Errorf("failed decoding dimensions for %s", sanitize.Log(m.BaseName()))
	}

	if m.IsJpeg() || m.IsPng() || m.IsGif() {
		file, err := os.Open(m.FileName())

		if err != nil || file == nil {
			return err
		}

		defer file.Close()

		size, _, err := image.DecodeConfig(file)

		if err != nil {
			return err
		}

		orientation := m.Orientation()

		if orientation > 4 && orientation <= 8 {
			m.width = size.Height
			m.height = size.Width
		} else {
			m.width = size.Width
			m.height = size.Height
		}
	} else if data := m.MetaData(); data.Error == nil {
		m.width = data.ActualWidth()
		m.height = data.ActualHeight()
	} else {
		return data.Error
	}

	return nil
}

// Width return the width dimension of a MediaFile.
func (m *MediaFile) Width() int {
	if !m.IsMedia() {
		return 0
	}

	if m.width < 0 {
		if err := m.decodeDimensions(); err != nil {
			log.Debugf("media: %s", err)
		}
	}

	return m.width
}

// Height returns the height dimension of a MediaFile.
func (m *MediaFile) Height() int {
	if !m.IsMedia() {
		return 0
	}

	if m.height < 0 {
		if err := m.decodeDimensions(); err != nil {
			log.Debugf("media: %s", err)
		}
	}

	return m.height
}

// AspectRatio returns the aspect ratio of a MediaFile.
func (m *MediaFile) AspectRatio() float32 {
	width := float64(m.Width())
	height := float64(m.Height())

	if width <= 0 || height <= 0 {
		return 0
	}

	aspectRatio := float32(math.Round((width/height)*100) / 100)

	return aspectRatio
}

// Portrait tests if the image is a portrait.
func (m *MediaFile) Portrait() bool {
	return m.Width() < m.Height()
}

// Megapixels returns the resolution in megapixels.
func (m *MediaFile) Megapixels() int {
	return int(math.Round(float64(m.Width()*m.Height()) / 1000000))
}

// Orientation returns the Exif orientation of the media file.
func (m *MediaFile) Orientation() int {
	if data := m.MetaData(); data.Error == nil {
		return data.Orientation
	}

	return 1
}

// ToJson uses exiftool to export metadata to a json file.
func (m *MediaFile) ToJson() (jsonName string, err error) {

	jsonName, err = m.ExifToolJsonName()

	if err != nil {
		return "", nil
	}

	if fs.FileExists(jsonName) {
		return jsonName, nil
	}

	cmd := exec.Command("exiftool", "-n", "-m", "-api", "LargeFileSupport", "-j", m.FileName())

	// Fetch command output.
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run convert command.
	if err := cmd.Run(); err != nil {
		if stderr.String() != "" {
			return "", errors.New(stderr.String())
		} else {
			return "", err
		}
	}

	// Write output to file.
	if err := os.WriteFile(jsonName, []byte(out.String()), os.ModePerm); err != nil {
		return "", err
	}

	// Check if file exists.
	if !fs.FileExists(jsonName) {
		return "", fmt.Errorf("exiftool: failed creating %s", filepath.Base(jsonName))
	}

	return jsonName, err
}

//-------
// MediaFiles represents a slice of files.
type MediaFiles []*MediaFile

// Len returns the length of the file slice.
func (f MediaFiles) Len() int {
	return len(f)
}

// Less compares two files based on the filename.
func (f MediaFiles) Less(i, j int) bool {
	fileName1 := f[i].FileName()
	fileName2 := f[j].FileName()

	if len(fileName1) == len(fileName2) {
		return fileName1 < fileName2
	}

	return len(fileName1) < len(fileName2)
}

// Swap the position of two files in the slice.
func (f MediaFiles) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
