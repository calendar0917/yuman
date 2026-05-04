package manager

import (
	"errors"
	"strings"
)

// errNotImplemented is returned by install/remove/upgrade operations that
// are not yet wired to real subprocess calls. This is intentional in the
// MVP to prevent accidental system changes during development.
var errNotImplemented = errors.New("operation not implemented (MVP)")

// splitLines splits a byte slice into trimmed non-empty lines.
func splitLines(s string) []string {
	var lines []string
	for _, l := range splitByNewline(s) {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func splitByNewline(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
