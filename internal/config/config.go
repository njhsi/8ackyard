package config

func OriginalsPath() string {
	return "/tmp/Originals"
}

func CachePath() string {
	return "/tmp/Caches"
}

func SidecarPath() string {
	return "/tmp/Caches/sidecar"
}

func OriginalsLimit() int64 {
	return 4 * 1024 * 1024 * 1024
}
