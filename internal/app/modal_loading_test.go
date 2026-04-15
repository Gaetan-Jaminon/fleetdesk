package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLoadingContent_HandleKey_EscNotDismissable(t *testing.T) {
	lc := &LoadingContent{message: "Loading services..."}
	_, _, done := lc.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if done {
		t.Error("Esc should not dismiss LoadingContent")
	}
}

func TestLoadingContent_HandleKey_EnterNotDismissable(t *testing.T) {
	lc := &LoadingContent{message: "Loading services..."}
	_, _, done := lc.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if done {
		t.Error("Enter should not dismiss LoadingContent")
	}
}

func TestLoadingContent_HandleKey_ArbitraryKeyNotDismissable(t *testing.T) {
	lc := &LoadingContent{message: "Loading services..."}
	_, _, done := lc.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if done {
		t.Error("arbitrary key should not dismiss LoadingContent")
	}
}

func TestLoadingContent_View(t *testing.T) {
	lc := &LoadingContent{message: "Loading services..."}
	view := lc.View(80)
	if !strings.Contains(view, "Loading services...") {
		t.Errorf("view should contain message, got: %q", view)
	}
}

func TestLoadingContent_Result(t *testing.T) {
	lc := &LoadingContent{message: "Loading services..."}
	if lc.Result() != nil {
		t.Error("Result should be nil")
	}
}

func TestShowLoading(t *testing.T) {
	m := &Model{}
	showLoading(m, "services", "Loading services...")
	if m.modal == nil {
		t.Fatal("showLoading should set m.modal")
	}
	lc, ok := m.modal.steps[m.modal.current].Content.(*LoadingContent)
	if !ok {
		t.Fatal("modal content should be *LoadingContent")
	}
	if lc.tag != "services" {
		t.Errorf("tag = %q, want %q", lc.tag, "services")
	}
}

func TestDismissLoadingFor_MatchingTag(t *testing.T) {
	m := &Model{}
	showLoading(m, "services", "Loading services...")
	dismissLoadingFor(m, "services")
	if m.modal != nil {
		t.Error("dismissLoadingFor should clear loading modal with matching tag")
	}
}

func TestDismissLoadingFor_WrongTag(t *testing.T) {
	m := &Model{}
	showLoading(m, "subscription", "Loading subscription...")
	dismissLoadingFor(m, "services")
	if m.modal == nil {
		t.Error("dismissLoadingFor should NOT clear loading modal with different tag")
	}
}

func TestDismissLoadingFor_DoesNotClearConfirmModal(t *testing.T) {
	m := &Model{}
	m.modal = NewConfirmModal("Confirm", "Delete? [Y/n]", nil)
	dismissLoadingFor(m, "services")
	if m.modal == nil {
		t.Error("dismissLoadingFor should not clear confirm modal")
	}
}

func TestDismissLoadingFor_NilModalIsNoOp(t *testing.T) {
	m := &Model{}
	dismissLoadingFor(m, "services") // should not panic
}

func TestLoadingModal_EscDoesNotDismissOverlay(t *testing.T) {
	m := &Model{}
	showLoading(m, "services", "Loading services...")
	m.modal.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.modal.Done() {
		t.Error("Esc should not dismiss loading modal through ModalOverlay")
	}
}
