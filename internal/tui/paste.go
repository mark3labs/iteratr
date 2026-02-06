package tui

import (
	"regexp"
	"strings"
)

// ansiEscapePattern matches ANSI escape sequences including:
// - Control sequences (ESC [ ...)
// - Private sequences (ESC [ ? ...)
// - Cursor control, color codes, etc.
var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// SanitizePaste cleans up pasted content by:
// - Stripping ANSI escape sequences
// - Removing null bytes and non-printable control chars (except \n, \t, \r)
// - Normalizing CRLF (\r\n) to LF (\n)
// - Trimming trailing whitespace
func SanitizePaste(content string) string {
	// Strip ANSI escape sequences
	content = ansiEscapePattern.ReplaceAllString(content, "")

	// Remove null bytes and non-printable control chars (keep \n, \t, \r)
	var result strings.Builder
	for _, r := range content {
		switch {
		case r == 0: // null byte
			continue
		case r >= 1 && r <= 8: // control chars (SOH through BS)
			continue
		case r == 11 || r == 12: // VT, FF
			continue
		case r >= 14 && r <= 31: // control chars (SO through US)
			continue
		case r == 127: // DEL
			continue
		default:
			result.WriteRune(r)
		}
	}
	content = result.String()

	// Normalize CRLF to LF
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Trim trailing whitespace from entire content
	content = strings.TrimRight(content, " \t\n\r")

	return content
}

// newlinePattern matches one or more newline characters
var newlinePattern = regexp.MustCompile(`\n+`)

// collapseNewlines replaces all sequences of newlines with a single space.
// This is used for single-line text inputs where newlines should be collapsed.
func collapseNewlines(content string) string {
	return newlinePattern.ReplaceAllString(content, " ")
}
