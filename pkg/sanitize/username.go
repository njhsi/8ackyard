package sanitize

import (
	"strings"
)

// Username returns the normalized username (lowercase, whitespace trimmed).
func Username(s string) string {
	s = strings.TrimSpace(s)

	if s == "" {
		return ""
	}

	return strings.ToLower(s)
}
