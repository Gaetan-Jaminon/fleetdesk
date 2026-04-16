package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBuildDnfCommand_NoFlags(t *testing.T) {
	cmd := buildDnfCommand("sudo dnf update", nil)
	if !strings.Contains(cmd, "sudo dnf update") {
		t.Error("should contain base command")
	}
	if !strings.Contains(cmd, "-y") {
		t.Error("should always contain -y")
	}
	if !strings.Contains(cmd, "--setopt=skip_if_unavailable=1") {
		t.Error("should always contain --setopt=skip_if_unavailable=1")
	}
	if !strings.Contains(cmd, "Press Enter to return") {
		t.Error("should always contain echo suffix")
	}
}

func TestBuildDnfCommand_OneFlag(t *testing.T) {
	cmd := buildDnfCommand("sudo dnf update", []string{"--allowerasing"})
	if !strings.Contains(cmd, "--allowerasing") {
		t.Error("should contain the flag")
	}
	// Flag should appear before -y
	flagIdx := strings.Index(cmd, "--allowerasing")
	yIdx := strings.Index(cmd, "-y")
	if flagIdx > yIdx {
		t.Error("flag should appear before -y")
	}
}

func TestBuildDnfCommand_MultipleFlags(t *testing.T) {
	cmd := buildDnfCommand("sudo dnf update", []string{"--allowerasing", "--skip-broken", "--nobest"})
	if !strings.Contains(cmd, "--allowerasing") {
		t.Error("should contain --allowerasing")
	}
	if !strings.Contains(cmd, "--skip-broken") {
		t.Error("should contain --skip-broken")
	}
	if !strings.Contains(cmd, "--nobest") {
		t.Error("should contain --nobest")
	}
	// Order should be deterministic: allowerasing before skip-broken before nobest
	i1 := strings.Index(cmd, "--allowerasing")
	i2 := strings.Index(cmd, "--skip-broken")
	i3 := strings.Index(cmd, "--nobest")
	if i1 > i2 || i2 > i3 {
		t.Errorf("flags out of order: allowerasing@%d, skip-broken@%d, nobest@%d", i1, i2, i3)
	}
}

func TestBuildDnfCommand_SecurityBase(t *testing.T) {
	cmd := buildDnfCommand("sudo dnf update --security", []string{"--nobest"})
	if !strings.Contains(cmd, "sudo dnf update --security") {
		t.Error("should contain security base")
	}
	if !strings.Contains(cmd, "--nobest") {
		t.Error("should contain --nobest")
	}
}

func TestBuildDnfCommand_EmptyFlags(t *testing.T) {
	cmd := buildDnfCommand("sudo dnf update", []string{})
	// Should not have double spaces
	if strings.Contains(cmd, "  ") {
		t.Error("empty flags should not produce double spaces")
	}
}

// --- Full flow tests for NewOptionsConfirmModal ---

func TestNewOptionsConfirmModal_ConfirmPath(t *testing.T) {
	var gotCmd string
	opts := []MultiSelectOption{
		{Key: "--allowerasing", Label: "--allowerasing", Description: "replace conflicts"},
		{Key: "--skip-broken", Label: "--skip-broken", Description: "skip failures"},
	}

	m := NewOptionsConfirmModal(
		"Update Options",
		"Select flags",
		opts,
		func(keys []string) string {
			return buildDnfCommand("sudo dnf update", keys)
		},
		func(cmd string) tea.Cmd {
			return func() tea.Msg {
				gotCmd = cmd
				return confirmCancelledMsg{} // dummy msg for test
			}
		},
		func() tea.Cmd { return nil },
	)

	// Step 1: toggle first option, then Enter
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}) // toggle --allowerasing
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})                     // advance to step 2

	if m.current != 1 {
		t.Fatalf("expected step 1 (confirm), got step %d", m.current)
	}

	// Step 2: confirm with Y
	cmd := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if cmd == nil {
		t.Fatal("Y on step 2 should return a cmd")
	}
	// Execute the cmd to capture gotCmd
	cmd()
	if !strings.Contains(gotCmd, "--allowerasing") {
		t.Errorf("resolved cmd should contain --allowerasing, got %q", gotCmd)
	}
}

