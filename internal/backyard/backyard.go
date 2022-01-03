package backyard

import (
	"github.com/njhsi/8ackyard/internal/event"
)

var log = event.Log

type S []string

func logWarn(prefix string, err error) {
	if err != nil {
		log.Warnf("%s: %s", prefix, err.Error())
	}
}
