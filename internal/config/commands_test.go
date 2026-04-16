package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeCommands_DefaultsOnly(t *testing.T) {
	defaults := []CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -h"},
		{Name: "uptime", Group: "OS", Run: "uptime"},
	}
	got := MergeCommands(defaults, nil, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(got))
	}
	if got[0].Name != "check-disk" || got[1].Name != "uptime" {
		t.Errorf("unexpected order: %v", got)
	}
}

func TestMergeCommands_GroupAdds(t *testing.T) {
	defaults := []CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -h"},
	}
	group := []CommandEntry{
		{Name: "restart-nginx", Group: "Web", Run: "sudo systemctl restart nginx"},
	}
	got := MergeCommands(defaults, group, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(got))
	}
	if got[0].Name != "check-disk" || got[1].Name != "restart-nginx" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestMergeCommands_HostOverridesSameNameAndGroup(t *testing.T) {
	defaults := []CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -h"},
	}
	host := []CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -hT"},
	}
	got := MergeCommands(defaults, nil, host)
	if len(got) != 1 {
		t.Fatalf("expected 1 command, got %d", len(got))
	}
	if got[0].Run != "df -hT" {
		t.Errorf("expected host override, got %q", got[0].Run)
	}
}

func TestMergeCommands_SameNameDifferentGroup(t *testing.T) {
	defaults := []CommandEntry{
		{Name: "restart", Group: "Web", Run: "sudo systemctl restart nginx"},
	}
	group := []CommandEntry{
		{Name: "restart", Group: "AAP", Run: "sudo supervisorctl restart all"},
	}
	got := MergeCommands(defaults, group, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 commands (different groups), got %d", len(got))
	}
	if got[0].Group != "Web" || got[1].Group != "AAP" {
		t.Errorf("unexpected groups: %v", got)
	}
}

func TestMergeCommands_Empty(t *testing.T) {
	got := MergeCommands(nil, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(got))
	}
}

func TestMergeCommands_ThreeLevelCascade(t *testing.T) {
	defaults := []CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -h"},
		{Name: "check-mem", Group: "OS", Run: "free -m"},
	}
	group := []CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -hT"}, // override
		{Name: "migrate", Group: "AAP", Run: "awx-manage migrate"},
	}
	host := []CommandEntry{
		{Name: "migrate", Group: "AAP", Run: "sudo -u awx awx-manage migrate"}, // override
		{Name: "tail-log", Group: "Debug", Run: "tail -f /var/log/app.log"},
	}
	got := MergeCommands(defaults, group, host)
	if len(got) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(got))
	}
	// check-disk overridden by group
	if got[0].Run != "df -hT" {
		t.Errorf("check-disk should be group override, got %q", got[0].Run)
	}
	// check-mem from defaults
	if got[1].Run != "free -m" {
		t.Errorf("check-mem should be from defaults, got %q", got[1].Run)
	}
	// migrate overridden by host
	if got[2].Run != "sudo -u awx awx-manage migrate" {
		t.Errorf("migrate should be host override, got %q", got[2].Run)
	}
	// tail-log added by host
	if got[3].Name != "tail-log" {
		t.Errorf("tail-log should be added by host, got %q", got[3].Name)
	}
}

func TestValidateCommand_Valid(t *testing.T) {
	err := ValidateCommand(CommandEntry{Name: "check-disk", Group: "OS", Run: "df -h"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateCommand_MissingName(t *testing.T) {
	err := ValidateCommand(CommandEntry{Name: "", Group: "OS", Run: "df -h"})
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateCommand_MissingGroup(t *testing.T) {
	err := ValidateCommand(CommandEntry{Name: "check", Group: "", Run: "df -h"})
	if err == nil {
		t.Error("expected error for missing group")
	}
}

func TestValidateCommand_MissingRun(t *testing.T) {
	err := ValidateCommand(CommandEntry{Name: "check-disk", Group: "OS", Run: ""})
	if err == nil {
		t.Error("expected error for missing run")
	}
}

func TestParseFleetFile_CommandsCascade(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: test
type: vm
defaults:
  user: ansible
  commands:
    - name: check-disk
      group: OS
      run: df -h
    - name: uptime
      group: OS
      run: uptime
groups:
  - name: AAP
    commands:
      - name: migrate
        group: AAP
        run: awx-manage migrate
    hosts:
      - name: aap-01
        hostname: aap-01.example.com
        commands:
          - name: check-disk
            group: OS
            run: df -hT
hosts:
  - name: standalone
    hostname: standalone.example.com
`
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	fleet, err := ParseFleetFile(path)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// grouped host: aap-01 should have defaults (check-disk overridden, uptime) + group (migrate)
	aap := fleet.Groups[0].Hosts[0]
	if len(aap.Commands) != 3 {
		t.Fatalf("aap-01: expected 3 commands, got %d: %v", len(aap.Commands), aap.Commands)
	}
	// check-disk overridden by host
	if aap.Commands[0].Run != "df -hT" {
		t.Errorf("aap-01 check-disk: expected host override 'df -hT', got %q", aap.Commands[0].Run)
	}
	// uptime from defaults
	if aap.Commands[1].Name != "uptime" {
		t.Errorf("aap-01: expected uptime from defaults, got %q", aap.Commands[1].Name)
	}
	// migrate from group
	if aap.Commands[2].Name != "migrate" {
		t.Errorf("aap-01: expected migrate from group, got %q", aap.Commands[2].Name)
	}

	// ungrouped host: standalone should have defaults only
	standalone := fleet.Hosts[0]
	if len(standalone.Commands) != 2 {
		t.Fatalf("standalone: expected 2 commands, got %d", len(standalone.Commands))
	}
	if standalone.Commands[0].Run != "df -h" {
		t.Errorf("standalone check-disk: expected default 'df -h', got %q", standalone.Commands[0].Run)
	}
}
