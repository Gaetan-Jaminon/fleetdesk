package main

import "time"

// fleet represents a parsed fleet configuration file.
type fleet struct {
	Name     string       `yaml:"name"`
	Type     string       `yaml:"type"`
	Path     string       `yaml:"-"` // path to the fleet file on disk
	Defaults hostDefaults `yaml:"defaults"`
	Groups   []hostGroup  `yaml:"groups"`
	Hosts    []hostEntry  `yaml:"hosts"`
}

// hostDefaults holds default values applied to all hosts in a fleet.
type hostDefaults struct {
	User          string        `yaml:"user"`
	Port          int           `yaml:"port"`
	Timeout       time.Duration `yaml:"timeout"`
	SystemdMode   string        `yaml:"systemd_mode"`
	ServiceFilter []string      `yaml:"service_filter"`
}

// hostGroup provides visual grouping of hosts.
type hostGroup struct {
	Name  string      `yaml:"name"`
	Hosts []hostEntry `yaml:"hosts"`
}

// hostEntry represents a single host definition in a fleet file.
type hostEntry struct {
	Name          string        `yaml:"name"`
	Hostname      string        `yaml:"hostname"`
	User          string        `yaml:"user"`
	Port          int           `yaml:"port"`
	Timeout       time.Duration `yaml:"timeout"`
	SystemdMode   string        `yaml:"systemd_mode"`
	ServiceFilter []string
}

// host is the runtime representation of a host with connection state.
type host struct {
	Entry    hostEntry
	Group    string
	Status   hostStatus
	NeedsPassword bool // true if key auth failed

	// probe results
	FQDN             string
	OS               string
	UpSince          string
	ServiceCount     int
	ServiceRunning   int
	ServiceFailed    int
	ContainerCount   int
	ContainerRunning int
	LastUpdate       string
	LastSecurity     string
	Error            string
}

type hostStatus int

const (
	hostConnecting hostStatus = iota
	hostOnline
	hostUnreachable
)

// service represents a systemd unit on a remote host.
type service struct {
	Name        string
	State       string // active, inactive, failed
	Enabled     string // enabled, disabled, static
	Description string
}

// container represents a Podman container on a remote host.
type container struct {
	Name   string
	Image  string
	Status string // running, exited, etc.
	ID     string
}
