package ssh

import (
	"testing"
)

func TestFormatDateEU(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"datetime with seconds", "2026-03-31 09:57:00", "31/03/2026"},
		{"datetime without seconds", "2026-03-31 09:57", "31/03/2026"},
		{"date only", "2026-03-31", "31/03/2026"},
		{"empty string", "", ""},
		{"dash", "—", "—"},
		{"unknown", "unknown", "unknown"},
		{"unparseable", "not-a-date", "not-a-date"},
		{"whitespace", "  2026-04-05 15:56  ", "05/04/2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDateEU(tt.input)
			if got != tt.want {
				t.Errorf("FormatDateEU(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseProbeOutput_Full(t *testing.T) {
	// 11 lines: hostname, uptime, OS, cron, errors, disk, disk_high, users, ifaces_up, ifaces_total, ports
	output := `web-01.example.com
2026-03-31 09:57
Red Hat Enterprise Linux 9.5 (Plow)
12
3
6
1
4
3
5
8`

	info, err := ParseProbeOutput(output, "system")
	if err != nil {
		t.Fatalf("ParseProbeOutput() error = %v", err)
	}

	if info.FQDN != "web-01.example.com" {
		t.Errorf("FQDN = %q, want %q", info.FQDN, "web-01.example.com")
	}
	if info.UpSince != "31/03/2026" {
		t.Errorf("UpSince = %q, want %q", info.UpSince, "31/03/2026")
	}
	if info.OS != "Red Hat Enterprise Linux 9.5 (Plow)" {
		t.Errorf("OS = %q", info.OS)
	}
	if info.CronCount != 12 {
		t.Errorf("CronCount = %d, want 12", info.CronCount)
	}
	if info.ErrorCount != 3 {
		t.Errorf("ErrorCount = %d, want 3", info.ErrorCount)
	}
	if info.DiskCount != 6 {
		t.Errorf("DiskCount = %d, want 6", info.DiskCount)
	}
	if info.DiskHighCount != 1 {
		t.Errorf("DiskHighCount = %d, want 1", info.DiskHighCount)
	}
	if info.UserCount != 4 {
		t.Errorf("UserCount = %d, want 4", info.UserCount)
	}
	if info.InterfacesUp != 3 {
		t.Errorf("InterfacesUp = %d, want 3", info.InterfacesUp)
	}
	if info.InterfacesTotal != 5 {
		t.Errorf("InterfacesTotal = %d, want 5", info.InterfacesTotal)
	}
	if info.ListeningPorts != 8 {
		t.Errorf("ListeningPorts = %d, want 8", info.ListeningPorts)
	}
	if info.SystemdMode != "system" {
		t.Errorf("SystemdMode = %q, want %q", info.SystemdMode, "system")
	}
}

func TestParseProbeOutput_MinimalLines(t *testing.T) {
	// 3 lines is the minimum required
	output := `host1
2026-01-01 00:00
RHEL 9`

	info, err := ParseProbeOutput(output, "user")
	if err != nil {
		t.Fatalf("ParseProbeOutput() error = %v", err)
	}
	if info.FQDN != "host1" {
		t.Errorf("FQDN = %q, want %q", info.FQDN, "host1")
	}
	if info.SystemdMode != "user" {
		t.Errorf("SystemdMode = %q, want %q", info.SystemdMode, "user")
	}
}

func TestParseProbeOutput_TooFewLines(t *testing.T) {
	_, err := ParseProbeOutput("only\ntwo", "system")
	if err == nil {
		t.Fatal("expected error for too few lines, got nil")
	}
}

func TestParseProbeOutput_EmptyOutput(t *testing.T) {
	_, err := ParseProbeOutput("", "system")
	if err == nil {
		t.Fatal("expected error for empty output, got nil")
	}
}
