package service

import (
	"sync"

	"github.com/njhsi/8ackyard/internal/backyard"
)

var onceIndex sync.Once

func initIndex() {
	services.Index = backyard.NewIndex(Files(), Photos())
}

func Index() *backyard.Index {
	onceIndex.Do(initIndex)

	return services.Index
}
