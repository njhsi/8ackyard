package backyard

import (
	"fmt"
	"path/filepath"

	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/pkg/sanitize"
)

// IndexMain indexes the main file from a group of related files and returns the result.
func IndexMain(related *RelatedFiles, ind *Index, opt IndexOptions) (result IndexResult) {
	// Skip sidecar files without related media file.
	if related.Main == nil {
		result.Err = fmt.Errorf("index: found no main file for %s", sanitize.Log(related.String()))
		result.Status = IndexFailed
		return result
	}

	f := related.Main
	sizeLimit := config.OriginalsLimit()

	// Enforce file size limit for originals.
	if sizeLimit > 0 && f.FileSize() > sizeLimit {
		result.Err = fmt.Errorf("index: %s exceeds file size limit (%d / %d MB)", sanitize.Log(f.BaseName()), f.FileSize()/(1024*1024), sizeLimit/(1024*1024))
		result.Status = IndexFailed
		return result
	}

	if f.NeedsExifToolJson() {
		if jsonName, err := f.ToJson(); err != nil {
			log.Debugf("index: %s in %s (extract metadata)", sanitize.Log(err.Error()), sanitize.Log(f.BaseName()))
		} else {
			log.Debugf("index: created %s", filepath.Base(jsonName))
		}
	}

	//	result = ind.MediaFile(f, opt, "")
	result.Status = IndexAdded

	log.Infof("index: %s main %s file %s", result, f.FileType(), sanitize.Log(f.RelName(ind.originalsPath())))

	return result
}

// IndexMain indexes a group of related files and returns the result.
func IndexRelated(related RelatedFiles, ind *Index, opt IndexOptions) (result IndexResult) {
	done := make(map[string]bool)
	sizeLimit := config.OriginalsLimit()

	result = IndexMain(&related, ind, opt)

	if result.Failed() {
		log.Warn(result.Err)
		return result
	} else if !result.Success() || result.Stacked() {
		// Skip related files if main file was stacked or indexing was not completely successful.
		return result
	}

	done[related.Main.FileName()] = true

	i := 0

	for i < len(related.Files) {
		f := related.Files[i]
		i++

		if f == nil {
			continue
		}

		if done[f.FileName()] {
			continue
		}

		done[f.FileName()] = true

		// Enforce file size limit for originals.
		if sizeLimit > 0 && f.FileSize() > sizeLimit {
			log.Warnf("index: %s exceeds file size limit (%d / %d MB)", sanitize.Log(f.BaseName()), f.FileSize()/(1024*1024), sizeLimit/(1024*1024))
			continue
		}

		if f.NeedsExifToolJson() {
			if jsonName, err := f.ToJson(); err != nil {
				log.Debugf("index: %s in %s (extract metadata)", sanitize.Log(err.Error()), sanitize.Log(f.BaseName()))
			} else {
				log.Debugf("index: created %s", filepath.Base(jsonName))
			}
		}

		//		res := ind.MediaFile(f, opt, "")
		res := result

		log.Infof("index: %s related %s file %s", res, f.FileType(), sanitize.Log(f.BaseName()))
	}

	return result
}
