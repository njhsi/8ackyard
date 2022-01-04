package main

import (
	"os"
	"path/filepath"

	"github.com/njhsi/8ackyard/internal/commands"
	"github.com/njhsi/8ackyard/internal/event"
	"github.com/urfave/cli"
)

var version = "development"
var log = event.Log

func main() {
	app := cli.NewApp()
	app.Name = "8ackyard"
	app.HelpName = filepath.Base(os.Args[0])
	app.Usage = "TODO usage"
	app.Description = "TODO desc"
	app.Version = version
	app.Copyright = "TODO copyright"
	app.EnableBashCompletion = true

	app.Commands = []cli.Command{
		commands.IndexCommand,
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
	}
}
