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
	t.Run("g without config shows flash error", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false without config")
		}
		if !m2.flashError {
			t.Error("expected flashError = true")
		}
	})

	t.Run("g with CDN config registers to CDN", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		m.hosts[0].Entry.RHOrgID = "12345"
		m.hosts[0].Entry.RHActivationKey = "ak-cdn"
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		result, cmd := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if cmd != nil {
			t.Error("expected nil cmd — confirm not yet fired")
		}
		if !m2.showConfirm {
			t.Error("expected showConfirm = true")
		}
		if !strings.Contains(m2.confirmMessage, "Red Hat CDN") {
			t.Errorf("confirmMessage = %q, should mention Red Hat CDN", m2.confirmMessage)
		}
		if strings.Contains(m2.confirmCmd, "katello") {
			t.Errorf("CDN register should not install katello, got: %s", m2.confirmCmd)
		}
	})

	t.Run("g with satellite_url registers to Satellite", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		m.hosts[0].Entry.RHOrgID = "Fluxys"
		m.hosts[0].Entry.RHActivationKey = "ak-sat"
		m.hosts[0].Entry.SatelliteURL = "flxsatprd01.central.fluxys.int"
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		result, _ := m.handleSubscriptionKeys(msg)
		m2 := result.(Model)
		if !strings.Contains(m2.confirmMessage, "Satellite") {
			t.Errorf("confirmMessage = %q, should mention Satellite", m2.confirmMessage)
		}
		if !strings.Contains(m2.confirmCmd, "katello-ca-consumer") {
			t.Errorf("Satellite register should install katello, got: %s", m2.confirmCmd)
		}
		if !strings.Contains(m2.confirmCmd, "--force") {
			t.Errorf("Satellite register should use --force, got: %s", m2.confirmCmd)
		}
	})

	t.Run("g is always available regardless of registration state", func(t *testing.T) {
		for _, regType := range []string{"", "Unknown", "Red Hat CDN", "Satellite"} {
			m := subscriptionModel(regType)
			m.hosts[0].Entry.RHOrgID = "12345"
			m.hosts[0].Entry.RHActivationKey = "ak-test"
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
		m.confirmCmd = "sudo subscription-manager register --org=12345 --activationkey=ak-test"
		m.confirmBanner = "register to Red Hat CDN on host1"

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		result, cmd := m.handleKey(msg)
		m2 := result.(Model)
		if cmd == nil {
			t.Error("expected non-nil cmd after confirming register")
		}
		if m2.showConfirm {
			t.Error("expected showConfirm = false after confirm")
		}
	})

	t.Run("g confirm no cancels", func(t *testing.T) {
		m := subscriptionModel("Unknown")
		m.showConfirm = true
		m.confirmCmd = "sudo subscription-manager register --org=12345 --activationkey=ak-test"

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
	t.Run("CDN host shows Register CDN", func(t *testing.T) {
		m := subscriptionModel("Red Hat CDN")
		rendered := m.renderSubscription()
		if !strings.Contains(rendered, "Unregister") {
			t.Error("hint bar missing Unregister")
		}
		if !strings.Contains(rendered, "Register CDN") {
			t.Error("hint bar missing Register CDN")
		}
	})

	t.Run("Satellite host shows Register Satellite", func(t *testing.T) {
		m := subscriptionModel("Satellite")
		m.hosts[0].Entry.SatelliteURL = "sat.example.com"
		rendered := m.renderSubscription()
		if !strings.Contains(rendered, "Register Satellite") {
			t.Error("hint bar missing Register Satellite")
		}
	})
}
