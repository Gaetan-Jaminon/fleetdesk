package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppConfig_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadAppConfig(dir)
	if err != ErrNoConfig {
		t.Errorf("got %v, want ErrNoConfig", err)
	}
}

func TestLoadAppConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAppConfig(dir)
	if err == nil || err == ErrNoConfig {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestLoadAppConfig_MissingFleetDir(t *testing.T) {
	dir := t.TempDir()
	content := "editor: vim\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAppConfig(dir)
	if err == nil {
		t.Error("expected validation error for missing fleet_dir")
	}
}

func TestLoadAppConfig_DirNotExist(t *testing.T) {
	dir := t.TempDir()
	content := "fleet_dir: /nonexistent/path/that/does/not/exist\neditor: vim\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAppConfig(dir)
	if err == nil {
		t.Error("expected error for non-existent fleet_dir")
	}
}

func TestLoadAppConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	fleetDir := t.TempDir()
	content := "fleet_dir: " + fleetDir + "\neditor: nvim\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FleetDir != fleetDir {
		t.Errorf("FleetDir = %q, want %q", cfg.FleetDir, fleetDir)
	}
	if cfg.Editor() != "nvim" {
		t.Errorf("Editor() = %q, want %q", cfg.Editor(), "nvim")
	}
}

func TestAppConfig_Editor_ConfigWins(t *testing.T) {
	t.Setenv("EDITOR", "emacs")
	t.Setenv("VISUAL", "code")
	cfg := AppConfig{FleetDir: "/tmp", editor: "nvim"}
	if got := cfg.Editor(); got != "nvim" {
		t.Errorf("Editor() = %q, want %q", got, "nvim")
	}
}

func TestAppConfig_Editor_EnvFallback(t *testing.T) {
	t.Setenv("EDITOR", "emacs")
	t.Setenv("VISUAL", "code")
	cfg := AppConfig{FleetDir: "/tmp"}
	if got := cfg.Editor(); got != "emacs" {
		t.Errorf("Editor() = %q, want %q", got, "emacs")
	}
}

func TestAppConfig_Editor_VisualFallback(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "code")
	cfg := AppConfig{FleetDir: "/tmp"}
	if got := cfg.Editor(); got != "code" {
		t.Errorf("Editor() = %q, want %q", got, "code")
	}
}

func TestAppConfig_Editor_Default(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	cfg := AppConfig{FleetDir: "/tmp"}
	if got := cfg.Editor(); got != "vi" {
		t.Errorf("Editor() = %q, want %q", got, "vi")
	}
}

func TestWriteDefaultAppConfig(t *testing.T) {
	dir := t.TempDir()
	fleetDir := t.TempDir()
	err := WriteDefaultAppConfig(dir, fleetDir, "nvim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify file was created and is readable
	cfg, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatalf("LoadAppConfig after write: %v", err)
	}
	if cfg.FleetDir != fleetDir {
		t.Errorf("FleetDir = %q, want %q", cfg.FleetDir, fleetDir)
	}
	if cfg.Editor() != "nvim" {
		t.Errorf("Editor() = %q, want %q", cfg.Editor(), "nvim")
	}
}

func TestValidateAppConfig_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	// Create a temp dir inside home to test tilde expansion
	dir := t.TempDir()
	fleetDir := t.TempDir()

	// Write config with ~/... path
	rel, err := filepath.Rel(home, fleetDir)
	if err != nil {
		t.Skip("fleet dir not under home")
	}
	content := "fleet_dir: ~/" + rel + "\neditor: vim\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FleetDir != fleetDir {
		t.Errorf("FleetDir = %q, want %q (tilde not expanded)", cfg.FleetDir, fleetDir)
	}
}
