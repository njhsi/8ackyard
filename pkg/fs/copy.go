package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CopyWithStat(src, dest string) (err error) {
	if err := Copy(src, dest); err != nil {
		return nil
	}
	si, err := os.Lstat(src)
	if err != nil {
		return err
	}
	err = os.Chmod(dest, si.Mode())
	err = os.Chtimes(dest, si.ModTime(), si.ModTime())
	return nil
}

// Copies a file to a destination.
func Copy(src, dest string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("copy: %s (panic)", r)
		}
	}()

	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	thisFile, err := os.Open(src)

	if err != nil {
		return err
	}

	defer thisFile.Close()

	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, os.ModePerm)

	if err != nil {
		return err
	}

	defer destFile.Close()

	_, err = io.Copy(destFile, thisFile)

	if err != nil {
		return err
	}

	return nil
}
