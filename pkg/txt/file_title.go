package txt

import (
	"regexp"
	"strings"

	"github.com/njhsi/8ackyard/pkg/fs"
)

var FileTitleRegexp = regexp.MustCompile("[\\p{L}\\-,':&+!?！]{1,}|( [&+] )?")

// FileTitle returns the string with the first characters of each word converted to uppercase.
func FileTitle(s string) string {
	s = fs.BasePrefix(s, true)

	if len(s) < 3 && IsASCII(s) {
		return ""
	}

	words := FileTitleRegexp.FindAllString(s, -1)
	var result []string

	found := 0

	for _, w := range words {
		w = strings.ToLower(w)

		if IsASCII(w) && (len(w) < 3 && found == 0 || len(w) == 1) {
			continue
		}

		if _, ok := StopWords[w]; ok && found == 0 {
			continue
		}

		if UnknownWord(w) {
			continue
		}

		result = append(result, w)

		found++

		if found > 10 {
			break
		}
	}

	if found == 0 {
		return ""
	}

	title := strings.Join(result, " ")

	title = strings.ReplaceAll(title, "--", " / ")
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "  ", " ")

	// Remove small words from title ending.
	for w, _ := range SmallWords {
		title = strings.TrimSuffix(title, " "+w)
	}

	if len(title) <= 4 && IsASCII(title) {
		return ""
	}

	return Title(title)
}
