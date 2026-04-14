package app

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

func newTestModel() Model {
	logger := slog.Default()
	return Model{
		ssh:    ssh.NewManager(logger),
		logger: logger,
		width:  80,
		height: 24,
	}
}

// TestSudoPromptKeyCapture verifies keyboard handling when showSudoPrompt is active.
func TestSudoPromptKeyCapture(t *testing.T) {
	t.Run("rune appended", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		result, _ := m.handleKey(msg)
		m2 := result.(Model)
		if m2.sudoInput != "a" {
			t.Errorf("sudoInput = %q, want %q", m2.sudoInput, "a")
		}
	})

	t.Run("backspace single byte", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		m.sudoInput = "abc"
		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		result, _ := m.handleKey(msg)
		m2 := result.(Model)
		if m2.sudoInput != "ab" {
			t.Errorf("sudoInput = %q, want %q", m2.sudoInput, "ab")
		}
	})

	t.Run("backspace multi-byte rune", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		m.sudoInput = "pàss" // à is 2 bytes in UTF-8
		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		result, _ := m.handleKey(msg)
		m2 := result.(Model)
		if m2.sudoInput != "pàs" {
			t.Errorf("sudoInput = %q, want %q", m2.sudoInput, "pàs")
		}
	})

	t.Run("enter with empty input is no-op", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		m.sudoInput = ""
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.handleKey(msg)
		m2 := result.(Model)
		if !m2.showSudoPrompt {
			t.Error("showSudoPrompt should remain true on empty enter")
		}
	})

	t.Run("enter with password clears prompt", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		m.sudoInput = "mypassword"
		m.hosts = []config.Host{{Entry: config.HostEntry{Name: "host1"}}}
		m.selectedHost = 0
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.handleKey(msg)
		m2 := result.(Model)
		if m2.showSudoPrompt {
			t.Error("showSudoPrompt should be false after entering password")
		}
		if m2.sudoInput != "" {
			t.Errorf("sudoInput should be cleared, got %q", m2.sudoInput)
		}
	})

	t.Run("esc cancels prompt", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		m.sudoInput = "typing"
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ := m.handleKey(msg)
		m2 := result.(Model)
		if m2.showSudoPrompt {
			t.Error("showSudoPrompt should be false after Esc")
		}
		if m2.flash == "" {
			t.Error("flash should be set after Esc")
		}
	})
}

// TestHandleSudoOrFlash verifies the sudo error detection and branching logic.
func TestHandleSudoOrFlash(t *testing.T) {
	retryCmd := func() tea.Msg { return nil }

	t.Run("non-sudo error returns false", func(t *testing.T) {
		m := newTestModel()
		err := errors.New("connection refused")
		_, cmd, ok := m.handleSudoOrFlash(err, retryCmd)
		if ok {
			t.Error("expected ok=false for non-sudo error")
		}
		if cmd != nil {
			t.Error("expected cmd=nil for non-sudo error")
		}
	})

	t.Run("sudo error with no cached passwords shows prompt", func(t *testing.T) {
		m := newTestModel()
		m.hosts = []config.Host{{Entry: config.HostEntry{Name: "host1"}}}
		err := ssh.ErrSudoRequired
		m2, cmd, ok := m.handleSudoOrFlash(err, retryCmd)
		if !ok {
			t.Error("expected ok=true for sudo error")
		}
		if !m2.showSudoPrompt {
			t.Error("expected showSudoPrompt=true when no passwords cached")
		}
		if cmd != nil {
			t.Error("expected cmd=nil when showing prompt directly")
		}
	})

	t.Run("sudo error with cached SSH password returns test cmd", func(t *testing.T) {
		m := newTestModel()
		m.hosts = []config.Host{{Entry: config.HostEntry{Name: "host1"}}}
		m.ssh.SetCachedPassword("sshpw")
		err := ssh.ErrSudoRequired
		m2, cmd, ok := m.handleSudoOrFlash(err, retryCmd)
		if !ok {
			t.Error("expected ok=true for sudo error")
		}
		if m2.showSudoPrompt {
			t.Error("expected showSudoPrompt=false when silently testing SSH password")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd for silent sudo test")
		}
	})

	t.Run("sudo error with wrong cached sudo password clears and prompts", func(t *testing.T) {
		m := newTestModel()
		m.hosts = []config.Host{{Entry: config.HostEntry{Name: "host1"}}}
		m.ssh.SetSudoPassword(0, "wrongpw")
		err := ssh.ErrSudoRequired
		m2, cmd, ok := m.handleSudoOrFlash(err, retryCmd)
		if !ok {
			t.Error("expected ok=true for sudo error")
		}
		if !m2.showSudoPrompt {
			t.Error("expected showSudoPrompt=true after clearing wrong sudo password")
		}
		if cmd != nil {
			t.Error("expected cmd=nil when showing prompt")
		}
		if m2.ssh.GetSudoPassword(0) != "" {
			t.Error("expected sudo password to be cleared")
		}
	})
}

// TestRenderSudoPromptOrHintBar verifies the hint bar rendering with and without sudo prompt.
func TestRenderSudoPromptOrHintBar(t *testing.T) {
	t.Run("sudo prompt active shows masked password and username", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = true
		m.sudoInput = "pw"
		m.hosts = []config.Host{{Entry: config.HostEntry{Name: "host1", User: "alice"}}}
		m.selectedHost = 0
		out := m.renderSudoPromptOrHintBar(nil)
		if !strings.Contains(out, "alice") {
			t.Errorf("expected username 'alice' in output, got: %q", out)
		}
		if !strings.Contains(out, "**") {
			t.Errorf("expected masked password '**' in output, got: %q", out)
		}
	})

	t.Run("no sudo prompt delegates to hint bar", func(t *testing.T) {
		m := newTestModel()
		m.showSudoPrompt = false
		hints := [][]string{{"Enter", "Confirm"}}
		out := m.renderSudoPromptOrHintBar(hints)
		if !strings.Contains(out, "<") {
			t.Errorf("expected hint bar key brackets in output, got: %q", out)
		}
	})
}
