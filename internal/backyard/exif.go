package backyard

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"gopkg.in/photoprism/go-tz.v2/tz"
)

const MimeVideoMP4 = "video/mp4"
const MimeQuicktime = "video/quicktime"

type Keywords []string

const CodecUnknown = ""
const CodecJpeg = "jpeg"
const CodecAvc1 = "avc1"
const CodecHeic = "heic"
const CodecXMP = "xmp"

// Data represents image meta data.
type ExifData struct {
	MIMEType     string        `meta:"MIMEType"`
	DocumentID   string        `meta:"ImageUniqueID,OriginalDocumentID,DocumentID"`
	InstanceID   string        `meta:"InstanceID,DocumentID"`
	TakenAt      time.Time     `meta:"DateTimeOriginal,CreationDate,CreateDate,MediaCreateDate,ContentCreateDate,DateTimeDigitized,DateTime"`
	TakenAtLocal time.Time     `meta:"DateTimeOriginal,CreationDate,CreateDate,MediaCreateDate,ContentCreateDate,DateTimeDigitized,DateTime"`
	TimeZone     string        `meta:"-"`
	Duration     time.Duration `meta:"Duration,MediaDuration,TrackDuration"`
	Codec        string        `meta:"CompressorID,Compression,FileType"`
	Title        string        `meta:"Title"`
	Subject      string        `meta:"Subject,PersonInImage,ObjectName,HierarchicalSubject,CatalogSets"`
	Notes        string        `meta:"-"`
	Artist       string        `meta:"Artist,Creator,OwnerName"`
	Description  string        `meta:"Description"`
	Copyright    string        `meta:"Rights,Copyright"`
	Projection   string        `meta:"ProjectionType"`
	ColorProfile string        `meta:"ICCProfileName,ProfileDescription"`
	CameraMake   string        `meta:"CameraMake,Make"`
	CameraModel  string        `meta:"CameraModel,Model"`
	CameraOwner  string        `meta:"OwnerName"`
	CameraSerial string        `meta:"SerialNumber"`
	LensMake     string        `meta:"LensMake"`
	LensModel    string        `meta:"Lens,LensModel"`
	Flash        bool          `meta:"-"`
	FocalLength  int           `meta:"FocalLength"`
	Exposure     string        `meta:"ExposureTime"`
	Aperture     float32       `meta:"ApertureValue"`
	FNumber      float32       `meta:"FNumber"`
	Iso          int           `meta:"ISO"`
	GPSPosition  string        `meta:"GPSPosition"`
	GPSLatitude  string        `meta:"GPSLatitude"`
	GPSLongitude string        `meta:"GPSLongitude"`
	Lat          float32       `meta:"-"`
	Lng          float32       `meta:"-"`
	Altitude     int           `meta:"GlobalAltitude,GPSAltitude"`
	Width        int           `meta:"PixelXDimension,ImageWidth,ExifImageWidth,SourceImageWidth"`
	Height       int           `meta:"PixelYDimension,ImageHeight,ImageLength,ExifImageHeight,SourceImageHeight"`
	Orientation  int           `meta:"-"`
	Rotation     int           `meta:"Rotation"`
	Views        int           `meta:"-"`
	Albums       []string      `meta:"-"`
	Error        error         `meta:"-"`
	All          map[string]string
}

