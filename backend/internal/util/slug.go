package util

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug converts a string into a WordPress-style slug:
// diacritics are stripped, the result is lower-cased, non-alphanumeric
// characters are collapsed into single hyphens, and leading/trailing
// hyphens are trimmed.
//
//	GenerateSlug("Ikan Bakar!")  →  "ikan-bakar"
//	GenerateSlug("Café au lait") →  "cafe-au-lait"
func GenerateSlug(s string) string {
	// Strip diacritics via NFD decomposition.
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r) // Mn: non-spacing marks
	}), norm.NFC)
	result, _, _ := transform.String(t, s)

	result = strings.ToLower(result)
	result = nonAlphanumeric.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	return result
}
