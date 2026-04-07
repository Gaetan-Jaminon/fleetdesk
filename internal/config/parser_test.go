package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFleetFile_Valid(t *testing.T) {
	yaml := `
name: Test Fleet
type: vm
defaults:
  user: admin
  port: 2222
  timeout: 5s
  systemd_mode: user
  service_filter:
    - "nginx*"
  error_log_since: "30 min ago"
  refresh_interval: "30s"
groups:
  - name: Web Servers
    service_filter:
      - "httpd*"
    hosts:
      - name: web-01
        hostname: web-01.example.com
      - name: web-02
        hostname: web-02.example.com
hosts:
  - name: standalone-01
    hostname: standalone-01.example.com
    user: root
    port: 22
    timeout: 3s
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}

	if f.Name != "Test Fleet" {
		t.Errorf("Name = %q, want %q", f.Name, "Test Fleet")
	}
	if f.Type != "vm" {
		t.Errorf("Type = %q, want %q", f.Type, "vm")
	}
	if f.Defaults.User != "admin" {
		t.Errorf("Defaults.User = %q, want %q", f.Defaults.User, "admin")
	}
	if f.Defaults.Port != 2222 {
		t.Errorf("Defaults.Port = %d, want %d", f.Defaults.Port, 2222)
	}
	if f.Defaults.Timeout != 5*time.Second {
		t.Errorf("Defaults.Timeout = %v, want %v", f.Defaults.Timeout, 5*time.Second)
	}
	if f.Defaults.SystemdMode != "user" {
		t.Errorf("Defaults.SystemdMode = %q, want %q", f.Defaults.SystemdMode, "user")
	}
	if f.Defaults.ErrorLogSince != "30 min ago" {
		t.Errorf("Defaults.ErrorLogSince = %q, want %q", f.Defaults.ErrorLogSince, "30 min ago")
	}
	if f.Defaults.RefreshInterval != "30s" {
		t.Errorf("Defaults.RefreshInterval = %q, want %q", f.Defaults.RefreshInterval, "30s")
	}
	if len(f.Groups) != 1 {
		t.Fatalf("len(Groups) = %d, want 1", len(f.Groups))
	}
	if f.Groups[0].Name != "Web Servers" {
		t.Errorf("Groups[0].Name = %q, want %q", f.Groups[0].Name, "Web Servers")
	}
	if len(f.Groups[0].Hosts) != 2 {
		t.Errorf("len(Groups[0].Hosts) = %d, want 2", len(f.Groups[0].Hosts))
	}
	if len(f.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(f.Hosts))
	}
	if f.Hosts[0].User != "root" {
		t.Errorf("Hosts[0].User = %q, want %q", f.Hosts[0].User, "root")
	}
	if f.Hosts[0].Port != 22 {
		t.Errorf("Hosts[0].Port = %d, want %d", f.Hosts[0].Port, 22)
	}
	if f.Hosts[0].Timeout != 3*time.Second {
		t.Errorf("Hosts[0].Timeout = %v, want %v", f.Hosts[0].Timeout, 3*time.Second)
	}
}

func TestParseFleetFile_Defaults(t *testing.T) {
	yaml := `
name: Minimal
hosts:
  - name: h1
    hostname: h1.example.com
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}

	if f.Type != "vm" {
		t.Errorf("Type = %q, want default %q", f.Type, "vm")
	}
	if f.Defaults.Port != 22 {
		t.Errorf("Defaults.Port = %d, want default 22", f.Defaults.Port)
	}
	if f.Defaults.SystemdMode != "system" {
		t.Errorf("Defaults.SystemdMode = %q, want default %q", f.Defaults.SystemdMode, "system")
	}
	if f.Defaults.Timeout != 10*time.Second {
		t.Errorf("Defaults.Timeout = %v, want default %v", f.Defaults.Timeout, 10*time.Second)
	}
	if f.Defaults.ErrorLogSince != "1 hour ago" {
		t.Errorf("Defaults.ErrorLogSince = %q, want default %q", f.Defaults.ErrorLogSince, "1 hour ago")
	}
	if f.Defaults.RefreshInterval != "15s" {
		t.Errorf("Defaults.RefreshInterval = %q, want default %q", f.Defaults.RefreshInterval, "15s")
	}
}