func TestNewOptionsConfirmModal_NCancels(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "--nobest", Label: "--nobest", Description: "allow older"},
	}

	cancelled := false
	m := NewOptionsConfirmModal(
		"Update Options",
		"Select flags",
		opts,
		func(keys []string) string { return "test cmd" },
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd {
			cancelled = true
			return nil
		},
	)

	// Step 1: Enter (no flags selected)
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	// Step 2: N to cancel
	cmd := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	if m.Done() != true {
		t.Error("modal should be done after N")
	}
	// The cmd should produce confirmCancelledMsg
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(confirmCancelledMsg); !ok {
			t.Errorf("N should produce confirmCancelledMsg, got %T", msg)
		}
	}
	_ = cancelled
}

func TestNewOptionsConfirmModal_EscStep2GoesBack(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}

	m := NewOptionsConfirmModal(
		"Test", "Pick", opts,
		func(keys []string) string { return "cmd" },
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Step 1: toggle and Enter
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.current != 1 {
		t.Fatalf("expected step 1, got %d", m.current)
	}

	// Esc from step 2 should go back to step 1
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.current != 0 {
		t.Errorf("expected step 0 after Esc, got %d", m.current)
	}
}

func TestNewOptionsConfirmModal_SelectionPreservedOnBack(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
	}

	m := NewOptionsConfirmModal(
		"Test", "Pick", opts,
		func(keys []string) string { return "cmd" },
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Step 1: toggle first, Enter
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})

	// Esc back to step 1
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})

	// Check selection is preserved via the optionsStep adapter
	step := m.steps[0].Content.(*optionsStep)
	result := step.inner.Result().([]string)
	if len(result) != 1 || result[0] != "a" {
		t.Errorf("selection should be preserved after Esc-back, got %v", result)
	}
}

func TestNewOptionsConfirmModal_EscStep1Cancels(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}

	cancelled := false
	m := NewOptionsConfirmModal(
		"Test", "Pick", opts,
		func(keys []string) string { return "cmd" },
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd {
			cancelled = true
			return nil
		},
	)

	// Esc at step 0
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !cancelled {
		t.Error("Esc at step 0 should call onCancel")
	}
}

func TestNewOptionsConfirmModal_Step2MessageContainsCmd(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "--nobest", Label: "--nobest", Description: ""},
	}

	m := NewOptionsConfirmModal(
		"Test", "Pick", opts,
		func(keys []string) string {
			return buildDnfCommand("sudo dnf update", keys)
		},
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Toggle --nobest, advance
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2 view should contain the resolved command
	view := m.steps[1].Content.View(80)
	if !strings.Contains(view, "--nobest") {
		t.Errorf("step 2 view should contain --nobest, got %q", view)
	}
}

func TestNewOptionsConfirmModal_FooterStep1(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}

	m := NewOptionsConfirmModal(
		"Test", "Pick", opts,
		func(keys []string) string { return "cmd" },
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	footer := m.FooterFn()
	if !strings.Contains(footer, "Space") {
		t.Error("step 1 footer should contain Space")
	}
	if !strings.Contains(footer, "toggle") {
		t.Error("step 1 footer should contain toggle")
	}
}

func TestNewOptionsConfirmModal_FooterStep2(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}

	m := NewOptionsConfirmModal(
		"Test", "Pick", opts,
		func(keys []string) string { return "cmd" },
		func(cmd string) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Advance to step 2
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	footer := m.FooterFn()
	if !strings.Contains(footer, "Y/Enter") {
		t.Error("step 2 footer should contain Y/Enter")
	}
	if !strings.Contains(footer, "back") {
		t.Error("step 2 footer should contain back")
	}
}
