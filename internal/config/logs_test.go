package config

import (
	"strings"
	"testing"
)

func TestMergeLogEntries_DefaultsOnly(t *testing.T) {
	defaults := []LogEntry{{Name: "syslog", Path: "/var/log/messages", Sudo: true}}
	result := MergeLogEntries(defaults, nil, nil)
	if len(result) != 1 || result[0].Name != "syslog" {
		t.Errorf("got %v, want 1 entry named syslog", result)
	}
}

func TestMergeLogEntries_GroupAdds(t *testing.T) {
	defaults := []LogEntry{{Name: "syslog", Path: "/var/log/messages"}}
	group := []LogEntry{{Name: "app", Path: "/var/log/app.log"}}
	result := MergeLogEntries(defaults, group, nil)
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[0].Name != "syslog" || result[1].Name != "app" {
		t.Errorf("got names %q %q, want syslog app", result[0].Name, result[1].Name)
	}
}

func TestMergeLogEntries_HostAdds(t *testing.T) {
	defaults := []LogEntry{{Name: "syslog", Path: "/var/log/messages"}}
	host := []LogEntry{{Name: "tower", Path: "/var/log/tower/tower.log", Sudo: true}}
	result := MergeLogEntries(defaults, nil, host)
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[1].Name != "tower" {
		t.Errorf("got %q, want tower", result[1].Name)
	}
}

func TestMergeLogEntries_HostOverridesSameName(t *testing.T) {
	defaults := []LogEntry{{Name: "syslog", Path: "/var/log/messages", Sudo: false}}
	host := []LogEntry{{Name: "syslog", Path: "/var/log/syslog", Sudo: true}}
	result := MergeLogEntries(defaults, nil, host)
	if len(result) != 1 {
		t.Fatalf("got %d entries, want 1", len(result))
	}
	if result[0].Path != "/var/log/syslog" || !result[0].Sudo {
		t.Errorf("host should override: got path=%q sudo=%v", result[0].Path, result[0].Sudo)
	}
}

func TestMergeLogEntries_GroupOverridesDefaults(t *testing.T) {
	defaults := []LogEntry{{Name: "syslog", Path: "/var/log/messages"}}
	group := []LogEntry{{Name: "syslog", Path: "/var/log/syslog"}}
	result := MergeLogEntries(defaults, group, nil)
	if len(result) != 1 || result[0].Path != "/var/log/syslog" {
		t.Errorf("group should override defaults: got %v", result)
	}
}

func TestMergeLogEntries_AllThreeLevels(t *testing.T) {
	defaults := []LogEntry{
		{Name: "syslog", Path: "/var/log/messages"},
		{Name: "secure", Path: "/var/log/secure"},
	}
	group := []LogEntry{
		{Name: "app", Path: "/var/log/app.log"},
	}
	host := []LogEntry{
		{Name: "tower", Path: "/var/log/tower/tower.log", Sudo: true},
		{Name: "syslog", Path: "/var/log/syslog"}, // override defaults
	}
	result := MergeLogEntries(defaults, group, host)
	if len(result) != 4 {
		t.Fatalf("got %d entries, want 4", len(result))
	}
	// syslog should be overridden by host
	if result[0].Path != "/var/log/syslog" {
		t.Errorf("syslog should be overridden: got %q", result[0].Path)
	}
}

func TestMergeLogEntries_Empty(t *testing.T) {
	result := MergeLogEntries(nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("got %d entries, want 0", len(result))
	}
}

func TestValidateLogPath_Valid(t *testing.T) {
	if err := ValidateLogPath("test", "/var/log/test.log"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogPath_ValidWithSpaces(t *testing.T) {
	if err := ValidateLogPath("test", "/var/log/my app/test.log"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogPath_RelativePath(t *testing.T) {
	err := ValidateLogPath("test", "var/log/test.log")
	if err == nil {
		t.Fatal("expected error for relative path")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("error = %q, want to mention absolute", err.Error())
	}
}

func TestValidateLogPath_Empty(t *testing.T) {
	err := ValidateLogPath("test", "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestValidateLogPath_ShellMetacharacters(t *testing.T) {
	badPaths := []string{
		"/var/log/$HOME/test.log",
		"/var/log/test;rm -rf /",
		"/var/log/test&bg",
		"/var/log/test|pipe",
		"/var/log/test>redirect",
		"/var/log/test<input",
		"/var/log/test`cmd`",
		"/var/log/test\ninjection",
		"/var/log/test\\escaped",
		"/var/log/test'quote",
	}
	for _, p := range badPaths {
		if err := ValidateLogPath("test", p); err == nil {
			t.Errorf("expected error for path %q", p)
		}
	}
}

func TestValidateLogPath_TooLong(t *testing.T) {
	long := "/" + strings.Repeat("a", 512)
	err := ValidateLogPath("test", long)
	if err == nil {
		t.Fatal("expected error for long path")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("error = %q, want to mention too long", err.Error())
	}
}
