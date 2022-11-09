package backyard

import (
	"fmt"
	"path/filepath"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/event"
	"github.com/photoprism/photoprism/pkg/fs"
)

var log = event.Log

type S []string

func logWarn(prefix string, err error) {
	if err != nil {
		log.Warnf("%s: %s", prefix, err.Error())
	}
}

// CachePath returns a cache directory name based on the base path, file hash and cache namespace.
func CachePath(fileHash, namespace string) (cachePath string, err error) {
	return fs.CachePath(config.CachePath(), fileHash, namespace, true)
}

// CacheName returns an absolute cache file name based on the base path, file hash and cache namespace.
func CacheName(fileHash, namespace, cacheKey string) (cacheName string, err error) {
	if cacheKey == "" {
		return "", fmt.Errorf("cache: key for hash '%s' is empty", fileHash)
	}

	cachePath, err := CachePath(fileHash, namespace)

	if err != nil {
		return "", err
	}

	cacheName = filepath.Join(cachePath, fmt.Sprintf("%s_%s", fileHash, cacheKey))

	return cacheName, nil
}
