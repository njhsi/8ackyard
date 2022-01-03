package service

import (
	"sync"

	"github.com/njhsi/8ackyard/internal/backyard"
)

var onceFiles sync.Once

func initFiles() {
	services.Files = backyard.NewFiles()
}

func Files() *backyard.Files {
	onceFiles.Do(initFiles)

	return services.Files
}
