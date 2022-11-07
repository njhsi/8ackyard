package backyard

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dateRegexp = regexp.MustCompile("\\D\\d{4}[\\-_]\\d{2}[\\-_]\\d{2,}")
var datePathRegexp = regexp.MustCompile("\\D\\d{4}/\\d{1,2}/?\\d*")
var dateTimeRegexp = regexp.MustCompile("\\D\\d{4}[\\-_]\\d{2}[\\-_]\\d{2}.{1,4}\\d{2}\\D\\d{2}\\D\\d{2,}")
var dateIntRegexp = regexp.MustCompile("\\d{1,4}")
var yearRegexp = regexp.MustCompile("\\d{4,5}")
var isDateRegexp = regexp.MustCompile("\\d{4}[\\-_]?\\d{2}[\\-_]?\\d{2}")
var isDateTimeRegexp = regexp.MustCompile("\\d{4}[\\-_]?\\d{2}[\\-_]?\\d{2}.{1,4}\\d{2}\\D?\\d{2}\\D?\\d{2}")

var dateRegexp2 = regexp.MustCompile("(202[0-9]|201[0-9]|200[0-9]|[0-1][0-9]{3})(1[0-2]|0[1-9])(3[01]|[0-2][1-9]|[12]0)")

var (
	YearMin = 1990
	YearMax = time.Now().Year() + 3
)

const (
	MonthMin = 1
	MonthMax = 12
	DayMin   = 1
	DayMax   = 31
	HourMin  = 0
	HourMax  = 24
	MinMin   = 0
	MinMax   = 59
	SecMin   = 0
	SecMax   = 59
)

// Time returns a string as time or the zero time instant in case it can not be converted.
func TimeFromFileName(s string) (result time.Time) {
	defer func() {
		if r := recover(); r != nil {
			result = time.Time{}
		}
	}()

	if len(s) < 6 {
		return time.Time{}
	}

	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}

	b := []byte(s)

	if found := dateTimeRegexp.Find(b); len(found) > 0 { // Is it a date with time like "2020-01-30_09-57-18"?
		n := dateIntRegexp.FindAll(found, -1)

		if len(n) != 6 {
			return result
		}

		year := convInt(string(n[0]))
		month := convInt(string(n[1]))
		day := convInt(string(n[2]))
		hour := convInt(string(n[3]))
		min := convInt(string(n[4]))
		sec := convInt(string(n[5]))

		if year < YearMin || year > YearMax || month < MonthMin || month > MonthMax || day < DayMin || day > DayMax {
			return result
		}

		if hour < HourMin || hour > HourMax || min < MinMin || min > MinMax || sec < SecMin || sec > SecMax {
			return result
		}

		result = time.Date(
			year,
			time.Month(month),
			day,
			hour,
			min,
			sec,
			0,
			time.UTC)

	} else if found := dateRegexp.Find(b); len(found) > 0 { // Is it a date only like "2020-01-30"?
		n := dateIntRegexp.FindAll(found, -1)

		if len(n) != 3 {
			return result
		}

		year := convInt(string(n[0]))
		month := convInt(string(n[1]))
		day := convInt(string(n[2]))

		if year < YearMin || year > YearMax || month < MonthMin || month > MonthMax || day < DayMin || day > DayMax {
			return result
		}

		result = time.Date(
			year,
			time.Month(month),
			day,
			0,
			0,
			0,
			0,
			time.UTC)
	} else if found := datePathRegexp.Find(b); len(found) > 0 { // Is it a date path like "2020/01/03"?
		n := dateIntRegexp.FindAll(found, -1)

		if len(n) < 2 || len(n) > 3 {
			return result
		}

		year := convInt(string(n[0]))
		month := convInt(string(n[1]))

		if year < YearMin || year > YearMax || month < MonthMin || month > MonthMax {
			return result
		}

		if len(n) == 2 {
			result = time.Date(
				year,
				time.Month(month),
				1,
				0,
				0,
				0,
				0,
				time.UTC)
		} else if day := convInt(string(n[2])); day >= DayMin && day <= DayMax {
			result = time.Date(
				year,
				time.Month(month),
				day,
				0,
				0,
				0,
				0,
				time.UTC)
		}
	} else if found := dateRegexp2.Find(b); len(found) == 8 { // Is it a date like "20200103"?
		year := convInt(string(found[0:4]))
		month := convInt(string(found[4:6]))
		day := convInt(string(found[6:8]))

		if year < YearMin || year > YearMax || month < MonthMin || month > MonthMax || day < DayMin || day > DayMax {
			return result
		}

		result = time.Date(
			year,
			time.Month(month),
			day,
			0,
			0,
			0,
			0,
			time.UTC)

	}

	return result.UTC()
}

// IsTime tests if the string looks like a date and/or time.
func IsTime(s string) bool {
	if s == "" {
		return false
	} else if m := isDateRegexp.FindString(s); m == s {
		return true
	} else if m := isDateTimeRegexp.FindString(s); m == s {
		return true
	}

	return false
}

// Year tries to find a matching year for a given string e.g. from a file oder directory name.
func Year(s string) int {
	b := []byte(s)

	found := yearRegexp.FindAll(b, -1)

	for _, match := range found {
		year := convInt(string(match))

		if year > YearMin && year < YearMax {
			return year
		}
	}

	return 0
}

// /////
// Int converts a string to a signed integer or 0 if invalid.
func convInt(s string) int {
	if s == "" {
		return 0
	}

	result, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)

	if err != nil {
		return 0
	}

	return int(result)
}
