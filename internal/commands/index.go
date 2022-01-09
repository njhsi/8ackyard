package commands

import (
	"context"
	"strings"
	"time"

	"github.com/dustin/go-humanize/english"

	"github.com/urfave/cli"

	"github.com/njhsi/8ackyard/internal/backyard"
	"github.com/njhsi/8ackyard/internal/service"
	"github.com/njhsi/8ackyard/pkg/fs"
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
	cli.StringFlag{
		Name:  "backup, b",
		Usage: "backup to where, after indexing",
		Value: "",
	},
	cli.IntFlag{
		Name:  "workers, n",
		Usage: "number of workers",
		Value: 4,
	},
}

// indexAction indexes all photos in originals directory (photo library)
func indexAction(ctx *cli.Context) error {
	start := time.Now()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	backupPath := ctx.String("backup")
	numWorkers := ctx.Int("workers")

	// Use first argument to limit scope if set.
	subPath := strings.TrimSpace(ctx.Args().First())

	if subPath == "" {
		log.Errorf("indexing not going as subpath is not provided, but it's a must for originals=%s", subPath)
		return nil
	} else {
		log.Infof("indexing originals= %s, backup=%s, n=%d", subPath, backupPath, numWorkers)
	}

	var indexed fs.Done

	if w := service.Index(); w != nil {
		opt := backyard.IndexOptions{
			Path:       subPath,
			BackupPath: backupPath,
			NumWorkers: numWorkers,
			Rescan:     true,
			Convert:    false,
			Stack:      true,
		}

		indexed = w.Start(opt)
	}

	elapsed := time.Since(start)

	log.Infof("indexed %s in %s", english.Plural(len(indexed), "file", "files"), elapsed)

	return nil
}
