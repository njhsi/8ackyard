package config

import (
	"github.com/klauspost/cpuid/v2"
	"github.com/urfave/cli"
)

// GlobalFlags describes global command-line parameters and flags.
var GlobalFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "log-level, l",
		Usage:  "trace, debug, info, warning, error, fatal, or panic",
		Value:  "info",
		EnvVar: "BACKYARD_LOG_LEVEL",
	},
	cli.BoolFlag{
		Name:   "debug",
		Usage:  "enable debug mode, show additional log messages",
		EnvVar: "BACKYARD_DEBUG",
	},
	cli.BoolFlag{
		Name:   "test",
		Hidden: true,
		Usage:  "enable test mode",
	},
	cli.BoolFlag{
		Name:   "unsafe",
		Hidden: true,
		Usage:  "enable unsafe mode",
		EnvVar: "BACKYARD_UNSAFE",
	},
	cli.BoolFlag{
		Name:   "demo",
		Hidden: true,
		Usage:  "enable demo mode",
		EnvVar: "BACKYARD_DEMO",
	},
	cli.BoolFlag{
		Name:   "sponsor",
		Hidden: true,
		Usage:  "your continuous support helps to pay for development and operating expenses",
		EnvVar: "BACKYARD_SPONSOR",
	},
	cli.BoolFlag{
		Name:   "public, p",
		Usage:  "disable password authentication",
		EnvVar: "BACKYARD_PUBLIC",
	},
	cli.BoolFlag{
		Name:   "read-only, r",
		Usage:  "disable import, upload, and delete",
		EnvVar: "BACKYARD_READONLY",
	},
	cli.BoolFlag{
		Name:   "experimental, e",
		Usage:  "enable experimental features",
		EnvVar: "BACKYARD_EXPERIMENTAL",
	},
	cli.StringFlag{
		Name:   "config-file, c",
		Usage:  "load config options from `FILENAME`",
		EnvVar: "BACKYARD_CONFIG_FILE",
	},
	cli.StringFlag{
		Name:   "config-path",
		Usage:  "config `PATH` to be searched for additional configuration and settings files",
		EnvVar: "BACKYARD_CONFIG_PATH",
	},
	cli.StringFlag{
		Name:   "originals-path",
		Usage:  "original media library `PATH` containing your photo and video collection",
		EnvVar: "BACKYARD_ORIGINALS_PATH",
	},
	cli.IntFlag{
		Name:   "originals-limit",
		Value:  1000,
		Usage:  "original media file size limit in `MB`",
		EnvVar: "BACKYARD_ORIGINALS_LIMIT",
	},
	cli.StringFlag{
		Name:   "import-path",
		Usage:  "optional import `PATH` from which files can be added to originals",
		EnvVar: "BACKYARD_IMPORT_PATH",
	},
	cli.StringFlag{
		Name:   "storage-path",
		Usage:  "writable storage `PATH` for cache, database, and sidecar files",
		EnvVar: "BACKYARD_STORAGE_PATH",
	},
	cli.StringFlag{
		Name:   "cache-path",
		Usage:  "optional custom cache `PATH` for sessions and thumbnail files",
		EnvVar: "BACKYARD_CACHE_PATH",
	},
	cli.StringFlag{
		Name:   "sidecar-path",
		Usage:  "optional custom relative or absolute sidecar `PATH`",
		EnvVar: "BACKYARD_SIDECAR_PATH",
	},
	cli.StringFlag{
		Name:   "temp-path",
		Usage:  "optional custom temporary file `PATH`",
		EnvVar: "BACKYARD_TEMP_PATH",
	},
	cli.StringFlag{
		Name:   "backup-path",
		Usage:  "optional custom backup `PATH` for index backup files",
		EnvVar: "BACKYARD_BACKUP_PATH",
	},
	cli.StringFlag{
		Name:   "assets-path",
		Usage:  "assets `PATH` containing static resources like icons, models, and translations",
		EnvVar: "BACKYARD_ASSETS_PATH",
	},
	cli.IntFlag{
		Name:   "workers, w",
		Usage:  "maximum `NUMBER` of indexing workers, default depends on the number of physical cores",
		EnvVar: "BACKYARD_WORKERS",
		Value:  cpuid.CPU.PhysicalCores / 2,
	},
	cli.IntFlag{
		Name:   "wakeup-interval",
		Usage:  "metadata, share & sync background worker wakeup interval in `SECONDS` (1-604800)",
		Value:  DefaultWakeupIntervalSeconds,
		EnvVar: "BACKYARD_WAKEUP_INTERVAL",
	},
	cli.IntFlag{
		Name:   "auto-index",
		Usage:  "WebDAV auto index safety delay in `SECONDS`, disable with -1",
		Value:  DefaultAutoIndexDelay,
		EnvVar: "BACKYARD_AUTO_INDEX",
	},
	cli.IntFlag{
		Name:   "auto-import",
		Usage:  "WebDAV auto import safety delay in `SECONDS`, disable with -1",
		Value:  DefaultAutoImportDelay,
		EnvVar: "BACKYARD_AUTO_IMPORT",
	},
	cli.BoolFlag{
		Name:   "disable-webdav",
		Usage:  "disable built-in WebDAV server",
		EnvVar: "BACKYARD_DISABLE_WEBDAV",
	},
	cli.BoolFlag{
		Name:   "disable-settings",
		Usage:  "disable settings UI and API",
		EnvVar: "BACKYARD_DISABLE_SETTINGS",
	},
	cli.BoolFlag{
		Name:   "disable-places",
		Usage:  "disable reverse geocoding and maps",
		EnvVar: "BACKYARD_DISABLE_PLACES",
	},
	cli.BoolFlag{
		Name:   "disable-backups",
		Usage:  "disable creating YAML metadata backup files",
		EnvVar: "BACKYARD_DISABLE_BACKUPS",
	},
	cli.BoolFlag{
		Name:   "disable-exiftool",
		Usage:  "disable creating JSON metadata sidecar files with ExifTool",
		EnvVar: "BACKYARD_DISABLE_EXIFTOOL",
	},
	cli.BoolFlag{
		Name:   "disable-ffmpeg",
		Usage:  "disable video transcoding and thumbnail extraction with FFmpeg",
		EnvVar: "BACKYARD_DISABLE_FFMPEG",
	},
	cli.BoolFlag{
		Name:   "disable-darktable",
		Usage:  "disable converting RAW files with Darktable",
		EnvVar: "BACKYARD_DISABLE_DARKTABLE",
	},
	cli.BoolFlag{
		Name:   "disable-rawtherapee",
		Usage:  "disable converting RAW files with RawTherapee",
		EnvVar: "BACKYARD_DISABLE_RAWTHERAPEE",
	},
	cli.BoolFlag{
		Name:   "disable-sips",
		Usage:  "disable converting RAW files with Sips (macOS only)",
		EnvVar: "BACKYARD_DISABLE_SIPS",
	},
	cli.BoolFlag{
		Name:   "disable-heifconvert",
		Usage:  "disable converting HEIC/HEIF files",
		EnvVar: "BACKYARD_DISABLE_HEIFCONVERT",
	},
	cli.BoolFlag{
		Name:   "disable-tensorflow",
		Usage:  "disable all features depending on TensorFlow",
		EnvVar: "BACKYARD_DISABLE_TENSORFLOW",
	},
	cli.BoolFlag{
		Name:   "disable-faces",
		Usage:  "disable facial recognition",
		EnvVar: "BACKYARD_DISABLE_FACES",
	},
	cli.BoolFlag{
		Name:   "disable-classification",
		Usage:  "disable image classification",
		EnvVar: "BACKYARD_DISABLE_CLASSIFICATION",
	},
	cli.BoolFlag{
		Name:   "detect-nsfw",
		Usage:  "flag photos as private that may be offensive (requires TensorFlow)",
		EnvVar: "BACKYARD_DETECT_NSFW",
	},
	cli.BoolFlag{
		Name:   "upload-nsfw",
		Usage:  "allow uploads that may be offensive",
		EnvVar: "BACKYARD_UPLOAD_NSFW",
	},
	cli.StringFlag{
		Name:   "default-theme",
		Usage:  "standard user interface theme `NAME`",
		Hidden: true,
		EnvVar: "BACKYARD_DEFAULT_THEME",
	},

	cli.StringFlag{
		Name:   "app-icon",
		Usage:  "application `ICON` (logo, app, crisp, mint, bold)",
		EnvVar: "BACKYARD_APP_ICON",
	},
	cli.StringFlag{
		Name:   "app-name",
		Usage:  "application `NAME` when installed on a device",
		Value:  "8ackyard",
		EnvVar: "BACKYARD_APP_NAME",
	},
	cli.StringFlag{
		Name:   "app-mode",
		Usage:  "application `MODE` (fullscreen, standalone, minimal-ui, browser)",
		Value:  "standalone",
		EnvVar: "BACKYARD_APP_MODE",
	},
	cli.StringFlag{
		Name:   "cdn-url",
		Usage:  "optional content delivery network `URL`",
		EnvVar: "BACKYARD_CDN_URL",
	},
	cli.StringFlag{
		Name:   "site-url",
		Usage:  "public site `URL`",
		Value:  "http://localhost:2342/",
		EnvVar: "BACKYARD_SITE_URL",
	},
	cli.StringFlag{
		Name:   "site-author",
		Usage:  "`COPYRIGHT`, artist, or owner name",
		EnvVar: "BACKYARD_SITE_AUTHOR",
	},
	cli.StringFlag{
		Name:   "site-title",
		Usage:  "site `TITLE`",
		Value:  "8ackyard",
		EnvVar: "BACKYARD_SITE_TITLE",
	},
	cli.StringFlag{
		Name:   "site-caption",
		Usage:  "site `CAPTION`",
		Value:  "Browse Your Life",
		EnvVar: "BACKYARD_SITE_CAPTION",
	},
	cli.StringFlag{
		Name:   "site-description",
		Usage:  "optional site `DESCRIPTION`",
		EnvVar: "BACKYARD_SITE_DESCRIPTION",
	},
	cli.StringFlag{
		Name:   "site-preview",
		Usage:  "optional preview image `URL`",
		EnvVar: "BACKYARD_SITE_PREVIEW",
	},
	cli.IntFlag{
		Name:   "http-port",
		Value:  2342,
		Usage:  "http server port `NUMBER`",
		EnvVar: "BACKYARD_HTTP_PORT",
	},
	cli.StringFlag{
		Name:   "http-host",
		Usage:  "http server `IP` address",
		EnvVar: "BACKYARD_HTTP_HOST",
	},
	cli.StringFlag{
		Name:   "http-mode, m",
		Usage:  "http server `MODE` (debug, release, or test)",
		EnvVar: "BACKYARD_HTTP_MODE",
	},
	cli.StringFlag{
		Name:   "http-compression, z",
		Usage:  "http server compression `METHOD` (none or gzip)",
		EnvVar: "BACKYARD_HTTP_COMPRESSION",
	},
	cli.StringFlag{
		Name:   "database-driver",
		Usage:  "database `DRIVER` (sqlite or mysql)",
		Value:  "sqlite",
		EnvVar: "BACKYARD_DATABASE_DRIVER",
	},
	cli.StringFlag{
		Name:   "database-dsn",
		Usage:  "sqlite file name, providing a `DSN` is optional for other drivers",
		EnvVar: "BACKYARD_DATABASE_DSN",
	},
	cli.StringFlag{
		Name:   "database-server",
		Usage:  "database server `HOST` with optional port e.g. mysql:3306",
		EnvVar: "BACKYARD_DATABASE_SERVER",
	},
	cli.StringFlag{
		Name:   "database-name",
		Value:  "8ackyard",
		Usage:  "database schema `NAME`",
		EnvVar: "BACKYARD_DATABASE_NAME",
	},
	cli.StringFlag{
		Name:   "database-user",
		Value:  "8ackyard",
		Usage:  "database user `NAME`",
		EnvVar: "BACKYARD_DATABASE_USER",
	},
	cli.StringFlag{
		Name:   "database-password",
		Usage:  "database user `PASSWORD`",
		EnvVar: "BACKYARD_DATABASE_PASSWORD",
	},
	cli.IntFlag{
		Name:   "database-conns",
		Usage:  "maximum `NUMBER` of open database connections",
		EnvVar: "BACKYARD_DATABASE_CONNS",
	},
	cli.IntFlag{
		Name:   "database-conns-idle",
		Usage:  "maximum `NUMBER` of idle database connections",
		EnvVar: "BACKYARD_DATABASE_CONNS_IDLE",
	},
	cli.BoolFlag{
		Name:   "raw-presets",
		Usage:  "enable RAW file converter presets (may reduce performance)",
		EnvVar: "BACKYARD_RAW_PRESETS",
	},
	cli.StringFlag{
		Name:   "darktable-bin",
		Usage:  "Darktable CLI `COMMAND` for RAW file conversion",
		Value:  "darktable-cli",
		EnvVar: "BACKYARD_DARKTABLE_BIN",
	},
	cli.StringFlag{
		Name:   "darktable-blacklist",
		Usage:  "file `EXTENSIONS` incompatible with Darktable",
		Value:  "cr3,dng",
		EnvVar: "BACKYARD_DARKTABLE_BLACKLIST",
	},
	cli.StringFlag{
		Name:   "rawtherapee-bin",
		Usage:  "RawTherapee CLI `COMMAND` for RAW file conversion",
		Value:  "rawtherapee-cli",
		EnvVar: "BACKYARD_RAWTHERAPEE_BIN",
	},
	cli.StringFlag{
		Name:   "rawtherapee-blacklist",
		Usage:  "file `EXTENSIONS` incompatible with RawTherapee",
		Value:  "",
		EnvVar: "BACKYARD_RAWTHERAPEE_BLACKLIST",
	},
	cli.StringFlag{
		Name:   "sips-bin",
		Usage:  "Sips `COMMAND` for RAW file conversion (macOS only)",
		Value:  "sips",
		EnvVar: "BACKYARD_SIPS_BIN",
	},
	cli.StringFlag{
		Name:   "heifconvert-bin",
		Usage:  "HEIC/HEIF image convert `COMMAND`",
		Value:  "heif-convert",
		EnvVar: "BACKYARD_HEIFCONVERT_BIN",
	},
	cli.StringFlag{
		Name:   "ffmpeg-bin",
		Usage:  "FFmpeg `COMMAND` for video transcoding and thumbnail extraction",
		Value:  "ffmpeg",
		EnvVar: "BACKYARD_FFMPEG_BIN",
	},
	cli.StringFlag{
		Name:   "ffmpeg-encoder",
		Usage:  "FFmpeg AVC encoder `NAME`",
		Value:  "libx264",
		EnvVar: "BACKYARD_FFMPEG_ENCODER",
	},
	cli.IntFlag{
		Name:   "ffmpeg-bitrate",
		Usage:  "maximum FFmpeg encoding `BITRATE` (Mbit/s)",
		Value:  50,
		EnvVar: "BACKYARD_FFMPEG_BITRATE",
	},
	cli.IntFlag{
		Name:   "ffmpeg-buffers",
		Usage:  "`NUMBER` of FFmpeg capture buffers",
		Value:  32,
		EnvVar: "BACKYARD_FFMPEG_BUFFERS",
	},
	cli.StringFlag{
		Name:   "exiftool-bin",
		Usage:  "ExifTool `COMMAND` for extracting metadata",
		Value:  "exiftool",
		EnvVar: "BACKYARD_EXIFTOOL_BIN",
	},
	cli.StringFlag{
		Name:   "download-token",
		Usage:  "custom download URL `TOKEN` (default: random)",
		EnvVar: "BACKYARD_DOWNLOAD_TOKEN",
	},
	cli.StringFlag{
		Name:   "preview-token",
		Usage:  "custom thumbnail and streaming URL `TOKEN` (default: random)",
		EnvVar: "BACKYARD_PREVIEW_TOKEN",
	},
	cli.StringFlag{
		Name:   "thumb-filter",
		Usage:  "thumbnail downscaling `FILTER` (best to worst: blackman, lanczos, cubic, linear)",
		Value:  "lanczos",
		EnvVar: "BACKYARD_THUMB_FILTER",
	},
	cli.IntFlag{
		Name:   "thumb-size, s",
		Usage:  "maximum pre-cached thumbnail image size in `PIXELS` (720-7680)",
		Value:  2048,
		EnvVar: "BACKYARD_THUMB_SIZE",
	},
	cli.BoolFlag{
		Name:   "thumb-uncached, u",
		Usage:  "enable on-demand thumbnail generation (high memory and cpu usage)",
		EnvVar: "BACKYARD_THUMB_UNCACHED",
	},
	cli.IntFlag{
		Name:   "thumb-size-uncached, x",
		Usage:  "maximum size of on-demand generated thumbnails in `PIXELS` (720-7680)",
		Value:  7680,
		EnvVar: "BACKYARD_THUMB_SIZE_UNCACHED",
	},
	cli.IntFlag{
		Name:   "jpeg-size",
		Usage:  "maximum size of generated JPEG images in `PIXELS` (720-30000)",
		Value:  7680,
		EnvVar: "BACKYARD_JPEG_SIZE",
	},
	cli.IntFlag{
		Name:   "jpeg-quality, q",
		Usage:  "`QUALITY` of generated JPEG images, a higher value reduces compression (25-100)",
		Value:  92,
		EnvVar: "BACKYARD_JPEG_QUALITY",
	},
	cli.StringFlag{
		Name:   "pid-filename",
		Usage:  "process id `FILENAME` (daemon mode only)",
		EnvVar: "BACKYARD_PID_FILENAME",
	},
	cli.StringFlag{
		Name:   "log-filename",
		Usage:  "server log `FILENAME` (daemon mode only)",
		EnvVar: "BACKYARD_LOG_FILENAME",
		Value:  "",
	},
}
