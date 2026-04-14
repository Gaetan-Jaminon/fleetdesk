package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func TestDeployKeyKeybind(t *testing.T) {
	t.Run("K on online host shows confirm prompt", func(t *testing.T) {
		m := newTestModel()
		m.view = viewHostList
		m.hosts = []config.Host{{
			Entry:  config.HostEntry{Name: "host1", Hostname: "10.0.0.1", User: "ansible", Port: 22},
			Status: config.HostOnline,
		}}
		m.hostCursor = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}
		result, cmd := m.handleHostListKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd — confirm prompt should not fire command yet")
		}
		if !m2.showConfirm {
			t.Error("expected showConfirm = true")
		}
		if m2.pendingHandover == nil {
			t.Error("expected pendingHandover to be set")
		}
	})

	t.Run("K confirm yes executes handover", func(t *testing.T) {
		m := newTestModel()
		m.view = viewHostList
		m.hosts = []config.Host{{
			Entry:  config.HostEntry{Name: "host1", Hostname: "10.0.0.1", User: "ansible", Port: 22},
			Status: config.HostOnline,
		}}
		m.hostCursor = 0
		m.selectedHost = 0
		m.showConfirm = true
		m.confirmMessage = "Deploy SSH key to ansible@host1? [Y/n]"
		m.pendingHandover = func() tea.Msg { return nil } // stub

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd == nil {
			t.Error("expected non-nil cmd after confirming deploy key")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false after confirm")
		}
		if m2.pendingHandover != nil {
			t.Error("expected pendingHandover to be cleared after confirm")
		}
	})

	t.Run("K confirm no cancels", func(t *testing.T) {
		m := newTestModel()
		m.view = viewHostList
		m.hosts = []config.Host{{
			Entry:  config.HostEntry{Name: "host1", Hostname: "10.0.0.1", User: "ansible", Port: 22},
			Status: config.HostOnline,
		}}
		m.hostCursor = 0
		m.showConfirm = true
		m.pendingHandover = func() tea.Msg { return nil } // stub

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd after cancelling")
		}
		if m2.flash != "Cancelled" {
			t.Errorf("flash = %q, want %q", m2.flash, "Cancelled")
		}
		if m2.pendingHandover != nil {
			t.Error("expected pendingHandover to be cleared after cancel")
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
