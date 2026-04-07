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

func TestParseAccountLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want config.Account
	}{
		{
			name: "normal user",
			line: "jaminong|1000|jaminong wheel docker|/bin/bash|Apr 06 2026|PS|never",
			want: config.Account{
				User: "jaminong", UID: 1000, Groups: "jaminong wheel docker",
				Shell: "/bin/bash", LastLogin: "Apr 06 2026", PasswordStatus: "PS",
				Expiry: "never", IsSudo: true, IsLocked: false,
			},
		},
		{
			name: "locked user",
			line: "svcaccount|1001|svcaccount|/sbin/nologin|Never|LK|never",
			want: config.Account{
				User: "svcaccount", UID: 1001, Groups: "svcaccount",
				Shell: "/sbin/nologin", LastLogin: "Never", PasswordStatus: "LK",
				Expiry: "never", IsSudo: false, IsLocked: true,
			},
		},
		{
			name: "sudo group member",
			line: "admin|1002|admin sudo|/bin/zsh|Mar 15 2026|PS|never",
			want: config.Account{
				User: "admin", UID: 1002, Groups: "admin sudo",
				Shell: "/bin/zsh", LastLogin: "Mar 15 2026", PasswordStatus: "PS",
				Expiry: "never", IsSudo: true, IsLocked: false,
			},
		},
		{
			name: "L status (locked alternate)",
			line: "olduser|1003|olduser|/bin/bash|Never|L|2025-12-31",
			want: config.Account{
				User: "olduser", UID: 1003, Groups: "olduser",
				Shell: "/bin/bash", LastLogin: "Never", PasswordStatus: "L",
				Expiry: "2025-12-31", IsSudo: false, IsLocked: true,
			},
		},
		{
			name: "missing fields",
			line: "partial|1004",
			want: config.Account{User: "partial", UID: 1004},
		},
		{
			name: "empty line",
			line: "",
			want: config.Account{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAccountLine(tt.line)
			if got.User != tt.want.User {
				t.Errorf("User = %q, want %q", got.User, tt.want.User)
			}
			if got.UID != tt.want.UID {
				t.Errorf("UID = %d, want %d", got.UID, tt.want.UID)
			}
			if got.Groups != tt.want.Groups {
				t.Errorf("Groups = %q, want %q", got.Groups, tt.want.Groups)
			}
			if got.Shell != tt.want.Shell {
				t.Errorf("Shell = %q, want %q", got.Shell, tt.want.Shell)
			}
			if got.LastLogin != tt.want.LastLogin {
				t.Errorf("LastLogin = %q, want %q", got.LastLogin, tt.want.LastLogin)
			}
			if got.PasswordStatus != tt.want.PasswordStatus {
				t.Errorf("PasswordStatus = %q, want %q", got.PasswordStatus, tt.want.PasswordStatus)
			}
			if got.Expiry != tt.want.Expiry {
				t.Errorf("Expiry = %q, want %q", got.Expiry, tt.want.Expiry)
			}
			if got.IsSudo != tt.want.IsSudo {
				t.Errorf("IsSudo = %v, want %v", got.IsSudo, tt.want.IsSudo)
			}
			if got.IsLocked != tt.want.IsLocked {
				t.Errorf("IsLocked = %v, want %v", got.IsLocked, tt.want.IsLocked)
			}
		})
	}
}

func TestAccountStateOrder(t *testing.T) {
	locked := config.Account{IsLocked: true}
	normal := config.Account{}

	if AccountStateOrder(locked) >= AccountStateOrder(normal) {
		t.Error("locked should sort before normal")
	}
}

// errString is a simple error type for testing.
type errString string

func (e errString) Error() string { return string(e) }
