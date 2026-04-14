package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func subscriptionModel(regType string, extraSubs ...config.Subscription) Model {
	m := newTestModel()
	m.view = viewSubscription
	m.fleets = []config.Fleet{{Name: "test-fleet", Type: "vm"}}
	m.selectedFleet = 0
	m.hosts = []config.Host{{
		Entry:  config.HostEntry{Name: "host1", Hostname: "10.0.0.1", User: "ansible", Port: 22},
		Status: config.HostOnline,
	}}
	m.selectedHost = 0
	m.subscriptions = []config.Subscription{
		{Field: "Registration", Value: regType},
	}
	m.subscriptions = append(m.subscriptions, extraSubs...)
	return m
}

// --- Unregister (u key) ---

func TestUnregisterAction(t *testing.T) {
	t.Run("u on unregistered host flashes error, no confirm", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false")
		}
		if m2.flash != "Host is not registered" {
			t.Errorf("flash = %q, want %q", m2.flash, "Host is not registered")
		}
		if !m2.flashError {
			t.Error("expected flashError = true")
		}
	})

	t.Run("u on empty registration flashes error, no confirm", func(t *testing.T) {
		m := subscriptionModel("")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false")
		}
		if !m2.flashError {
			t.Error("expected flashError = true")
		}
	})

	t.Run("u on CDN host shows confirm, no katello removal", func(t *testing.T) {
		m := subscriptionModel("Red Hat CDN")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if !m2.showConfirm {
			t.Error("expected showConfirm = true")
		}
		if strings.Contains(m2.confirmCmd, "katello") {
			t.Errorf("CDN unregister should not remove katello, got: %s", m2.confirmCmd)
		}
		if !strings.Contains(m2.confirmCmd, "subscription-manager unregister") {
			t.Errorf("expected unregister command, got: %s", m2.confirmCmd)
		}
		if !strings.Contains(m2.confirmCmd, "subscription-manager clean") {
			t.Errorf("expected clean command, got: %s", m2.confirmCmd)
		}
		if !strings.Contains(m2.confirmMessage, "Red Hat CDN") {
			t.Errorf("confirmMessage = %q, should mention registration type", m2.confirmMessage)
		}
	})

	t.Run("u on Satellite host shows confirm with katello removal", func(t *testing.T) {
		m := subscriptionModel("Satellite")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if !m2.showConfirm {
			t.Error("expected showConfirm = true")
		}
		if !strings.Contains(m2.confirmCmd, "katello") {
			t.Errorf("Satellite unregister must remove katello, got: %s", m2.confirmCmd)
		}
		if !strings.Contains(m2.confirmMessage, "Satellite") {
			t.Errorf("confirmMessage = %q, should mention Satellite", m2.confirmMessage)
		}
	})

	t.Run("u confirm yes fires sshHandover", func(t *testing.T) {
		m := subscriptionModel("Red Hat CDN")
		m.showConfirm = true
		m.confirmMessage = "Unregister from Red Hat CDN? [Y/n]"
		m.confirmCmd = "sudo subscription-manager unregister && sudo subscription-manager clean"
		m.confirmBanner = "unregister from Red Hat CDN on host1"

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd == nil {
			t.Error("expected non-nil cmd after confirming unregister")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false after confirm")
		}
	})

	t.Run("u confirm no cancels", func(t *testing.T) {
		m := subscriptionModel("Red Hat CDN")
		m.showConfirm = true
		m.confirmCmd = "sudo subscription-manager unregister && sudo subscription-manager clean"

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd after cancel")
		}
		if m2.flash != "Cancelled" {
			t.Errorf("flash = %q, want %q", m2.flash, "Cancelled")
		}
	})
}

// --- Register CDN (g key) ---

func TestRegisterCDNAction(t *testing.T) {
	t.Run("g shows confirm with pendingHandover set", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd — confirm not yet fired")
		}
		if !m2.showConfirm {
			t.Error("expected showConfirm = true")
		}
		if m2.pendingHandover == nil {
			t.Error("expected pendingHandover to be set")
		}
		if !strings.Contains(m2.confirmMessage, "CDN") {
			t.Errorf("confirmMessage = %q, should mention CDN", m2.confirmMessage)
		}
	})

	t.Run("g is always available (no guard)", func(t *testing.T) {
		for _, regType := range []string{"", "Unknown", "Red Hat CDN", "Satellite"} {
			m := subscriptionModel(regType)
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
			result, _ := m.handleSubscriptionKeys(msg)
			m2 := result.(Model)
			if !m2.showConfirm {
				t.Errorf("regType=%q: expected showConfirm = true", regType)
			}
		}
	})

	t.Run("g confirm yes fires handover", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		m.showConfirm = true
		m.confirmMessage = "Register to Red Hat CDN? [Y/n]"
		m.pendingHandover = func() tea.Msg { return nil } // stub

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd == nil {
			t.Error("expected non-nil cmd after confirming register")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false after confirm")
		}
		if m2.pendingHandover != nil {
			t.Error("expected pendingHandover to be cleared")
		}
	})

	t.Run("g confirm no cancels", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		m.showConfirm = true
		m.pendingHandover = func() tea.Msg { return nil } // stub

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd after cancel")
		}
		if m2.flash != "Cancelled" {
			t.Errorf("flash = %q, want %q", m2.flash, "Cancelled")
		}
	})
}

// --- Hint bar ---

func TestSubscriptionHintBar(t *testing.T) {
	m := subscriptionModel("Red Hat CDN")
	rendered := m.renderSubscription()
	for _, hint := range []string{"u", "Unregister", "g", "Register"} {
		if !strings.Contains(rendered, hint) {
			t.Errorf("hint bar missing %q", hint)
		}
	}
}
