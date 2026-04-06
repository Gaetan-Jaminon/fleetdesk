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
	output := `web-01.example.com
2026-03-31 09:57
Red Hat Enterprise Linux 9.5 (Plow)
45
38
2
5
8
2026-03-30 14:22
2026-03-28 10:15
12
3
7
6
1`

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
	if info.ServiceCount != 45 {
		t.Errorf("ServiceCount = %d, want 45", info.ServiceCount)
	}
	if info.ServiceRunning != 38 {
		t.Errorf("ServiceRunning = %d, want 38", info.ServiceRunning)
	}
	if info.ServiceFailed != 2 {
		t.Errorf("ServiceFailed = %d, want 2", info.ServiceFailed)
	}
	if info.ContainerRunning != 5 {
		t.Errorf("ContainerRunning = %d, want 5", info.ContainerRunning)
	}
	if info.ContainerCount != 8 {
		t.Errorf("ContainerCount = %d, want 8", info.ContainerCount)
	}
	if info.LastUpdate != "30/03/2026" {
		t.Errorf("LastUpdate = %q, want %q", info.LastUpdate, "30/03/2026")
	}
	if info.LastSecurity != "28/03/2026" {
		t.Errorf("LastSecurity = %q, want %q", info.LastSecurity, "28/03/2026")
	}
	if info.CronCount != 12 {
		t.Errorf("CronCount = %d, want 12", info.CronCount)
	}
	if info.ErrorCount != 3 {
		t.Errorf("ErrorCount = %d, want 3", info.ErrorCount)
	}
	if info.UpdateCount != 7 {
		t.Errorf("UpdateCount = %d, want 7", info.UpdateCount)
	}
	if info.DiskCount != 6 {
		t.Errorf("DiskCount = %d, want 6", info.DiskCount)
	}
	if info.DiskHighCount != 1 {
		t.Errorf("DiskHighCount = %d, want 1", info.DiskHighCount)
	}
	if info.SystemdMode != "system" {
		t.Errorf("SystemdMode = %q, want %q", info.SystemdMode, "system")
	}
}

func TestParseProbeOutput_MinimalLines(t *testing.T) {
	// 7 lines is the minimum required
	output := `host1
2026-01-01 00:00
RHEL 9
10
8
0
2`

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
	_, err := ParseProbeOutput("only\nthree\nlines", "system")
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
