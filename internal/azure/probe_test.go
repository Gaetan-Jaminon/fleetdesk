package azure

import "testing"

func TestCountVMStates(t *testing.T) {
	tests := []struct {
		name            string
		vms             []VM
		wantTotal       int
		wantRunning     int
		wantStopped     int
		wantDeallocated int
	}{
		{
			name:      "nil",
			vms:       nil,
			wantTotal: 0,
		},
		{
			name: "mixed states",
			vms: []VM{
				{PowerState: "running"},
				{PowerState: "running"},
				{PowerState: "stopped"},
				{PowerState: "deallocated"},
				{PowerState: "running"},
			},
			wantTotal:       5,
			wantRunning:     3,
			wantStopped:     1,
			wantDeallocated: 1,
		},
		{
			name: "all running",
			vms: []VM{
				{PowerState: "running"},
				{PowerState: "running"},
			},
			wantTotal:   2,
			wantRunning: 2,
		},
		{
			name: "unknown state ignored",
			vms: []VM{
				{PowerState: "running"},
				{PowerState: "starting"},
			},
			wantTotal:   2,
			wantRunning: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, running, stopped, deallocated := CountVMStates(tt.vms)
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if running != tt.wantRunning {
				t.Errorf("running = %d, want %d", running, tt.wantRunning)
			}
			if stopped != tt.wantStopped {
				t.Errorf("stopped = %d, want %d", stopped, tt.wantStopped)
			}
			if deallocated != tt.wantDeallocated {
				t.Errorf("deallocated = %d, want %d", deallocated, tt.wantDeallocated)
			}
		})
	}
}

func TestSumAKSNodes(t *testing.T) {
	tests := []struct {
		name      string
		clusters  []AKSCluster
		wantCount int
		wantNodes int
	}{
		{
			name:      "nil",
			clusters:  nil,
			wantCount: 0,
			wantNodes: 0,
		},
		{
			name: "multiple clusters",
			clusters: []AKSCluster{
				{NodeCount: 3},
				{NodeCount: 5},
				{NodeCount: 2},
			},
			wantCount: 3,
			wantNodes: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, nodes := SumAKSNodes(tt.clusters)
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
			if nodes != tt.wantNodes {
				t.Errorf("nodes = %d, want %d", nodes, tt.wantNodes)
			}
		})
	}
}
