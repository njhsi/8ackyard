package service

import (
	"github.com/njhsi/8ackyard/internal/backyard"
	gc "github.com/patrickmn/go-cache"
)

var services struct {
	FolderCache *gc.Cache
	CoverCache  *gc.Cache
	ThumbCache  *gc.Cache
	//	Convert     *backyard.Convert
	Files *backyard.Files
	//	Import      *backyard.Import
	Index *backyard.Index
	//	Moments     *backyard.Moments
	//	Places      *backyard.Places
	//	Purge       *backyard.Purge
	//	CleanUp     *backyard.CleanUp
}
