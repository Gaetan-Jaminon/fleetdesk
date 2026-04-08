package config

import "testing"

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
