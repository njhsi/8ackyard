package backyard

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/h2non/filetype"
	"github.com/zeebo/xxh3"
)

type TimeBornSrcType string

const (
	TimeBornSrcMeta TimeBornSrcType = "meta"
	TimeBornSrcStat TimeBornSrcType = "stat"
	TimeBornSrcName TimeBornSrcType = "name"
)

type FileIndexed struct {
	ID    uint64 //xxh3 of file content
	Path  string //full path
	Size  int64
	Mtime time.Time //mod time

	TimeBorn    time.Time       //to be save in db as unix seconds
	TimeBornSrc TimeBornSrcType //meta, name, auto
	Format      string          // extension etc..
	Mime        string
	Duplica     map[string]int64 //fullpath:modtime
	Info        string
}

func fileStat(fileName string) (error, time.Time, int64) {
	s, err := os.Stat(fileName)
	if err != nil {
		return err, time.Time{}, -1
	}

	return nil, s.ModTime().Round(time.Second), s.Size()

}

func fileFormat(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if typ, err := filetype.MatchFile(fileName); err == nil {
		ext = typ.Extension
	}
	return ext
}

func fileXXH3(fileName string) uint64 {
	file, err := os.Open(fileName)
	if err != nil {
		return 0
	}
	defer file.Close()

	hash := xxh3.New()
	if _, err := io.Copy(hash, file); err != nil {
		return 0
	}
	return hash.Sum64()
}

func NewFileIndex(fileName string) (error, *FileIndexed) {
	err, mtimeF, sizeF := fileStat(fileName)
	if err != nil || sizeF == 0 {
		log.Errorf("NewFileIndex: stat %v err - %v", fileName, err)
		return err, nil
	}

	birthF, birthSrcF := guestTimeBorn(fileName), TimeBornSrcName
	if birthF.Year() < 1900 || mtimeF.Before(birthF) {
		birthF, birthSrcF = mtimeF, TimeBornSrcStat
	}

	fi := &FileIndexed{
		Path:        fileName,
		Size:        sizeF,
		Mtime:       mtimeF,
		TimeBorn:    birthF,
		TimeBornSrc: birthSrcF,
	}

	file, err := os.Open(fileName)
	if err != nil {
		log.Errorf("NewFileIndex: open %v err - %v", fileName, err)
		return err, fi
	}
	defer file.Close()
	buffer := make([]byte, 8192) // 8K makes msooxml tests happy and allows for expanded custom file checks

	//1. format
	formatF := "UNK"
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		log.Errorf("NewFileIndex: read %v err - %v", fileName, err)
		formatF = "UNK"
	} else {
		typ, err := filetype.Match(buffer)
		if err != nil {
			log.Errorf("NewFileIndex: Match %v err - %v", fileName, err)
			formatF = "UNK"
		} else {
			formatF = typ.Extension
		}
	}
	fi.Format = formatF

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatalf("NewFileIndex: Seek %v err - %v", fileName, err)
		return err, fi
	}

	//2. hash
	hash := xxh3.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Errorf("NewFileIndex: Copy for hash %v err - %v", fileName, err)
	}
	fi.ID = hash.Sum64()

	return nil, fi
}
func Uint64ToString(s uint64) string {
	var result []byte
	result = append(
		result,
		byte(s>>56),
		byte(s>>48),
		byte(s>>40),
		byte(s>>32),
		byte(s>>24),
		byte(s>>16),
		byte(s>>8),
		byte(s),
	)

	return hex.EncodeToString(result)

}

func buildExifJson(fileName string, et *exiftool.Exiftool) ([]byte, error) {
	err := errors.New("buildExifJson: non exif existed in " + fileName)
	var result []byte
	fileInfos := et.ExtractMetadata(fileName)
	for _, fileInfo := range fileInfos {
		if fileInfo.Err != nil {
			log.Errorf("buildExifJson: Error in exiftool %v: %v\n", fileInfo.File, fileInfo.Err)
			continue
		}

		result, err = json.MarshalIndent(fileInfo.Fields, "", "")
		log.Infof("buildExifJson: got exif - %v , err=%v", fileInfo.File, err)
	}
	return result, err
}

func guestTimeBorn(fileName string) time.Time {
	//try name
	tname, tbase := TimeFromFileName(fileName), TimeFromFileName(filepath.Base(fileName))
	if tbase.Year() > 1980 && tbase.Before(tname) {
		tname = tbase
	}
	return tname
}
