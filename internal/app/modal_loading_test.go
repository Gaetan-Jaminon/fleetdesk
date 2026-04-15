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
	showLoading(m, "Loading services...")
	if m.modal == nil {
		t.Fatal("showLoading should set m.modal")
	}
	if m.modal.Done() {
		t.Error("modal should not be done")
	}
	// Verify content is LoadingContent
	if m.modal.current >= len(m.modal.steps) {
		t.Fatal("modal has no steps")
	}
	lc, ok := m.modal.steps[m.modal.current].Content.(*LoadingContent)
	if !ok {
		t.Fatal("modal content should be *LoadingContent")
	}
	if lc.message != "Loading services..." {
		t.Errorf("message = %q, want %q", lc.message, "Loading services...")
	}
}

func TestDismissLoading_ClearsLoadingModal(t *testing.T) {
	m := &Model{}
	showLoading(m, "Loading services...")
	dismissLoading(m)
	if m.modal != nil {
		t.Error("dismissLoading should clear loading modal")
	}
}

func TestDismissLoading_DoesNotClearConfirmModal(t *testing.T) {
	m := &Model{}
	m.modal = NewConfirmModal("Confirm", "Delete? [Y/n]", nil)
	dismissLoading(m)
	if m.modal == nil {
		t.Error("dismissLoading should not clear confirm modal")
	}
}

func TestDismissLoading_DoesNotClearAboutModal(t *testing.T) {
	m := &Model{}
	m.modal, _ = NewAboutModal("0.10.0", "abc1234")
	dismissLoading(m)
	if m.modal == nil {
		t.Error("dismissLoading should not clear about modal")
	}
}

func TestDismissLoading_DoesNotClearStaticContentModal(t *testing.T) {
	m := &Model{}
	m.modal = NewModalOverlay("Help", []ModalStep{
		{Title: "", Content: NewStaticContent("help text")},
	}, func(_ []any) tea.Cmd { return nil },
		func() tea.Cmd { return nil })
	dismissLoading(m)
	if m.modal == nil {
		t.Error("dismissLoading should not clear help modal")
	}
}

func TestDismissLoading_NilModalIsNoOp(t *testing.T) {
	m := &Model{}
	dismissLoading(m) // should not panic
}

func TestDismissLoading_DoneModalIsNoOp(t *testing.T) {
	m := &Model{}
	showLoading(m, "Loading...")
	// Manually mark done
	m.modal.done = true
	dismissLoading(m)
	// Should not clear a done modal
	if m.modal == nil {
		t.Error("dismissLoading should not clear a done modal")
	}
}
