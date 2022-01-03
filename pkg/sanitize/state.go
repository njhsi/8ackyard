package sanitize

import (
	"strings"
	"unicode"
)

// State returns the full, normalized state name.
func State(s, countryCode string) string {
	if s == "" {
		return Empty
	}

	// Remove whitespace from name.
	s = strings.TrimSpace(s)

	// Empty?
	if s == "" {
		// State doesn't have a name.
		return ""
	}

	// Remove non-printable and other potentially problematic characters.
	s = strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return -1
		}

		switch r {
		case '~', '\\', ':', '|', '"', '?', '*', '<', '>', '{', '}':
			return -1
		default:
			return r
		}
	}, s)

	// Normalize country code.
	countryCode = strings.ToLower(strings.TrimSpace(countryCode))

	// Is the name an abbreviation that should be normalized?

	// Return normalized state name.
	return s
}
