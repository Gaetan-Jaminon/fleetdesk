package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configDir = ".config/fleetdesk"

// configPath returns the full path to the fleetdesk config directory.
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, configDir)
}

// scanFleets reads all .yaml files from the config directory (excluding config.yaml)
// and returns parsed fleet definitions.
func scanFleets() ([]fleet, error) {
	dir := configPath()
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

	var fleets []fleet
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
		f, err := parseFleetFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		fleets = append(fleets, f)
	}

	return fleets, nil
}
