package commands

import (
	"os"
	"syscall"

	"github.com/njhsi/8ackyard/internal/event"
	"github.com/njhsi/8ackyard/pkg/fs"
	"github.com/sevlyar/go-daemon"
)

var log = event.Log

// childAlreadyRunning tests if a .pid file at filePath is a running process.
// it returns the pid value and the running status (true or false).
func childAlreadyRunning(filePath string) (pid int, running bool) {
	if !fs.FileExists(filePath) {
		return pid, false
	}

	pid, err := daemon.ReadPidFile(filePath)
	if err != nil {
		return pid, false
	}

	process, err := os.FindProcess(int(pid))
	if err != nil {
		return pid, false
	}

	return pid, process.Signal(syscall.Signal(0)) == nil
}
