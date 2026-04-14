package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func TestDeployKeyKeybind(t *testing.T) {
	t.Run("K on online host returns exec command", func(t *testing.T) {
		m := newTestModel()
		m.view = viewHostList
		m.hosts = []config.Host{{
			Entry:  config.HostEntry{Name: "host1", Hostname: "10.0.0.1", User: "ansible", Port: 22},
			Status: config.HostOnline,
		}}
		m.hostCursor = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}
		_, cmd := m.handleHostListKeys(msg)
		if cmd == nil {
			t.Error("expected non-nil cmd for K on online host")
		}
	})

	t.Run("K on offline host shows flash error", func(t *testing.T) {
		m := newTestModel()
		m.view = viewHostList
		m.hosts = []config.Host{{
			Entry:  config.HostEntry{Name: "host1", Hostname: "10.0.0.1", User: "ansible", Port: 22},
			Status: config.HostUnreachable,
		}}
		m.hostCursor = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}
		result, cmd := m.handleHostListKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd for K on offline host")
		}
		if m2.flash != "Host is not reachable" {
			t.Errorf("flash = %q, want %q", m2.flash, "Host is not reachable")
		}
		if !m2.flashError {
			t.Error("expected flashError = true")
		}
	})
}
