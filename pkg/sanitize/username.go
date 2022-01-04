package sanitize

import (
	"strings"

	"github.com/njhsi/8ackyard/pkg/txt"
)

// Username returns the normalized username (lowercase, whitespace trimmed).
func Username(s string) string {
	s = strings.TrimSpace(s)

	if s == "" || reject(s, txt.ClipUsername) {
		return ""
	}

	return strings.ToLower(s)
}
