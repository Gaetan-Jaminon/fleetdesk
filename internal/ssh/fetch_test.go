package ssh

import (
	"testing"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func TestServiceStateOrder(t *testing.T) {
	tests := []struct {
		state string
		want  int
	}{
		{"failed", 0},
		{"running", 1},
		{"exited", 2},
		{"waiting", 3},
		{"inactive", 4},
		{"unknown", 5},
		{"", 5},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := ServiceStateOrder(tt.state)
			if got != tt.want {
				t.Errorf("ServiceStateOrder(%q) = %d, want %d", tt.state, got, tt.want)
			}
		})
	}

	// verify ordering: failed < running < exited < waiting < inactive
	if ServiceStateOrder("failed") >= ServiceStateOrder("running") {
		t.Error("failed should sort before running")
	}
	if ServiceStateOrder("running") >= ServiceStateOrder("inactive") {
		t.Error("running should sort before inactive")
	}
}

func TestContainerStateOrder(t *testing.T) {
	tests := []struct {
		status string
		want   int
	}{
		{"Up 3 hours", 0},
		{"Up 5 minutes", 0},
		{"Exited (0) 2 hours ago", 1},
		{"Exited (1) 5 minutes ago", 1},
		{"Created", 2},
		{"", 2},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := ContainerStateOrder(tt.status)
			if got != tt.want {
				t.Errorf("ContainerStateOrder(%q) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}

func TestParseServiceLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want config.Service
	}{
		{
			name: "running service",
			line: "nginx.service loaded active running A high performance web server",
			want: config.Service{Name: "nginx", State: "running", Enabled: "—", Description: "A high performance web server"},
		},
		{
			name: "failed service",
			line: "postgresql.service loaded failed failed PostgreSQL database server",
			want: config.Service{Name: "postgresql", State: "failed", Enabled: "—", Description: "PostgreSQL database server"},
		},
		{
			name: "inactive service",
			line: "sshd-keygen@.service loaded inactive dead OpenSSH per-connection server daemon",
			want: config.Service{Name: "sshd-keygen@", State: "inactive", Enabled: "—", Description: "OpenSSH per-connection server daemon"},
		},
		{
			name: "too few fields",
			line: "incomplete",
			want: config.Service{},
		},
		{
			name: "no description",
			line: "test.service loaded active running",
			want: config.Service{Name: "test", State: "running", Enabled: "—", Description: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseServiceLine(tt.line)
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.State != tt.want.State {
				t.Errorf("State = %q, want %q", got.State, tt.want.State)
			}
			if got.Enabled != tt.want.Enabled {
				t.Errorf("Enabled = %q, want %q", got.Enabled, tt.want.Enabled)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name    string
		svcName string
		filters []string
		want    bool
	}{
		{"empty filter matches all", "anything", nil, true},
		{"empty filter slice matches all", "anything", []string{}, true},
		{"exact glob match", "nginx", []string{"nginx"}, true},
		{"wildcard match", "automation-controller", []string{"automation-*"}, true},
		{"no match", "postgresql", []string{"automation-*"}, false},
		{"multiple filters, one matches", "nginx", []string{"automation-*", "nginx*"}, true},
		{"multiple filters, none match", "redis", []string{"automation-*", "nginx*"}, false},
		{"star matches all", "anything", []string{"*"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesFilter(tt.svcName, tt.filters)
			if got != tt.want {
				t.Errorf("MatchesFilter(%q, %v) = %v, want %v", tt.svcName, tt.filters, got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"auth error", errString("unable to authenticate"), true},
		{"no methods", errString("no supported methods remain"), true},
		{"handshake", errString("handshake failed"), true},
		{"connection refused", errString("connection refused"), false},
		{"timeout", errString("i/o timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuthError(tt.err)
			if got != tt.want {
				t.Errorf("IsAuthError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"tilde path", "~/.ssh/id_rsa"},
		{"absolute path", "/home/user/.ssh/id_rsa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.path)
			if got == "" {
				t.Error("ExpandPath() returned empty string")
			}
			if tt.path[0] == '~' && got[0] == '~' {
				t.Error("tilde was not expanded")
			}
			if tt.path[0] == '/' && got != tt.path {
				t.Errorf("absolute path changed: %q → %q", tt.path, got)
			}
		})
	}
}

func TestExtractPkgName(t *testing.T) {
	tests := []struct {
		name string
		nvra string
		want string
	}{
		{"simple", "nginx-1.20.1-1.el9.x86_64", "nginx"},
		{"with epoch", "ansible-core-1:2.16.17-1.el9ap.noarch", "ansible-core"},
		{"complex name", "python3-dnf-plugins-core-4.3.0-5.el9.noarch", "python3-dnf-plugins-core"},
		{"no arch suffix", "simple-1.0-1", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPkgName(tt.nvra)
			if got != tt.want {
				t.Errorf("ExtractPkgName(%q) = %q, want %q", tt.nvra, got, tt.want)
			}
		})
	}
}

// errString is a simple error type for testing.
type errString string

func (e errString) Error() string { return string(e) }