func TestParseFleetFile_NameFromFilename(t *testing.T) {
	yaml := `
hosts:
  - name: h1
    hostname: h1.example.com
`
	dir := t.TempDir()
	path := filepath.Join(dir, "my-fleet.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}
	if f.Name != "my-fleet" {
		t.Errorf("Name = %q, want %q (derived from filename)", f.Name, "my-fleet")
	}
}

func TestParseFleetFile_InvalidTimeout(t *testing.T) {
	yaml := `
name: Bad
defaults:
  timeout: not-a-duration
hosts:
  - name: h1
    hostname: h1.example.com
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for invalid timeout, got nil")
	}
}

func TestParseFleetFile_MalformedYAML(t *testing.T) {
	path := writeTempYAML(t, `{{{not yaml`)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestParseFleetFile_MissingFile(t *testing.T) {
	_, err := ParseFleetFile("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseHosts_MissingName(t *testing.T) {
	yaml := `
name: Bad
hosts:
  - hostname: h1.example.com
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for missing host name, got nil")
	}
}

func TestParseHosts_MissingHostname(t *testing.T) {
	yaml := `
name: Bad
hosts:
  - name: h1
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for missing hostname, got nil")
	}
}

func TestParseHosts_ServiceFilterInheritance(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantHost []string
		wantGrp  []string
	}{
		{
			name: "host filter wins",
			yaml: `
name: Test
defaults:
  service_filter: ["default-*"]
groups:
  - name: grp
    service_filter: ["group-*"]
    hosts:
      - name: h1
        hostname: h1.example.com
        service_filter: ["host-*"]
hosts:
  - name: h2
    hostname: h2.example.com
`,
			wantGrp:  []string{"host-*"},
			wantHost: []string{"default-*"},
		},
		{
			name: "group filter used when host has none",
			yaml: `
name: Test
defaults:
  service_filter: ["default-*"]
groups:
  - name: grp
    service_filter: ["group-*"]
    hosts:
      - name: h1
        hostname: h1.example.com
hosts:
  - name: h2
    hostname: h2.example.com
`,
			wantGrp:  []string{"group-*"},
			wantHost: []string{"default-*"},
		},
		{
			name: "defaults used when group and host have none",
			yaml: `
name: Test
defaults:
  service_filter: ["default-*"]
groups:
  - name: grp
    hosts:
      - name: h1
        hostname: h1.example.com
hosts:
  - name: h2
    hostname: h2.example.com
`,
			wantGrp:  []string{"default-*"},
			wantHost: []string{"default-*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempYAML(t, tt.yaml)
			f, err := ParseFleetFile(path)
			if err != nil {
				t.Fatalf("ParseFleetFile() error = %v", err)
			}

			grpHost := f.Groups[0].Hosts[0]
			if !sliceEqual(grpHost.ServiceFilter, tt.wantGrp) {
				t.Errorf("group host filter = %v, want %v", grpHost.ServiceFilter, tt.wantGrp)
			}

			ungrouped := f.Hosts[0]
			if !sliceEqual(ungrouped.ServiceFilter, tt.wantHost) {
				t.Errorf("ungrouped host filter = %v, want %v", ungrouped.ServiceFilter, tt.wantHost)
			}
		})
	}
}

func TestParseHosts_DefaultsApplied(t *testing.T) {
	yaml := `
name: Test
defaults:
  user: admin
  port: 2222
  systemd_mode: user
  timeout: 5s
hosts:
  - name: h1
    hostname: h1.example.com
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}

	h := f.Hosts[0]
	if h.User != "admin" {
		t.Errorf("User = %q, want %q", h.User, "admin")
	}
	if h.Port != 2222 {
		t.Errorf("Port = %d, want %d", h.Port, 2222)
	}
	if h.SystemdMode != "user" {
		t.Errorf("SystemdMode = %q, want %q", h.SystemdMode, "user")
	}
	if h.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want %v", h.Timeout, 5*time.Second)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Fatal("ConfigPath() returned empty string")
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "fleetdesk")
	if path != want {
		t.Errorf("ConfigPath() = %q, want %q", path, want)
	}
}

// helpers

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
