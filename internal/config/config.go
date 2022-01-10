package config

var CacheDir = "/tmp/cache8/"

func CachePath() string {
	return CacheDir
}

func OriginalsLimit() int64 {
	return 8 * 1024 * 1024 * 1024
}
