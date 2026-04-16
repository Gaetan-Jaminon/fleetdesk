package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MergeLogEntries merges log entries from defaults, group, and host levels.
// Dedupe by name: host overrides group, group overrides defaults.
// New names from any level are added (additive catalog, not replace).
func MergeLogEntries(defaults, group, host []LogEntry) []LogEntry {
	seen := make(map[string]int) // name → index in result
	var result []LogEntry

	// Seed with defaults
	for _, e := range defaults {
		seen[e.Name] = len(result)
		result = append(result, e)
	}

	// Overlay group: add new, override existing
	for _, e := range group {
		if idx, ok := seen[e.Name]; ok {
			result[idx] = e
		} else {
			seen[e.Name] = len(result)
			result = append(result, e)
		}
	}

	// Overlay host: add new, override existing
	for _, e := range host {
		if idx, ok := seen[e.Name]; ok {
			result[idx] = e
		} else {
			seen[e.Name] = len(result)
			result = append(result, e)
		}
	}

	return result
}

// shellMetachars is the set of characters not allowed in log paths.
const shellMetachars = "$`;&|<>\n\r\\(){}[]'"

// ValidateLogPath checks that a log path is absolute and contains no shell metacharacters.
func ValidateLogPath(name, path string) error {
	if path == "" {
		return fmt.Errorf("log %q: path is empty", name)
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("log %q: path must be absolute, got %q", name, path)
	}
	if len(path) > 512 {
		return fmt.Errorf("log %q: path too long (%d chars, max 512)", name, len(path))
	}
	for _, c := range shellMetachars {
		if strings.ContainsRune(path, c) {
			return fmt.Errorf("log %q: path contains illegal character %q", name, string(c))
		}
	}
	if filepath.Clean(path) != path {
		return fmt.Errorf("log %q: path must be canonical (no .. components), got %q", name, path)
	}
	return nil
}
