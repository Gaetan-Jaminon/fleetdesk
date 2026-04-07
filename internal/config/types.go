package config

import "time"

// Fleet represents a parsed fleet configuration file.
type Fleet struct {
	Name     string       `yaml:"name"`
	Type     string       `yaml:"type"`
	Path     string       `yaml:"-"`
	Defaults HostDefaults `yaml:"defaults"`
	Groups   []HostGroup  `yaml:"groups"`
	Hosts    []HostEntry  `yaml:"hosts"`
}

// HostDefaults holds default values applied to all hosts in a fleet.
type HostDefaults struct {
	User            string        `yaml:"user"`
	Port            int           `yaml:"port"`
	Timeout         time.Duration `yaml:"timeout"`
	SystemdMode     string        `yaml:"systemd_mode"`
	ServiceFilter   []string      `yaml:"service_filter"`
	ErrorLogSince   string
	RefreshInterval string
}

// HostGroup provides visual grouping of hosts.
type HostGroup struct {
	Name  string      `yaml:"name"`
	Hosts []HostEntry `yaml:"hosts"`
}

// HostEntry represents a single host definition in a fleet file.
type HostEntry struct {
	Name          string        `yaml:"name"`
	Hostname      string        `yaml:"hostname"`
	User          string        `yaml:"user"`
	Port          int           `yaml:"port"`
	Timeout       time.Duration `yaml:"timeout"`
	SystemdMode   string        `yaml:"systemd_mode"`
	ServiceFilter []string
}

// Host is the runtime representation of a host with connection state.
type Host struct {
	Entry         HostEntry
	Group         string
	Status        HostStatus
	NeedsPassword bool
	ErrorLogSince string

	// probe results
	FQDN             string
	OS               string
	UpSince          string
	ServiceCount     int
	ServiceRunning   int
	ServiceFailed    int
	ContainerCount   int
	ContainerRunning int
	CronCount        int
	ErrorCount       int
	UpdateCount      int
	DiskCount        int
	DiskHighCount    int
	UserCount        int
	LockedUsers      int
	InterfacesUp     int
	InterfacesTotal  int
	ListeningPorts   int
	LastUpdate       string
	LastSecurity     string
	Error            string
}

// HostStatus represents the connection state of a host.
type HostStatus int

const (
	HostConnecting HostStatus = iota
	HostOnline
	HostUnreachable
)

// Service represents a systemd unit on a remote host.
type Service struct {
	Name        string
	State       string
	Enabled     string
	Description string
}

// Container represents a Podman container on a remote host.
type Container struct {
	Name   string
	Image  string
	Status string
	ID     string
}

// CronJob represents a scheduled task.
type CronJob struct {
	Schedule string
	Command  string
	Source   string
}

// LogLevelEntry represents a log severity level with its count.
type LogLevelEntry struct {
	Level string
	Code  string
	Count int
}

// ErrorLog represents a journal error entry.
type ErrorLog struct {
	Time    string
	Unit    string
	Message string
}

// Update represents a pending package update.
type Update struct {
	Package string
	Version string
	Type    string
}

// Subscription represents a key-value pair from subscription-manager.
type Subscription struct {
	Field string
	Value string
}

// Disk represents a filesystem partition.
type Disk struct {
	Filesystem string
	Size       string
	Used       string
	Avail      string
	UsePercent string
	Mount      string
}

// Account represents a local user account on a remote host.
type Account struct {
	User           string
	UID            int
	Groups         string
	Shell          string
	LastLogin      string
	PasswordStatus string // PS (set), LK (locked), NP (no password)
	Expiry         string
	IsSudo         bool // true if groups contains "wheel" or "sudo"
	IsLocked       bool // true if PasswordStatus == "LK" or "L"
}

// NetInterface represents a network interface on a remote host.
type NetInterface struct {
	Name  string
	State string // UP, DOWN, UNKNOWN
	IPs   string // space-separated, CIDR stripped
	MTU   string
}

// ListeningPort represents a listening TCP port on a remote host.
type ListeningPort struct {
	Port        int
	Protocol    string
	Process     string
	BindAddress string
}

// Route represents a network route on a remote host.
type Route struct {
	Destination string
	Gateway     string
	Interface   string
	Metric      string
	IsDefault   bool
}

// FirewallRule represents a firewall rule from any backend.
type FirewallRule struct {
	Zone     string // zone name (firewalld) or chain name (iptables/nft)
	Service  string // service name or port/proto
	Protocol string // tcp, udp, or —
	Source   string // source IP or —
	Action   string // allow, drop, reject
	Backend  string // firewalld, nftables, iptables
}
