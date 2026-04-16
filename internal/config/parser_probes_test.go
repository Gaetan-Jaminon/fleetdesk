package config

import (
	"strings"
	"testing"
	"time"
)

func TestParseProbeFleet_Valid(t *testing.T) {
	yaml := `
name: Platform Endpoints
type: probes
defaults:
  interval: 30s
  timeout: 15s
  proxy: http://proxy.corp:8080
groups:
  - name: API Gateway
    probes:
      - name: health
        url: https://api.example.com/health
        expected_code: 200
      - name: metrics
        url: https://api.example.com/metrics
        expected_code: 200
        interval: 60s
  - name: Auth
    probes:
      - name: oidc
        url: https://auth.example.com/.well-known/openid-configuration
probes:
  - name: dashboard
    url: http://dashboard.internal/
    interval: 10s
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}

	if f.Name != "Platform Endpoints" {
		t.Errorf("Name = %q, want %q", f.Name, "Platform Endpoints")
	}
	if f.Type != "probes" {
		t.Errorf("Type = %q, want %q", f.Type, "probes")
	}
	if f.ProbeFleet == nil {
		t.Fatal("ProbeFleet is nil")
	}

	pf := f.ProbeFleet

	// defaults
	if pf.Defaults.Interval != 30*time.Second {
		t.Errorf("Defaults.Interval = %v, want 30s", pf.Defaults.Interval)
	}
	if pf.Defaults.Timeout != 15*time.Second {
		t.Errorf("Defaults.Timeout = %v, want 15s", pf.Defaults.Timeout)
	}
	if pf.Defaults.ProxyURL != "http://proxy.corp:8080" {
		t.Errorf("Defaults.ProxyURL = %q, want %q", pf.Defaults.ProxyURL, "http://proxy.corp:8080")
	}

	// groups
	if len(pf.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(pf.Groups))
	}
	if pf.Groups[0].Name != "API Gateway" {
		t.Errorf("Groups[0].Name = %q, want %q", pf.Groups[0].Name, "API Gateway")
	}
	if len(pf.Groups[0].Probes) != 2 {
		t.Fatalf("len(Groups[0].Probes) = %d, want 2", len(pf.Groups[0].Probes))
	}

	// probe fields
	health := pf.Groups[0].Probes[0]
	if health.Name != "health" {
		t.Errorf("health.Name = %q", health.Name)
	}
	if health.URL != "https://api.example.com/health" {
		t.Errorf("health.URL = %q", health.URL)
	}
	if health.ExpectedCode != 200 {
		t.Errorf("health.ExpectedCode = %d, want 200", health.ExpectedCode)
	}
	if health.Protocol != "http" {
		t.Errorf("health.Protocol = %q, want %q", health.Protocol, "http")
	}
	if health.Interval != 0 {
		t.Errorf("health.Interval = %v, want 0 (use fleet default)", health.Interval)
	}

	// per-probe interval override
	metrics := pf.Groups[0].Probes[1]
	if metrics.Interval != 60*time.Second {
		t.Errorf("metrics.Interval = %v, want 60s", metrics.Interval)
	}

	// ungrouped probes
	if len(pf.Probes) != 1 {
		t.Fatalf("len(Probes) = %d, want 1", len(pf.Probes))
	}
	if pf.Probes[0].Name != "dashboard" {
		t.Errorf("Probes[0].Name = %q, want %q", pf.Probes[0].Name, "dashboard")
	}
	if pf.Probes[0].Interval != 10*time.Second {
		t.Errorf("Probes[0].Interval = %v, want 10s", pf.Probes[0].Interval)
	}
}

func TestParseProbeFleet_DefaultExpectedCode(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - name: p1
    url: https://example.com/health
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}
	if f.ProbeFleet.Probes[0].ExpectedCode != 200 {
		t.Errorf("ExpectedCode = %d, want 200", f.ProbeFleet.Probes[0].ExpectedCode)
	}
}

func TestParseProbeFleet_DefaultInterval(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - name: p1
    url: https://example.com/health
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}
	if f.ProbeFleet.Defaults.Interval != 30*time.Second {
		t.Errorf("Defaults.Interval = %v, want 30s", f.ProbeFleet.Defaults.Interval)
	}
	if f.ProbeFleet.Defaults.Timeout != 10*time.Second {
		t.Errorf("Defaults.Timeout = %v, want 10s", f.ProbeFleet.Defaults.Timeout)
	}
}

func TestParseProbeFleet_MissingURL(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - name: p1
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
	if !strings.Contains(err.Error(), "missing required field 'url'") {
		t.Errorf("error = %q, want to contain 'missing required field url'", err.Error())
	}
}

func TestParseProbeFleet_MissingName(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - url: https://example.com/health
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "missing required field 'name'") {
		t.Errorf("error = %q, want to contain 'missing required field name'", err.Error())
	}
}

func TestParseProbeFleet_InvalidURLScheme(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - name: p1
    url: ftp://example.com/file
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for invalid URL scheme")
	}
	if !strings.Contains(err.Error(), "scheme must be http or https") {
		t.Errorf("error = %q, want to contain 'scheme must be http or https'", err.Error())
	}
}

func TestParseProbeFleet_IntervalTooShort(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - name: p1
    url: https://example.com/health
    interval: 2s
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for short interval")
	}
	if !strings.Contains(err.Error(), ">= 5s") {
		t.Errorf("error = %q, want to contain '>= 5s'", err.Error())
	}
}

func TestParseProbeFleet_DefaultIntervalTooShort(t *testing.T) {
	yaml := `
name: test
type: probes
defaults:
  interval: 3s
probes:
  - name: p1
    url: https://example.com/health
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for short default interval")
	}
	if !strings.Contains(err.Error(), ">= 5s") {
		t.Errorf("error = %q, want to contain '>= 5s'", err.Error())
	}
}

func TestParseProbeFleet_UnsupportedProtocol(t *testing.T) {
	yaml := `
name: test
type: probes
probes:
  - name: p1
    url: https://example.com/health
    protocol: tcp
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if !strings.Contains(err.Error(), "unsupported protocol") {
		t.Errorf("error = %q, want to contain 'unsupported protocol'", err.Error())
	}
}

func TestParseProbeFleet_NameFromFilename(t *testing.T) {
	yaml := `
type: probes
probes:
  - name: p1
    url: https://example.com/health
`
	path := writeTempYAML(t, yaml)
	f, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("ParseFleetFile() error = %v", err)
	}
	if f.Name != "test" {
		t.Errorf("Name = %q, want %q (from filename)", f.Name, "test")
	}
}

func TestParseProbeFleet_GroupMissingName(t *testing.T) {
	yaml := `
name: test
type: probes
groups:
  - probes:
      - name: p1
        url: https://example.com/health
`
	path := writeTempYAML(t, yaml)
	_, err := ParseFleetFile(path)
	if err == nil {
		t.Fatal("expected error for group missing name")
	}
	if !strings.Contains(err.Error(), "group missing required field 'name'") {
		t.Errorf("error = %q, want to contain 'group missing required field name'", err.Error())
	}
}
