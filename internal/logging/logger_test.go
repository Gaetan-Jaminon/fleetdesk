package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLogger_Disabled(t *testing.T) {
	dir := t.TempDir()
	logger := InitLogger(false, dir)
	if logger == nil {
		t.Fatal("InitLogger(false) returned nil")
	}
	// should not create any files
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files when disabled, got %d", len(entries))
	}
}

func TestInitLogger_Enabled(t *testing.T) {
	dir := t.TempDir()
	logger := InitLogger(true, dir)
	if logger == nil {
		t.Fatal("InitLogger(true) returned nil")
	}
	// should create debug.log
	path := filepath.Join(dir, "debug.log")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("debug.log not created")
	}
	// write a test log
	logger.Info("test message", "key", "value")
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "test message") {
		t.Errorf("debug.log does not contain test message: %s", content)
	}
}

func TestNewTargetLogger_Disabled(t *testing.T) {
	dir := t.TempDir()
	global := InitLogger(false, dir)
	target := NewTargetLogger(global, false, dir, "host", "test-01")
	if target == nil {
		t.Fatal("NewTargetLogger returned nil")
	}
	// should not create target file
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files when disabled, got %d", len(entries))
	}
}

func TestNewTargetLogger_Enabled(t *testing.T) {
	dir := t.TempDir()
	global := InitLogger(true, dir)
	target := NewTargetLogger(global, true, dir, "host", "aap-ctrl-01")
	if target == nil {
		t.Fatal("NewTargetLogger returned nil")
	}
	// should create per-target file
	path := filepath.Join(dir, "host-aap-ctrl-01.log")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("host-aap-ctrl-01.log not created")
	}
	// write a test log — should appear in both files
	target.Info("target message", "host", "aap-ctrl-01")

	targetContent, _ := os.ReadFile(path)
	if !strings.Contains(string(targetContent), "target message") {
		t.Errorf("target log missing message: %s", targetContent)
	}

	globalContent, _ := os.ReadFile(filepath.Join(dir, "debug.log"))
	if !strings.Contains(string(globalContent), "target message") {
		t.Errorf("global log missing target message: %s", globalContent)
	}
}

func TestLogDir(t *testing.T) {
	dir := LogDir()
	if dir == "" {
		t.Fatal("LogDir() returned empty")
	}
	if !strings.Contains(dir, "fleetdesk") {
		t.Errorf("LogDir() = %q, expected to contain 'fleetdesk'", dir)
	}
}
