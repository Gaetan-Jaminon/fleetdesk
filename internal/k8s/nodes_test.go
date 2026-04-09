package k8s

import (
	"testing"
	"time"
)

func TestParseNodes(t *testing.T) {
	input := []byte(`{"items":[
		{
			"metadata":{"name":"aks-system-vmss000001","labels":{"agentpool":"system","kubernetes.io/os":"linux","kubernetes.io/arch":"amd64","node.kubernetes.io/instance-type":"Standard_D4s_v5"}},
			"status":{
				"conditions":[{"type":"Ready","status":"True"}],
				"nodeInfo":{"kubeletVersion":"v1.34.2"},
				"capacity":{"cpu":"4","memory":"32874620Ki","pods":"250"}
			}
		},
		{
			"metadata":{"name":"aks-default-vmss000001","labels":{"agentpool":"default","kubernetes.io/os":"linux","kubernetes.io/arch":"amd64","node.kubernetes.io/instance-type":"Standard_E4s_v5"}},
			"status":{
				"conditions":[{"type":"MemoryPressure","status":"False"},{"type":"Ready","status":"True"}],
				"nodeInfo":{"kubeletVersion":"v1.34.2"},
				"capacity":{"cpu":"4","memory":"32874620Ki","pods":"250"}
			}
		}
	]}`)
	nodes, err := ParseNodes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].Pool != "system" {
		t.Errorf("nodes[0].Pool = %q, want system", nodes[0].Pool)
	}
	if nodes[0].Status != "Ready" {
		t.Errorf("nodes[0].Status = %q, want Ready", nodes[0].Status)
	}
	if nodes[0].VMSize != "Standard_D4s_v5" {
		t.Errorf("nodes[0].VMSize = %q", nodes[0].VMSize)
	}
	if nodes[0].OS != "linux/amd64" {
		t.Errorf("nodes[0].OS = %q", nodes[0].OS)
	}
}

func TestParseNodeDetail(t *testing.T) {
	input := []byte(`{
		"metadata":{"name":"aks-system-vmss000001","labels":{"agentpool":"system","kubernetes.io/os":"linux","kubernetes.io/arch":"amd64","node.kubernetes.io/instance-type":"Standard_D4s_v5","env":"dev"}},
		"spec":{"taints":[{"key":"agentpool","value":"system","effect":"NoSchedule"}]},
		"status":{
			"conditions":[{"type":"Ready","status":"True"},{"type":"MemoryPressure","status":"False"},{"type":"DiskPressure","status":"False"}],
			"nodeInfo":{"kubeletVersion":"v1.34.2"},
			"capacity":{"cpu":"4","memory":"32874620Ki","pods":"250"}
		}
	}`)
	detail, err := ParseNodeDetail(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Name != "aks-system-vmss000001" {
		t.Errorf("Name = %q", detail.Name)
	}
	if len(detail.Conditions) != 3 {
		t.Fatalf("Conditions = %d, want 3", len(detail.Conditions))
	}
	if detail.Conditions[0].Type != "Ready" {
		t.Errorf("Conditions[0].Type = %q", detail.Conditions[0].Type)
	}
	if len(detail.Taints) != 1 {
		t.Fatalf("Taints = %d, want 1", len(detail.Taints))
	}
	if detail.Taints[0].Effect != "NoSchedule" {
		t.Errorf("Taints[0].Effect = %q", detail.Taints[0].Effect)
	}
	if detail.Labels["env"] != "dev" {
		t.Errorf("Labels[env] = %q", detail.Labels["env"])
	}
}

func TestParseTopNodes(t *testing.T) {
	output := `aks-connect-vmss00001   321m    8%    6211Mi    23%
aks-default-vmss00001   953m    24%   13348Mi   49%
aks-system-vmss00001    681m    17%   4242Mi    35%`

	result := ParseTopNodes(output)
	if len(result) != 3 {
		t.Fatalf("got %d nodes, want 3", len(result))
	}
	if result["aks-connect-vmss00001"].CPUUsage != "321m" {
		t.Errorf("CPUUsage = %q", result["aks-connect-vmss00001"].CPUUsage)
	}
	if result["aks-default-vmss00001"].MemPct != "49%" {
		t.Errorf("MemPct = %q", result["aks-default-vmss00001"].MemPct)
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"32874620Ki", "31.4Gi"},
		{"1048576Ki", "1.0Gi"},
		{"524288Ki", "0.5Gi"},
		{"invalid", "invalid"},
		// Mi suffix
		{"8192Mi", "8.0Gi"},
		{"512Mi", "0.5Gi"},
		// Gi suffix (passthrough)
		{"16Gi", "16Gi"},
		{"1Gi", "1Gi"},
		// Ti suffix
		{"2Ti", "2048Gi"},
		{"1Ti", "1024Gi"},
	}
	for _, tt := range tests {
		got := formatMemory(tt.input)
		if got != tt.want {
			t.Errorf("formatMemory(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "2 hours ago",
			input: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			want:  "2h",
		},
		{
			name:  "3 days ago",
			input: time.Now().Add(-3 * 24 * time.Hour).Format(time.RFC3339),
			want:  "3d",
		},
		{
			name:  "invalid input",
			input: "not-a-timestamp",
			want:  "—",
		},
		{
			name:  "very recent",
			input: time.Now().Add(-30 * time.Second).Format(time.RFC3339),
			want:  "0m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.input)
			if got != tt.want {
				t.Errorf("formatAge(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
