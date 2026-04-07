package app

import (
	"testing"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func TestFilteredContainers(t *testing.T) {
	m := Model{
		containers: []config.Container{
			{Name: "nginx-proxy", Image: "nginx:latest", Status: "Up 3 hours"},
			{Name: "postgres-db", Image: "postgres:15", Status: "Exited (0)"},
			{Name: "redis-cache", Image: "redis:7", Status: "Up 1 hour"},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   int
	}{
		{"no filter", "", 3},
		{"by name", "nginx", 1},
		{"by image", "postgres", 1},
		{"by status", "exited", 1},
		{"partial match", "redis", 1},
		{"no match", "mongodb", 0},
		{"case insensitive", "NGINX", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterText = tt.filter
			got := m.filteredContainers()
			if len(got) != tt.want {
				t.Errorf("filteredContainers(%q) = %d results, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestFilteredUpdates(t *testing.T) {
	m := Model{
		updates: []config.Update{
			{Package: "openssl", Version: "3.0.7-1.el9", Type: "security"},
			{Package: "bash", Version: "5.2.15-1.el9", Type: "bugfix"},
			{Package: "kernel", Version: "5.14.0-362.el9", Type: "security"},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   int
	}{
		{"no filter", "", 3},
		{"by package", "openssl", 1},
		{"by type", "security", 2},
		{"by version", "5.14", 1},
		{"no match", "python", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterText = tt.filter
			got := m.filteredUpdates()
			if len(got) != tt.want {
				t.Errorf("filteredUpdates(%q) = %d results, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestFilteredCronJobs(t *testing.T) {
	m := Model{
		cronJobs: []config.CronJob{
			{Schedule: "0 * * * *", Command: "/usr/bin/backup.sh", Source: "user"},
			{Schedule: "*/5 * * * *", Command: "/usr/bin/monitor.sh", Source: "system"},
			{Schedule: "0 3 * * *", Command: "/usr/bin/cleanup.sh", Source: "user"},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   int
	}{
		{"no filter", "", 3},
		{"by command", "backup", 1},
		{"by source", "system", 1},
		{"by schedule", "*/5", 1},
		{"no match", "reboot", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterText = tt.filter
			got := m.filteredCronJobs()
			if len(got) != tt.want {
				t.Errorf("filteredCronJobs(%q) = %d results, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestFilteredDisks(t *testing.T) {
	m := Model{
		disks: []config.Disk{
			{Filesystem: "/dev/sda1", Size: "50G", Used: "30G", Avail: "20G", UsePercent: "60%", Mount: "/"},
			{Filesystem: "/dev/sdb1", Size: "200G", Used: "180G", Avail: "20G", UsePercent: "90%", Mount: "/data"},
			{Filesystem: "/dev/sdc1", Size: "100G", Used: "10G", Avail: "90G", UsePercent: "10%", Mount: "/backup"},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   int
	}{
		{"no filter", "", 3},
		{"by mount", "/data", 1},
		{"by filesystem", "sda", 1},
		{"by usage", "90%", 1},
		{"no match", "/var", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterText = tt.filter
			got := m.filteredDisks()
			if len(got) != tt.want {
				t.Errorf("filteredDisks(%q) = %d results, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestFilteredInterfaces(t *testing.T) {
	m := Model{
		interfaces: []config.NetInterface{
			{Name: "eth0", State: "UP", IPs: "10.0.0.1"},
			{Name: "lo", State: "UNKNOWN", IPs: "127.0.0.1"},
			{Name: "eth1", State: "DOWN", IPs: ""},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   int
	}{
		{"no filter", "", 3},
		{"by name", "eth0", 1},
		{"by state", "down", 1},
		{"by IP", "10.0.0", 1},
		{"partial name", "eth", 2},
		{"no match", "wlan", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterText = tt.filter
			got := m.filteredInterfaces()
			if len(got) != tt.want {
				t.Errorf("filteredInterfaces(%q) = %d results, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestFilteredRoutes(t *testing.T) {
	m := Model{
		routes: []config.Route{
			{Destination: "default", Gateway: "10.0.0.1", Interface: "eth0", Metric: "100", IsDefault: true},
			{Destination: "10.0.0.0/24", Gateway: "direct", Interface: "eth0", Metric: "—", IsDefault: false},
			{Destination: "172.16.0.0/16", Gateway: "10.0.0.254", Interface: "eth1", Metric: "200", IsDefault: false},
		},
	}

	tests := []struct {
		name   string
		filter string
		want   int
	}{
		{"no filter", "", 3},
		{"by destination", "172.16", 1},
		{"by gateway", "direct", 1},
		{"by interface", "eth1", 1},
		{"default route", "default", 1},
		{"no match", "wlan", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.filterText = tt.filter
			got := m.filteredRoutes()
			if len(got) != tt.want {
				t.Errorf("filteredRoutes(%q) = %d results, want %d", tt.filter, len(got), tt.want)
			}
		})
	}
}

func TestSortIndicator(t *testing.T) {
	tests := []struct {
		name       string
		sortColumn int
		sortAsc    bool
		col        int
		want       string
	}{
		{"no sort", 0, true, 1, ""},
		{"different column", 2, true, 1, ""},
		{"ascending", 1, true, 1, " \u25b2"},
		{"descending", 1, false, 1, " \u25bc"},
		{"col 3 asc", 3, true, 3, " \u25b2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{sortColumn: tt.sortColumn, sortAsc: tt.sortAsc}
			got := m.sortIndicator(tt.col)
			if got != tt.want {
				t.Errorf("sortIndicator(%d) = %q, want %q", tt.col, got, tt.want)
			}
		})
	}
}

func TestSortServices(t *testing.T) {
	m := Model{
		view: viewServiceList,
		services: []config.Service{
			{Name: "nginx", State: "running", Enabled: "enabled", Description: "Web server"},
			{Name: "apache", State: "failed", Enabled: "disabled", Description: "HTTP server"},
			{Name: "sshd", State: "running", Enabled: "enabled", Description: "SSH daemon"},
		},
	}

	// sort by name ascending
	m.sortColumn = 1
	m.sortAsc = true
	m.sortView()
	if m.services[0].Name != "apache" || m.services[2].Name != "sshd" {
		t.Errorf("sort by name asc: got %s, %s, %s", m.services[0].Name, m.services[1].Name, m.services[2].Name)
	}

	// sort by name descending
	m.sortAsc = false
	m.sortView()
	if m.services[0].Name != "sshd" || m.services[2].Name != "apache" {
		t.Errorf("sort by name desc: got %s, %s, %s", m.services[0].Name, m.services[1].Name, m.services[2].Name)
	}

	// sort by state ascending
	m.sortColumn = 2
	m.sortAsc = true
	m.sortView()
	if m.services[0].State != "failed" {
		t.Errorf("sort by state asc: first should be failed, got %s", m.services[0].State)
	}

	// no sort (column 0) — should not change order
	before := make([]config.Service, len(m.services))
	copy(before, m.services)
	m.sortColumn = 0
	m.sortView()
	for i := range before {
		if before[i].Name != m.services[i].Name {
			t.Errorf("sortColumn=0 changed order")
			break
		}
	}
}

func TestSortContainers(t *testing.T) {
	m := Model{
		view: viewContainerList,
		containers: []config.Container{
			{Name: "redis", Image: "redis:7", Status: "Up 1h"},
			{Name: "api", Image: "app:latest", Status: "Exited (1)"},
			{Name: "nginx", Image: "nginx:latest", Status: "Up 3h"},
		},
		sortColumn: 1,
		sortAsc:    true,
	}
	m.sortView()
	if m.containers[0].Name != "api" || m.containers[2].Name != "redis" {
		t.Errorf("sort containers by name asc: got %s, %s, %s", m.containers[0].Name, m.containers[1].Name, m.containers[2].Name)
	}
}

func TestSortPorts(t *testing.T) {
	m := Model{
		view: viewNetworkPorts,
		ports: []config.ListeningPort{
			{Port: 443, Protocol: "tcp", Process: "nginx"},
			{Port: 22, Protocol: "tcp", Process: "sshd"},
			{Port: 80, Protocol: "tcp", Process: "nginx"},
		},
		sortColumn: 1,
		sortAsc:    true,
	}
	m.sortView()
	if m.ports[0].Port != 22 || m.ports[2].Port != 443 {
		t.Errorf("sort ports by port asc: got %d, %d, %d", m.ports[0].Port, m.ports[1].Port, m.ports[2].Port)
	}
}
