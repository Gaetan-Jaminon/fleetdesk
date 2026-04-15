package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFleetTypeOrder(t *testing.T) {
	tests := []struct {
		ftype string
		want  int
	}{
		{"vm", 0},
		{"azure", 1},
		{"kubernetes", 2},
		{"unknown", 3},
		{"", 3},
	}

	for _, tt := range tests {
		t.Run(tt.ftype, func(t *testing.T) {
			got := fleetTypeOrder(tt.ftype)
			if got != tt.want {
				t.Errorf("fleetTypeOrder(%q) = %d, want %d", tt.ftype, got, tt.want)
			}
		})
	}

	// verify ordering: vm < azure < kubernetes
	if fleetTypeOrder("vm") >= fleetTypeOrder("azure") {
		t.Error("vm should sort before azure")
	}
	if fleetTypeOrder("azure") >= fleetTypeOrder("kubernetes") {
		t.Error("azure should sort before kubernetes")
	}
}

func TestScanFleets_AcceptsDir(t *testing.T) {
	dir := t.TempDir()
	// write a minimal fleet file
	content := "name: test\ntype: vm\nhosts:\n  - name: h1\n    hostname: 10.0.0.1\n"
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	fleets, err := ScanFleets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fleets) != 1 {
		t.Fatalf("got %d fleets, want 1", len(fleets))
	}
	if fleets[0].Name != "test" {
		t.Errorf("fleet name = %q, want %q", fleets[0].Name, "test")
	}
}

func TestScanFleets_SkipsConfigYaml(t *testing.T) {
	dir := t.TempDir()
	// write config.yaml — should be skipped
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("fleet_dir: /tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// write a fleet file
	content := "name: real\ntype: vm\nhosts:\n  - name: h1\n    hostname: 10.0.0.1\n"
	if err := os.WriteFile(filepath.Join(dir, "fleet.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	fleets, err := ScanFleets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fleets) != 1 {
		t.Fatalf("got %d fleets, want 1 (config.yaml should be skipped)", len(fleets))
	}
	if fleets[0].Name != "real" {
		t.Errorf("fleet name = %q, want %q", fleets[0].Name, "real")
	}
}
