package store

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func buildSlug(name, explicit string) string {
	source := strings.TrimSpace(explicit)
	if source == "" {
		source = strings.TrimSpace(name)
	}

	source = strings.ToLower(source)
	source = nonSlugChars.ReplaceAllString(source, "-")
	source = strings.Trim(source, "-")
	if source == "" {
		return newHexID()
	}

	return source
}

func newHexID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}

	return hex.EncodeToString(buf)
}
