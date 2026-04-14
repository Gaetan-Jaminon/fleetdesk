package config

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
	Name             string          `yaml:"name"`
	Type             string          `yaml:"type"`
	TenantID         string          `yaml:"tenant_id"`
	ActivityLogHours int             `yaml:"activity_log_hours"`
	DisplayTags      []string        `yaml:"display_tags"`
	Defaults         defaultsFile    `yaml:"defaults"`
	Groups           []groupFile     `yaml:"groups"`
	Hosts            []hostEntryFile `yaml:"hosts"`
}

type defaultsFile struct {
	User            string   `yaml:"user"`
	Port            int      `yaml:"port"`
	Timeout         string   `yaml:"timeout"`
	SystemdMode     string   `yaml:"systemd_mode"`
	ServiceFilter   []string `yaml:"service_filter"`
	ErrorLogSince   string   `yaml:"error_log_since"`
	RefreshInterval string   `yaml:"refresh_interval"`
	RHOrgID         string   `yaml:"rh_org_id"`
	RHActivationKey string   `yaml:"rh_activation_key"`
	SatelliteURL    string   `yaml:"satellite_url"`
}

type groupFile struct {
	Name          string          `yaml:"name"`
	Hosts         []hostEntryFile `yaml:"hosts"`
	ServiceFilter []string        `yaml:"service_filter"`
}

type hostEntryFile struct {
	Name            string   `yaml:"name"`
	Hostname        string   `yaml:"hostname"`
	User            string   `yaml:"user"`
	Port            int      `yaml:"port"`
	Timeout         string   `yaml:"timeout"`
	SystemdMode     string   `yaml:"systemd_mode"`
	ServiceFilter   []string `yaml:"service_filter"`
	RHOrgID         string   `yaml:"rh_org_id"`
	RHActivationKey string   `yaml:"rh_activation_key"`
	SatelliteURL    string   `yaml:"satellite_url"`
}

// ParseFleetFile reads and parses a single fleet YAML file.
func ParseFleetFile(path string) (Fleet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Fleet{}, fmt.Errorf("reading file: %w", err)
	}

	var raw fleetFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Fleet{}, fmt.Errorf("parsing YAML: %w", err)
	}

	// default name from filename
	name := raw.Name
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// validate type
	switch raw.Type {
	case "vm", "azure", "kubernetes":
		// valid
	case "":
		return Fleet{}, fmt.Errorf("missing required field 'type' in %s", path)
	default:
		return Fleet{}, fmt.Errorf("unknown fleet type %q in %s (must be vm, azure, or kubernetes)", raw.Type, path)
	}

	// parse defaults
	defaults := HostDefaults{
		User:            raw.Defaults.User,
		Port:            raw.Defaults.Port,
		SystemdMode:     raw.Defaults.SystemdMode,
		ServiceFilter:   raw.Defaults.ServiceFilter,
		ErrorLogSince:   raw.Defaults.ErrorLogSince,
		RefreshInterval: raw.Defaults.RefreshInterval,
		RHOrgID:         raw.Defaults.RHOrgID,
		RHActivationKey: raw.Defaults.RHActivationKey,
		SatelliteURL:    raw.Defaults.SatelliteURL,
	}
	if defaults.Port == 0 {
		defaults.Port = 22
	}
	if defaults.SystemdMode == "" {
		defaults.SystemdMode = "system"
	}
	if defaults.ErrorLogSince == "" {
		defaults.ErrorLogSince = "1 hour ago"
	}
	if defaults.RefreshInterval == "" {
		defaults.RefreshInterval = "15s"
	}
	if raw.Defaults.Timeout != "" {
		d, err := time.ParseDuration(raw.Defaults.Timeout)
		if err != nil {
			return Fleet{}, fmt.Errorf("invalid timeout %q: %w", raw.Defaults.Timeout, err)
		}
		defaults.Timeout = d
	} else {
		defaults.Timeout = 10 * time.Second
	}

	// parse groups
	var groups []HostGroup
	for _, g := range raw.Groups {
		hosts, err := parseHosts(g.Hosts, defaults, g.ServiceFilter)
		if err != nil {
			return Fleet{}, fmt.Errorf("group %q: %w", g.Name, err)
		}
		groups = append(groups, HostGroup{
			Name:  g.Name,
			Hosts: hosts,
		})
	}

	// parse ungrouped hosts
	hosts, err := parseHosts(raw.Hosts, defaults, nil)
	if err != nil {
		return Fleet{}, fmt.Errorf("hosts: %w", err)
	}

	activityLogHours := raw.ActivityLogHours
	if activityLogHours <= 0 {
		activityLogHours = 3
	}

	return Fleet{
		Name:             name,
		Type:             raw.Type,
		TenantID:         raw.TenantID,
		ActivityLogHours: activityLogHours,
		DisplayTags:      raw.DisplayTags,
		Path:             path,
		Defaults:         defaults,
		Groups:           groups,
		Hosts:            hosts,
	}, nil
}

// parseHosts converts raw host entries, applying defaults where needed.
// groupFilter is the group-level service filter (can be nil).
func parseHosts(raw []hostEntryFile, defaults HostDefaults, groupFilter []string) ([]HostEntry, error) {
	var hosts []HostEntry
	for _, r := range raw {
		if r.Name == "" {
			return nil, fmt.Errorf("host missing required field 'name'")
		}
		if r.Hostname == "" {
			return nil, fmt.Errorf("host %q missing required field 'hostname'", r.Name)
		}

		h := HostEntry{
			Name:            r.Name,
			Hostname:        r.Hostname,
			User:            r.User,
			Port:            r.Port,
			SystemdMode:     r.SystemdMode,
			RHOrgID:         r.RHOrgID,
			RHActivationKey: r.RHActivationKey,
			SatelliteURL:    r.SatelliteURL,
		}

		// service filter: host → group → defaults
		switch {
		case len(r.ServiceFilter) > 0:
			h.ServiceFilter = r.ServiceFilter
		case len(groupFilter) > 0:
			h.ServiceFilter = groupFilter
		default:
			h.ServiceFilter = defaults.ServiceFilter
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
		// RH subscription: if host defines its own org, it's a complete override — don't inherit satellite_url
		if h.RHOrgID != "" {
			if h.RHActivationKey == "" {
				h.RHActivationKey = defaults.RHActivationKey
			}
		} else {
			h.RHOrgID = defaults.RHOrgID
			h.RHActivationKey = defaults.RHActivationKey
			h.SatelliteURL = defaults.SatelliteURL
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
