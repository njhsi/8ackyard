package service

import (
	"sync"

	"github.com/njhsi/8ackyard/internal/backyard"
)

var oncePhotos sync.Once

func initPhotos() {
	services.Photos = backyard.NewPhotos()
}

func Photos() *backyard.Photos {
	oncePhotos.Do(initPhotos)

	return services.Photos
}
