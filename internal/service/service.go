package service

import (
	"github.com/njhsi/8ackyard/internal/backyard"
	"github.com/njhsi/8ackyard/internal/config"

	gc "github.com/patrickmn/go-cache"
)

var conf *config.Config

var services struct {
	FolderCache *gc.Cache
	CoverCache  *gc.Cache
	ThumbCache  *gc.Cache
	Convert     *backyard.Convert
	Files       *backyard.Files
	Photos      *backyard.Photos
	Import      *backyard.Import
	Index       *backyard.Index
	Moments     *backyard.Moments
	Places      *backyard.Places
	Purge       *backyard.Purge
	CleanUp     *backyard.CleanUp
}

func SetConfig(c *config.Config) {
	if c == nil {
		panic("config is nil")
	}

	conf = c

	backyard.SetConfig(c)
}

func Config() *config.Config {
	if conf == nil {
		panic("config is nil")
	}

	return conf
}
