// Package fspath provides helpers for building safe filesystem paths from
// user- or config-provided strings such as fleet names, hostnames, and
// resource identifiers.
package fspath

import "strings"

// Sanitize replaces characters unsafe for directory or file names:
// forward/backward slashes, spaces, and colons are mapped to safe
// alternatives. Intended for individual path components — callers join
// sanitized components with filepath.Join.
func Sanitize(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", " ", "-", ":", "_")
	return r.Replace(s)
}
