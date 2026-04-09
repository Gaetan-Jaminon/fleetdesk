package azure

import (
	"reflect"
	"testing"
)

func TestParseAKSPowerStates(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
		want    map[string]string
	}{
		{
			name: "multiple clusters with different states",
			input: []byte(`{
				"count": 3,
				"data": [
					{"name": "aks-dev-01", "provisioningState": "Succeeded"},
					{"name": "aks-qua-01", "provisioningState": "Starting"},
					{"name": "AKS-PRD-01", "provisioningState": "Stopping"}
				]
			}`),
			want: map[string]string{
				"aks-dev-01": "succeeded",
				"aks-qua-01": "starting",
				"aks-prd-01": "stopping",
			},
		},
		{
			name:  "empty result",
			input: []byte(`{"count": 0, "data": []}`),
			want:  map[string]string{},
		},
		{
			name:    "malformed JSON",
			input:   []byte(`not json`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAKSPowerStates(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d states, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseGraphAKSWithTags(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantTags map[string]string
	}{
		{
			name: "tags parsed correctly",
			input: []byte(`{
				"data": [{
					"name": "aks-dev-01",
					"resourceGroup": "rg-dev",
					"location": "westeurope",
					"id": "/subscriptions/xxx/aks-dev-01",
					"k8sVersion": "1.34",
					"powerState": "Running",
					"provisioningState": "Succeeded",
					"networkPlugin": "azure",
					"createdDate": "2026-03-17T14:59:29Z",
					"pools": [],
					"tags": {
						"release": "release-2026-2-1",
						"Team": "SYS",
						"created_Date": "2026-03-17T14:59:29Z"
					}
				}]
			}`),
			wantTags: map[string]string{
				"release":      "release-2026-2-1",
				"Team":         "SYS",
				"created_Date": "2026-03-17T14:59:29Z",
			},
		},
		{
			name: "null tag value is skipped",
			input: []byte(`{
				"data": [{
					"name": "aks-dev-02",
					"resourceGroup": "rg-dev",
					"location": "westeurope",
					"id": "/subscriptions/xxx/aks-dev-02",
					"k8sVersion": "1.34",
					"powerState": "Running",
					"provisioningState": "Succeeded",
					"networkPlugin": "azure",
					"createdDate": "",
					"pools": [],
					"tags": {
						"env": "dev",
						"owner": null
					}
				}]
			}`),
			wantTags: map[string]string{
				"env": "dev",
			},
		},
		{
			name: "missing tags field results in empty map",
			input: []byte(`{
				"data": [{
					"name": "aks-dev-03",
					"resourceGroup": "rg-dev",
					"location": "westeurope",
					"id": "/subscriptions/xxx/aks-dev-03",
					"k8sVersion": "1.34",
					"powerState": "Running",
					"provisioningState": "Succeeded",
					"networkPlugin": "azure",
					"createdDate": "",
					"pools": []
				}]
			}`),
			wantTags: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters, err := ParseGraphAKS(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(clusters) != 1 {
				t.Fatalf("got %d clusters, want 1", len(clusters))
			}
			got := clusters[0].Tags
			if got == nil {
				got = map[string]string{}
			}
			if !reflect.DeepEqual(got, tt.wantTags) {
				t.Errorf("tags = %v, want %v", got, tt.wantTags)
			}
		})
	}
}
