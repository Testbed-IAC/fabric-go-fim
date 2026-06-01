// Package fabtime parses and formats FABRIC orchestrator lease timestamps.
package fabtime

import (
	"fmt"
	"time"
)

// Layout is the FABRIC orchestrator wire format documented in swagger.
const Layout = "2006-01-02 15:04:05 -07:00"

// Examples is a concise user-facing list of accepted timestamp examples.
const Examples = `2026-05-30 19:04:54 +00:00 or 2026-05-30T19:04:54Z`

const legacyLayout = "2006-01-02 15:04:05 -0700"

// Parse parses a FABRIC orchestrator timestamp or RFC3339 timestamp.
func Parse(s string) (time.Time, error) {
	layouts := []string{Layout, legacyLayout, time.RFC3339, time.RFC3339Nano}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse FABRIC time %q: %w", s, ErrInvalidTime)
}

// Format formats t in the FABRIC orchestrator wire format in UTC.
func Format(t time.Time) string {
	return t.UTC().Format(Layout)
}

// Canonical parses s and returns it formatted in the FABRIC orchestrator wire
// format. An empty string stays empty.
func Canonical(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	t, err := Parse(s)
	if err != nil {
		return "", fmt.Errorf("canonicalizing fabric time: %w", err)
	}
	return Format(t), nil
}
