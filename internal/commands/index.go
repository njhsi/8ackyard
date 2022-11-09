package commands

import (
	"context"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/dustin/go-humanize/english"

	"github.com/urfave/cli"

	"github.com/njhsi/8ackyard/internal/backyard"
	"github.com/njhsi/8ackyard/internal/mutex"
	"github.com/njhsi/8ackyard/internal/service"
	"github.com/photoprism/photoprism/pkg/fs"
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
	cli.StringFlag{
		Name:  "cache, s",
		Usage: "cache path",
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
	// handle ctrl+c
	contxt, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	go func() {
		select {
		case <-signalChan: // first signal, cancel context
			mutex.MainWorker.Cancel()
			//			cancel()
		case <-contxt.Done():
		}
		<-signalChan // second signal, hard exit
		os.Exit(2)
	}()

	// starting mainly
	start := time.Now()

	backupPath := ctx.String("backup")
	cachePath := ctx.String("cache")
	if cachePath == "" && backupPath != "" {
		cachePath = backupPath + "/.cache8"
	}
	if cachePath == "" {
		cachePath = "/tmp/cache8/"
	}
	numWorkers := ctx.Int("workers")

	// Use first argument to limit scope if set.
	subPath := strings.TrimSpace(ctx.Args().First())
	if subPath == "" {
		log.Errorf("indexing not going as subpath is not provided, but it's a must for originals=%s", subPath)
		return nil
	} else {
		if strings.HasPrefix(subPath, "/") || strings.HasPrefix(subPath, "~") {

		} else {
			cwd, _ := os.Getwd()
			subPath = path.Join(cwd, subPath)
		}
		subPath = path.Clean(subPath)
		log.Infof("indexing originals= %s, backup=%s, cache=%s, n=%d", subPath, backupPath, cachePath, numWorkers)
	}

	var indexed fs.Done

	if w := service.Index(); w != nil {
		opt := backyard.IndexOptions{
			Path:       subPath,
			BackupPath: backupPath,
			CachePath:  cachePath,
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
