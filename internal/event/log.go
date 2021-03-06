package event

import (
	"os"

	log "github.com/dsoprea/go-logging"
	"github.com/leandro-lugaresi/hub"
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

type Hook struct {
	hub *hub.Hub
}

func NewHook(hub *hub.Hub) *Hook {
	return &Hook{hub: hub}
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	log.Errorf("Fire not implemented TODO")
	return nil
}

func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func init() {
	hooks := logrus.LevelHooks{}
	//	hooks.Add(NewHook(SharedHub()))

	Log = &logrus.Logger{
		Out:          os.Stderr,
		Formatter:    &logrus.TextFormatter{},
		Hooks:        hooks,
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
}
