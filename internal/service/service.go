package service

import (
	"github.com/njhsi/8ackyard/internal/backyard"
	"github.com/njhsi/8ackyard/internal/classify"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/face"
	"github.com/njhsi/8ackyard/internal/nsfw"
	"github.com/njhsi/8ackyard/internal/query"
	"github.com/njhsi/8ackyard/internal/session"

	gc "github.com/patrickmn/go-cache"
)

var conf *config.Config

var services struct {
	FolderCache *gc.Cache
	CoverCache  *gc.Cache
	ThumbCache  *gc.Cache
	Classify    *classify.TensorFlow
	Convert     *backyard.Convert
	Files       *backyard.Files
	Photos      *backyard.Photos
	Import      *backyard.Import
	Index       *backyard.Index
	Moments     *backyard.Moments
	Faces       *backyard.Faces
	Places      *backyard.Places
	Purge       *backyard.Purge
	CleanUp     *backyard.CleanUp
	Nsfw        *nsfw.Detector
	FaceNet     *face.Net
	Query       *query.Query
	Resample    *backyard.Resample
	Session     *session.Session
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
