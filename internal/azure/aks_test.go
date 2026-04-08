package azure

import "testing"

func TestParseGraphAKS(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantCount int
		wantErr   bool
		check     func(t *testing.T, clusters []AKSDetail)
	}{
		{
			name:      "empty result",
			input:     []byte(`{"count": 0, "data": []}`),
			wantCount: 0,
		},
		{
			name: "single cluster with multiple pools",
			input: []byte(`{
				"count": 1,
				"data": [{
					"name": "AKS-APP-DEV-GREEN",
					"resourceGroup": "rg-app-dev",
					"location": "westeurope",
					"id": "/subscriptions/xxx/resourceGroups/rg-app-dev/providers/Microsoft.ContainerService/managedClusters/AKS-APP-DEV-GREEN",
					"k8sVersion": "1.34",
					"powerState": "Running",
					"networkPlugin": "azure",
					"pools": [
						{"name": "system", "mode": "System", "vmSize": "Standard_D4s_v5", "count": 3, "minCount": 3, "maxCount": 20, "currentOrchestratorVersion": "1.34.2", "enableAutoScaling": true},
						{"name": "default", "mode": "User", "vmSize": "Standard_E4s_v5", "count": 11, "minCount": 10, "maxCount": 20, "currentOrchestratorVersion": "1.34.2", "enableAutoScaling": true},
						{"name": "connect", "mode": "User", "vmSize": "Standard_E4s_v5", "count": 5, "minCount": 5, "maxCount": 20, "currentOrchestratorVersion": "1.34.2", "enableAutoScaling": true}
					]
				}]
			}`),
			wantCount: 1,
			check: func(t *testing.T, clusters []AKSDetail) {
				c := clusters[0]
				if c.Name != "AKS-APP-DEV-GREEN" {
					t.Errorf("Name = %q", c.Name)
				}
				if c.KubernetesVersion != "1.34" {
					t.Errorf("KubernetesVersion = %q", c.KubernetesVersion)
				}
				if c.NodeCount != 19 {
					t.Errorf("NodeCount = %d, want 19 (3+11+5)", c.NodeCount)
				}
				if c.PowerState != "Running" {
					t.Errorf("PowerState = %q", c.PowerState)
				}
				if c.NetworkPlugin != "azure" {
					t.Errorf("NetworkPlugin = %q", c.NetworkPlugin)
				}
				if len(c.Pools) != 3 {
					t.Fatalf("Pools = %d, want 3", len(c.Pools))
				}
				if c.Pools[0].Name != "system" {
					t.Errorf("Pools[0].Name = %q", c.Pools[0].Name)
				}
				if c.Pools[0].Mode != "System" {
					t.Errorf("Pools[0].Mode = %q", c.Pools[0].Mode)
				}
				if !c.Pools[0].AutoScale {
					t.Error("Pools[0].AutoScale = false, want true")
				}
				if c.Pools[1].Count != 11 {
					t.Errorf("Pools[1].Count = %d, want 11", c.Pools[1].Count)
				}
			},
		},
		{
			name: "cluster with no pools",
			input: []byte(`{
				"count": 1,
				"data": [{
					"name": "aks-empty",
					"resourceGroup": "rg",
					"location": "westeurope",
					"id": "/sub/xxx",
					"k8sVersion": "1.30",
					"powerState": "Stopped",
					"networkPlugin": "kubenet",
					"pools": []
				}]
			}`),
			wantCount: 1,
			check: func(t *testing.T, clusters []AKSDetail) {
				if clusters[0].NodeCount != 0 {
					t.Errorf("NodeCount = %d, want 0", clusters[0].NodeCount)
				}
				if len(clusters[0].Pools) != 0 {
					t.Errorf("Pools = %d, want 0", len(clusters[0].Pools))
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   []byte(`not json`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters, err := ParseGraphAKS(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(clusters) != tt.wantCount {
				t.Fatalf("got %d clusters, want %d", len(clusters), tt.wantCount)
			}
			if tt.check != nil {
				tt.check(t, clusters)
			}
		})
	}
}
