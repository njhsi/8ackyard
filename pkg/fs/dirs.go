package fs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/njhsi/8ackyard/pkg/fastwalk"
)

var OriginalPaths = []string{
	"~/Images",
	"/var/lib/8ackyard/originals",
}

var ImportPaths = []string{
	"~/Import",
	"/var/lib/8ackyard/import",
}

var AssetPaths = []string{
	"assets",
	"/var/lib/8ackyard/assets",
}

// Dirs returns a slice of directories in a path, optional recursively and with symlinks.
//
// Warning: Following symlinks can make the result non-deterministic and hard to test!
func Dirs(root string, recursive bool, followLinks bool) (result []string, err error) {
	result = []string{}
	mutex := sync.Mutex{}

	symlinks := make(map[string]bool)
	symlinksMutex := sync.Mutex{}

	appendResult := func(fileName string) {
		fileName = strings.Replace(fileName, root, "", 1)
		mutex.Lock()
		defer mutex.Unlock()
		result = append(result, fileName)
	}

	// Ignore hidden folders as well as those listed in an optional ".ppignore" file.
	ignore := NewIgnoreList(IgnoreFile, true, false)
	_ = ignore.Dir(root)

	err = fastwalk.Walk(root, func(fileName string, typ os.FileMode) error {
		if typ.IsDir() || typ == os.ModeSymlink && followLinks {
			if ignore.Ignore(fileName) {
				return filepath.SkipDir
			}

			if fileName != root {
				if !recursive {
					appendResult(fileName)

					return filepath.SkipDir
				} else if typ != os.ModeSymlink {
					appendResult(fileName)

					return nil
				} else if resolved, err := filepath.EvalSymlinks(fileName); err == nil {
					symlinksMutex.Lock()
					defer symlinksMutex.Unlock()

					if _, ok := symlinks[resolved]; ok {
						return filepath.SkipDir
					} else {
						symlinks[resolved] = true
						appendResult(fileName)
					}

					return fastwalk.ErrTraverseLink
				}
			}
		}

		return nil
	})

	sort.Strings(result)

	return result, err
}

func FindDir(dirs []string) string {
	for _, dir := range dirs {
		absDir := Abs(dir)
		if PathExists(absDir) {
			return absDir
		}
	}

	return ""
}
