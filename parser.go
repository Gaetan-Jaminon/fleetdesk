package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// fleetFile is the raw YAML structure of a fleet configuration file.
type fleetFile struct {
	Name     string          `yaml:"name"`
	Type     string          `yaml:"type"`
	Defaults defaultsFile    `yaml:"defaults"`
	Groups   []groupFile     `yaml:"groups"`
	Hosts    []hostEntryFile `yaml:"hosts"`
}

type defaultsFile struct {
	User          string   `yaml:"user"`
	Port          int      `yaml:"port"`
	Timeout       string   `yaml:"timeout"`
	SystemdMode   string   `yaml:"systemd_mode"`
	ServiceFilter []string `yaml:"service_filter"`
}

type groupFile struct {
	Name  string          `yaml:"name"`
	Hosts []hostEntryFile `yaml:"hosts"`
}

type hostEntryFile struct {
	Name        string `yaml:"name"`
	Hostname    string `yaml:"hostname"`
	User        string `yaml:"user"`
	Port        int    `yaml:"port"`
	Timeout     string `yaml:"timeout"`
	SystemdMode string `yaml:"systemd_mode"`
}

// parseFleetFile reads and parses a single fleet YAML file.
func parseFleetFile(path string) (fleet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fleet{}, fmt.Errorf("reading file: %w", err)
	}

	var raw fleetFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fleet{}, fmt.Errorf("parsing YAML: %w", err)
	}

	// default name from filename
	name := raw.Name
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// default type
	ftype := raw.Type
	if ftype == "" {
		ftype = "vm"
	}

	// parse defaults
	defaults := hostDefaults{
		User:          raw.Defaults.User,
		Port:          raw.Defaults.Port,
		SystemdMode:   raw.Defaults.SystemdMode,
		ServiceFilter: raw.Defaults.ServiceFilter,
	}
	if defaults.Port == 0 {
		defaults.Port = 22
	}
	if defaults.SystemdMode == "" {
		defaults.SystemdMode = "system"
	}
	if raw.Defaults.Timeout != "" {
		d, err := time.ParseDuration(raw.Defaults.Timeout)
		if err != nil {
			return fleet{}, fmt.Errorf("invalid timeout %q: %w", raw.Defaults.Timeout, err)
		}
		defaults.Timeout = d
	} else {
		defaults.Timeout = 10 * time.Second
	}

	// parse groups
	var groups []hostGroup
	for _, g := range raw.Groups {
		hosts, err := parseHosts(g.Hosts, defaults)
		if err != nil {
			return fleet{}, fmt.Errorf("group %q: %w", g.Name, err)
		}
		groups = append(groups, hostGroup{
			Name:  g.Name,
			Hosts: hosts,
		})
	}

	// parse ungrouped hosts
	hosts, err := parseHosts(raw.Hosts, defaults)
	if err != nil {
		return fleet{}, fmt.Errorf("hosts: %w", err)
	}

	return fleet{
		Name:     name,
		Type:     ftype,
		Path:     path,
		Defaults: defaults,
		Groups:   groups,
		Hosts:    hosts,
	}, nil
}

// parseHosts converts raw host entries, applying defaults where needed.
func parseHosts(raw []hostEntryFile, defaults hostDefaults) ([]hostEntry, error) {
	var hosts []hostEntry
	for _, r := range raw {
		if r.Name == "" {
			return nil, fmt.Errorf("host missing required field 'name'")
		}
		if r.Hostname == "" {
			return nil, fmt.Errorf("host %q missing required field 'hostname'", r.Name)
		}

		h := hostEntry{
			Name:     r.Name,
			Hostname: r.Hostname,
			User:     r.User,
			Port:     r.Port,
			SystemdMode: r.SystemdMode,
		}

		// apply defaults
		if h.User == "" {
			h.User = defaults.User
		}
		if h.Port == 0 {
			h.Port = defaults.Port
		}
		if h.SystemdMode == "" {
			h.SystemdMode = defaults.SystemdMode
		}

		if r.Timeout != "" {
			d, err := time.ParseDuration(r.Timeout)
			if err != nil {
				return nil, fmt.Errorf("host %q invalid timeout: %w", r.Name, err)
			}
			h.Timeout = d
		} else {
			h.Timeout = defaults.Timeout
		}

		hosts = append(hosts, h)
	}
	return hosts, nil
}
