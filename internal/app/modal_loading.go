package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// LoadingContent is a non-dismissable modal step that shows a loading message.
// It swallows all key events — only dismissed programmatically via dismissLoading.
type LoadingContent struct {
	message string
}

func (l *LoadingContent) HandleKey(msg tea.KeyMsg) (StepContent, tea.Cmd, bool) {
	return l, nil, false
}

func (l *LoadingContent) View(width int) string {
	return "  " + l.message
}

func (l *LoadingContent) Result() any {
	return nil
}

// showLoading sets a loading modal overlay on the model.
func showLoading(m *Model, message string) {
	m.modal = NewModalOverlay("", []ModalStep{
		{Title: "", Content: &LoadingContent{message: message}},
	}, func(_ []any) tea.Cmd { return nil },
		func() tea.Cmd { return nil })
	m.modal.FooterFn = func() string { return "" }
}

// dismissLoading clears the modal only if it is currently a loading modal.
// Other modal types (sudo, confirm, about, help) are left untouched.
func dismissLoading(m *Model) {
	if m.modal == nil || m.modal.Done() {
		return
	}
	if m.modal.current < len(m.modal.steps) {
		if _, ok := m.modal.steps[m.modal.current].Content.(*LoadingContent); ok {
			m.modal = nil
		}
	}
}
