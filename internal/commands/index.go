package commands

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize/english"

	"github.com/urfave/cli"

	"github.com/njhsi/8ackyard/internal/backyard"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/service"
	"github.com/njhsi/8ackyard/pkg/fs"
	"github.com/njhsi/8ackyard/pkg/sanitize"
)

// IndexCommand registers the index cli command.
var IndexCommand = cli.Command{
	Name:      "index",
	Usage:     "Indexes original media files",
	ArgsUsage: "[originals subfolder]",
	Flags:     indexFlags,
	Action:    indexAction,
}

var indexFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "force, f",
		Usage: "re-index all originals, including unchanged files",
	},
	cli.BoolFlag{
		Name:  "cleanup, c",
		Usage: "remove orphan index entries and thumbnails",
	},
}

// indexAction indexes all photos in originals directory (photo library)
func indexAction(ctx *cli.Context) error {
	start := time.Now()

	conf := config.NewConfig(ctx)
	service.SetConfig(conf)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := conf.Init(); err != nil {
		return err
	}

	// Use first argument to limit scope if set.
	subPath := strings.TrimSpace(ctx.Args().First())

	if subPath == "" {
		log.Infof("indexing originals in %s", sanitize.Log(conf.OriginalsPath()))
	} else {
		log.Infof("indexing originals in %s", sanitize.Log(filepath.Join(conf.OriginalsPath(), subPath)))
	}

	if conf.ReadOnly() {
		log.Infof("config: read-only mode enabled")
	}

	var indexed fs.Done

	if w := service.Index(); w != nil {
		opt := backyard.IndexOptions{
			Path:    subPath,
			Rescan:  true,
			Convert: false,
			Stack:   true,
		}

		indexed = w.Start(opt)
	}

	elapsed := time.Since(start)

	log.Infof("indexed %s in %s", english.Plural(len(indexed), "file", "files"), elapsed)

	conf.Shutdown()

	return nil
}
