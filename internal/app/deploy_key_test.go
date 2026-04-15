package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func TestDeployKeyKeybind(t *testing.T) {
	t.Run("K on online host shows confirm modal", func(t *testing.T) {
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
			t.Error("expected nil cmd — confirm modal should not fire command yet")
		}
		if m2.modal == nil {
			t.Error("expected modal to be set")
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

		// Set up modal via the K keybind
		result, _ := m.handleHostListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
		m2 := result.(Model)
		if m2.modal == nil {
			t.Fatal("expected modal to be set after K")
		}

		// Confirm with Y
		cmd := m2.modal.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		if cmd == nil {
			t.Error("expected non-nil cmd after confirming deploy key")
		}
		if !m2.modal.Done() {
			t.Error("expected modal to be done after confirm")
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

		// Set up modal via the K keybind
		result, _ := m.handleHostListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
		m2 := result.(Model)

		// Cancel with N
		cmd := m2.modal.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
		if cmd == nil {
			t.Error("expected non-nil cmd from cancel (confirmCancelledMsg)")
		}
		if !m2.modal.Done() {
			t.Error("expected modal to be done after cancel")
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
