package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAboutContent_View_ShowsStaticFields(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")
	view := ac.View(80)

	if !strings.Contains(view, "0.10.0 (abc1234)") {
		t.Errorf("view should contain version with commit, got:\n%s", view)
	}
	if !strings.Contains(view, repoURL) {
		t.Errorf("view should contain repo URL, got:\n%s", view)
	}
}

func TestAboutContent_View_ShowsLoadingInitially(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")
	view := ac.View(80)

	count := strings.Count(view, "loading...")
	if count != 3 {
		t.Errorf("expected 3 'loading...' fields, got %d in:\n%s", count, view)
	}
}

func TestAboutContent_View_NoCommitSuffix(t *testing.T) {
	ac := NewAboutContent("dev", "none")
	view := ac.View(80)

	if strings.Contains(view, "(none)") {
		t.Error("should not show (none) when commit is 'none'")
	}
	if !strings.Contains(view, "dev") {
		t.Error("should show version 'dev'")
	}
}

func TestAboutContent_UpdateField(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")

	ac.UpdateField("azVersion", "2.67.0")
	ac.UpdateField("azIdentity", "gaetan@example.com")
	ac.UpdateField("kubectl", "v1.31.4")

	view := ac.View(80)
	if !strings.Contains(view, "2.67.0") {
		t.Errorf("expected azVersion=2.67.0, got:\n%s", view)
	}
	if !strings.Contains(view, "gaetan@example.com") {
		t.Errorf("expected azIdentity, got:\n%s", view)
	}
	if !strings.Contains(view, "v1.31.4") {
		t.Errorf("expected kubectl version, got:\n%s", view)
	}
	if strings.Contains(view, "loading...") {
		t.Error("should not contain 'loading...' after all fields updated")
	}
}

func TestAboutContent_UpdateField_NotFound(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")
	ac.UpdateField("azVersion", "not found")
	ac.UpdateField("kubectl", "timeout")

	view := ac.View(80)
	if !strings.Contains(view, "not found") {
		t.Error("should show 'not found' for missing CLI")
	}
	if !strings.Contains(view, "timeout") {
		t.Error("should show 'timeout' for timed-out CLI")
	}
}

func TestAboutContent_HandleKey_EscDismisses(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")
	_, _, done := ac.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !done {
		t.Error("Esc should dismiss AboutContent")
	}
}

func TestAboutContent_HandleKey_EnterDismisses(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")
	_, _, done := ac.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !done {
		t.Error("Enter should dismiss AboutContent")
	}
}

func TestAboutContent_HandleKey_OtherKeyNoOp(t *testing.T) {
	ac := NewAboutContent("0.10.0", "abc1234")
	_, _, done := ac.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if done {
		t.Error("arbitrary key should not dismiss AboutContent")
	}
}

func TestNewAboutModal(t *testing.T) {
	modal, cmd := NewAboutModal("0.10.0", "abc1234")
	if modal == nil {
		t.Fatal("NewAboutModal returned nil modal")
	}
	if modal.Done() {
		t.Error("modal should not be done immediately")
	}
	if modal.title != "About FleetDesk" {
		t.Errorf("title = %q, want 'About FleetDesk'", modal.title)
	}
	if cmd == nil {
		t.Error("NewAboutModal should return a batch command for async fetches")
	}
}

func TestNewAboutModal_View(t *testing.T) {
	modal, _ := NewAboutModal("0.10.0", "abc1234")
	view := modal.View("", 100, 40)
	if !strings.Contains(view, "About FleetDesk") {
		t.Error("modal view should contain title")
	}
	if !strings.Contains(view, "0.10.0 (abc1234)") {
		t.Error("modal view should contain version")
	}
}

func TestAboutFieldMsg_UpdatesModal(t *testing.T) {
	// Simulate the Update() logic for aboutFieldMsg
	modal, _ := NewAboutModal("0.10.0", "abc1234")

	// Simulate receiving an aboutFieldMsg
	ac, ok := modal.steps[modal.current].Content.(*AboutContent)
	if !ok {
		t.Fatal("modal content should be *AboutContent")
	}
	ac.UpdateField("azVersion", "2.67.0")

	view := ac.View(80)
	if !strings.Contains(view, "2.67.0") {
		t.Errorf("expected updated azVersion in view, got:\n%s", view)
	}
}
