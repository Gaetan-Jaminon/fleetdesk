package app

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/k8s"
)

// Parity tests for views migrated to the shared renderer in FLE-81.
// Each test drives a real tea.Program and verifies the migrated view renders
// expected columns and data without regressions.

func TestRenderList_Parity_HostList(t *testing.T) {
	m := baselineModel([]config.Fleet{{Name: "test-fleet", Type: "vm"}})
	m.view = viewHostList
	m.selectedFleet = 0
	m.hosts = []config.Host{
		{
			Entry:       config.HostEntry{Name: "host-a"},
			OS:          "RHEL 9.5",
			UpSince:     "5 days",
			UpdateCount: 3,
			Status:      config.HostOnline,
		},
		{
			Entry:       config.HostEntry{Name: "host-b"},
			OS:          "Ubuntu 22.04",
			UpSince:     "2 hours",
			UpdateCount: 0,
			Status:      config.HostOnline,
		},
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))

	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("HOST")) &&
				bytes.Contains(bts, []byte("OS")) &&
				bytes.Contains(bts, []byte("UP SINCE")) &&
				bytes.Contains(bts, []byte("host-a")) &&
				bytes.Contains(bts, []byte("host-b")) &&
				bytes.Contains(bts, []byte("RHEL 9.5")) &&
				bytes.Contains(bts, []byte("Ubuntu 22.04"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestRenderList_Parity_ServiceList_WithFailedPrefix(t *testing.T) {
	m := baselineModel([]config.Fleet{{Name: "test-fleet", Type: "vm"}})
	m.view = viewServiceList
	m.selectedFleet = 0
	m.hosts = []config.Host{{Entry: config.HostEntry{Name: "host-a"}}}
	m.selectedHost = 0
	m.services = []config.Service{
		{Name: "ok-svc", State: "active", Enabled: "enabled", Description: "healthy service"},
		{Name: "bad-svc", State: "failed", Enabled: "enabled", Description: "broken service"},
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(140, 30))

	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("SERVICE")) &&
				bytes.Contains(bts, []byte("STATE")) &&
				bytes.Contains(bts, []byte("ok-svc")) &&
				bytes.Contains(bts, []byte("bad-svc")) &&
				// The ✗ prefix should appear on the failed service row
				bytes.Contains(bts, []byte("\u2717 bad-svc"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestRenderList_Parity_K8sPodList(t *testing.T) {
	m := baselineModel([]config.Fleet{{Name: "test-fleet", Type: "kubernetes"}})
	m.view = viewK8sPodList
	m.selectedFleet = 0
	m.k8sClusters = []k8s.K8sClusterItem{{Name: "test-cluster"}}
	m.selectedK8sCluster = 0
	m.selectedK8sContext = "test-ctx"
	m.k8sNamespaces = []k8s.K8sNamespace{{Name: "default"}}
	m.selectedK8sNamespace = 0
	m.k8sWorkloads = []k8s.K8sWorkload{{Name: "nginx", Kind: "Deployment"}}
	m.selectedK8sWorkload = 0
	m.k8sPodList = []k8s.K8sPod{
		{Name: "nginx-abc", Status: "Running", Ready: "1/1", Restarts: 0, Node: "node-1", Age: "2d"},
		{Name: "nginx-xyz", Status: "Pending", Ready: "0/1", Restarts: 2, Node: "node-2", Age: "5m"},
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(140, 30))

	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("NAME")) &&
				bytes.Contains(bts, []byte("STATUS")) &&
				bytes.Contains(bts, []byte("READY")) &&
				bytes.Contains(bts, []byte("RESTARTS")) &&
				bytes.Contains(bts, []byte("nginx-abc")) &&
				bytes.Contains(bts, []byte("nginx-xyz")) &&
				bytes.Contains(bts, []byte("Running")) &&
				bytes.Contains(bts, []byte("Pending"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
