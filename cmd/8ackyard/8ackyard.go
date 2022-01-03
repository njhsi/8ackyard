package main

import (
	"os"
	"path/filepath"

	"github.com/njhsi/8ackyard/internal/commands"
	"github.com/njhsi/8ackyard/internal/config"
	"github.com/njhsi/8ackyard/internal/event"
	"github.com/urfave/cli"
)

var version = "development"
var log = event.Log

func main() {
	app := cli.NewApp()
	app.Name = "PhotoPrism"
	app.HelpName = filepath.Base(os.Args[0])
	app.Usage = "Browse Your Life in Pictures"
	app.Description = "For setup instructions and a user guide, visit https://docs.photoprism.app/"
	app.Version = version
	app.Copyright = "(c) 2018-2021 Michael Mayer <hello@photoprism.app>"
	app.EnableBashCompletion = true
	app.Flags = config.GlobalFlags

	app.Commands = []cli.Command{
		commands.StartCommand,
		commands.StopCommand,
		commands.StatusCommand,
		commands.IndexCommand,
		commands.ImportCommand,
		commands.CopyCommand,
		commands.FacesCommand,
		commands.PlacesCommand,
		commands.PurgeCommand,
		commands.CleanUpCommand,
		commands.OptimizeCommand,
		commands.MomentsCommand,
		commands.ConvertCommand,
		commands.ThumbsCommand,
		commands.MigrateCommand,
		commands.BackupCommand,
		commands.RestoreCommand,
		commands.ResetCommand,
		commands.PasswdCommand,
		commands.UsersCommand,
		commands.ConfigCommand,
		commands.VersionCommand,
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
	}
}
