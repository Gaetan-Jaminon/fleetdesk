package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configDir = ".config/fleetdesk"

// ConfigPath returns the full path to the fleetdesk config directory.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, configDir)
}

// ScanFleets reads all .yaml files from the config directory (excluding config.yaml)
// and returns parsed fleet definitions.
func ScanFleets() ([]Fleet, error) {
	dir := ConfigPath()
	if dir == "" {
		return nil, fmt.Errorf("cannot determine home directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading config dir: %w", err)
	}

	var fleets []Fleet
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			continue
		}
		if name == "config.yaml" || name == "config.yml" {
			continue
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(dir, name)
		f, err := ParseFleetFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		fleets = append(fleets, f)
	}

	return fleets, nil
}
