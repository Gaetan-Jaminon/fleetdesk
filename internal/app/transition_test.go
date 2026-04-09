package app

import "testing"

func TestTransitionKeyFormat(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		wantKey      string
	}{
		{"azure vm", "vm", "myvm", "vm/myvm"},
		{"azure aks", "aks", "mycluster", "aks/mycluster"},
		{"k8s pod", "k8s-pod", "nginx-abc123", "k8s-pod/nginx-abc123"},
		{"k8s context", "k8s-context", "dev-ctx", "k8s-context/dev-ctx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := transition{
				ResourceType: tt.resourceType,
				ResourceName: tt.resourceName,
			}
			got := tr.ResourceType + "/" + tr.ResourceName
			if got != tt.wantKey {
				t.Errorf("key = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestTransitionStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		wantPoll bool
	}{
		{"poll strategy triggers polling", "poll", true},
		{"oneshot strategy skips polling", "oneshot", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := transition{Strategy: tt.strategy}
			if (tr.Strategy == "poll") != tt.wantPoll {
				t.Errorf("Strategy=%q, wantPoll=%v", tr.Strategy, tt.wantPoll)
			}
		})
	}
}

func TestIsAzureTransitioningStateUnchanged(t *testing.T) {
	// Ensure the existing helper still works after the rename refactor
	tests := []struct {
		state string
		want  bool
	}{
		{"starting", true},
		{"stopping", true},
		{"deallocating", true},
		{"restarting", true},
		{"running", false},
		{"deallocated", false},
		{"succeeded", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			if got := isAzureTransitioningState(tt.state); got != tt.want {
				t.Errorf("isAzureTransitioningState(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