// Exiftool parses JSON sidecar data as created by Exiftool.
func (data *ExifData) DataFromExiftool(jsonData []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("metadata: %s (exiftool panic)\nstack: %s", e, debug.Stack())
		}
	}()

	j := gjson.GetBytes(jsonData, "@flatten|@join")

	if !j.IsObject() {
		return fmt.Errorf("metadata: data is not an object in this file (exiftool)")
	}

	jsonStrings := make(map[string]string)
	jsonValues := j.Map()

	for key, val := range jsonValues {
		jsonStrings[key] = val.String()
	}

	v := reflect.ValueOf(data).Elem()

	// Iterate through all config fields
	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)

		tagData := v.Type().Field(i).Tag.Get("meta")

		// Automatically assign values to fields with "flag" tag
		if tagData != "" {
			tagValues := strings.Split(tagData, ",")

			var jsonValue gjson.Result
			var tagValue string

			for _, tagValue = range tagValues {
				if r, ok := jsonValues[tagValue]; !ok {
					continue
				} else {
					jsonValue = r
					break
				}
			}

			// Skip empty values.
			if !jsonValue.Exists() {
				continue
			}

			switch fieldValue.Interface().(type) {
			case time.Time:
				if !fieldValue.IsZero() {
					continue
				}

				s := strings.TrimSpace(jsonValue.String())
				s = strings.ReplaceAll(s, "/", ":")

				if tv, err := time.Parse("2006:01:02 15:04:05", strings.ReplaceAll(s, "-", ":")); err == nil {
					fieldValue.Set(reflect.ValueOf(tv.Round(time.Second).UTC()))
				} else if tv, err := time.Parse("2006:01:02 15:04:05-07:00", s); err == nil {
					fieldValue.Set(reflect.ValueOf(tv.Round(time.Second)))
				}
			case time.Duration:
				if !fieldValue.IsZero() {
					continue
				}

				fieldValue.Set(reflect.ValueOf(StringToDuration(jsonValue.String())))
			case int, int64:
				if !fieldValue.IsZero() {
					continue
				}

				fieldValue.SetInt(jsonValue.Int())
			case float32, float64:
				if !fieldValue.IsZero() {
					continue
				}

				fieldValue.SetFloat(jsonValue.Float())
			case uint, uint64:
				if !fieldValue.IsZero() {
					continue
				}

				fieldValue.SetUint(jsonValue.Uint())
			case []string:
				//TODO
			case Keywords:
				//TODO
			case string:
				if !fieldValue.IsZero() {
					continue
				}

				fieldValue.SetString(strings.TrimSpace(jsonValue.String()))
			case bool:
				if !fieldValue.IsZero() {
					continue
				}

				fieldValue.SetBool(jsonValue.Bool())
			default:
				//debug.Log("metadata: can't assign value of type %s to %s (exiftool)", t, tagValue)
			}
		}
	}
	/*
		// Set latitude and longitude if known and not already set.
		if data.Lat == 0 && data.Lng == 0 {
			if data.GPSPosition != "" {
				data.Lat, data.Lng = GpsToLatLng(data.GPSPosition)
			} else if data.GPSLatitude != "" && data.GPSLongitude != "" {
				data.Lat = GpsToDecimal(data.GPSLatitude)
				data.Lng = GpsToDecimal(data.GPSLongitude)
			}
		}

		if data.Altitude == 0 {
			// Parseable floating point number?
			if fl := GpsFloatRegexp.FindAllString(jsonStrings["GPSAltitude"], -1); len(fl) != 1 {
				// Ignore.
			} else if alt, err := strconv.ParseFloat(fl[0], 64); err == nil && alt != 0 {
				data.Altitude = int(alt)
			}
		}
	*/

	hasTimeOffset := false

	if _, offset := data.TakenAtLocal.Zone(); offset != 0 && !data.TakenAtLocal.IsZero() {
		hasTimeOffset = true
	} else if mt, ok := jsonStrings["MIMEType"]; ok && (mt == MimeVideoMP4 || mt == MimeQuicktime) {
		// Assume default time zone for MP4 & Quicktime videos is UTC.
		// see https://exiftool.org/TagNames/QuickTime.html
		data.TimeZone = time.UTC.String()
		data.TakenAt = data.TakenAt.UTC()
		data.TakenAtLocal = time.Time{}
	}

	// Set time zone and calculate UTC time.
	if data.Lat != 0 && data.Lng != 0 {
		zones, err := tz.GetZone(tz.Point{
			Lat: float64(data.Lat),
			Lon: float64(data.Lng),
		})

		if err == nil && len(zones) > 0 {
			data.TimeZone = zones[0]
		}

		if loc, err := time.LoadLocation(data.TimeZone); err != nil {
			//			log.Warnf("metadata: unknown time zone %s (exiftool)", data.TimeZone)
		} else if !data.TakenAtLocal.IsZero() {
			if tl, err := time.ParseInLocation("2006:01:02 15:04:05", data.TakenAtLocal.Format("2006:01:02 15:04:05"), loc); err == nil {
				if localUtc, err := time.ParseInLocation("2006:01:02 15:04:05", data.TakenAtLocal.Format("2006:01:02 15:04:05"), time.UTC); err == nil {
					data.TakenAtLocal = localUtc
				}

				data.TakenAt = tl.Round(time.Second).UTC()
			} else {
				//	log.Errorf("metadata: %s (exiftool)", err.Error()) // this should never happen
			}
		} else if !data.TakenAt.IsZero() {
			if localUtc, err := time.ParseInLocation("2006:01:02 15:04:05", data.TakenAt.In(loc).Format("2006:01:02 15:04:05"), time.UTC); err == nil {
				data.TakenAtLocal = localUtc
				data.TakenAt = data.TakenAt.UTC()
			} else {
				//				log.Errorf("metadata: %s (exiftool)", err.Error()) // this should never happen
			}
		}
	} else if hasTimeOffset {
		if localUtc, err := time.ParseInLocation("2006:01:02 15:04:05", data.TakenAtLocal.Format("2006:01:02 15:04:05"), time.UTC); err == nil {
			data.TakenAtLocal = localUtc
		}

		data.TakenAt = data.TakenAt.Round(time.Second).UTC()
	}

	// Set local time if still empty.
	if data.TakenAtLocal.IsZero() && !data.TakenAt.IsZero() {
		if loc, err := time.LoadLocation(data.TimeZone); data.TimeZone == "" || err != nil {
			data.TakenAtLocal = data.TakenAt
		} else if localUtc, err := time.ParseInLocation("2006:01:02 15:04:05", data.TakenAt.In(loc).Format("2006:01:02 15:04:05"), time.UTC); err == nil {
			data.TakenAtLocal = localUtc
			data.TakenAt = data.TakenAt.UTC()
		} else {
			//			log.Errorf("metadata: %s (exiftool)", err.Error()) // this should never happen
		}
	}

	if orientation, ok := jsonStrings["Orientation"]; ok && orientation != "" {
		switch orientation {
		case "1", "Horizontal (normal)":
			data.Orientation = 1
		case "2":
			data.Orientation = 2
		case "3", "Rotate 180 CW":
			data.Orientation = 3
		case "4":
			data.Orientation = 4
		case "5":
			data.Orientation = 5
		case "6", "Rotate 90 CW":
			data.Orientation = 6
		case "7":
			data.Orientation = 7
		case "8", "Rotate 270 CW":
			data.Orientation = 8
		}
	}

	if data.Orientation == 0 {
		// Set orientation based on rotation.
		switch data.Rotation {
		case 0:
			data.Orientation = 1
		case -180, 180:
			data.Orientation = 3
		case 90:
			data.Orientation = 6
		case -90, 270:
			data.Orientation = 8
		}
	}

	// Normalize compression information.
	data.Codec = strings.ToLower(data.Codec)
	if strings.Contains(data.Codec, CodecJpeg) {
		data.Codec = CodecJpeg
	}

	// Validate and normalize optional DocumentID.

	// Validate and normalize optional InstanceID.

	data.Title = data.Title
	data.Description = data.Description
	data.Subject = data.Subject
	data.Artist = data.Artist

	return nil
}

var DurationSecondsRegexp = regexp.MustCompile("[0-9\\.]+")

// StringToDuration converts a metadata string to a valid duration.
func StringToDuration(s string) (d time.Duration) {
	if s == "" {
		return d
	}

	s = strings.TrimSpace(s)
	sec := DurationSecondsRegexp.FindAllString(s, -1)

	if len(sec) == 1 {
		secFloat, _ := strconv.ParseFloat(sec[0], 64)
		d = time.Duration(secFloat) * time.Second
	} else if n := strings.Split(s, ":"); len(n) == 3 {
		h, _ := strconv.Atoi(n[0])
		m, _ := strconv.Atoi(n[1])
		s, _ := strconv.Atoi(n[2])

		d = time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second
	} else if pd, err := time.ParseDuration(s); err != nil {
		d = pd
	}

	return d
}
