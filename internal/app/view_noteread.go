package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) renderNoteRead() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := "Notes \u203a " + m.noteRef.Key() + " \u203a Read"
	s := m.renderHeader(breadcrumb, m.noteReadOffset+1, len(m.noteReadLines)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 1
	}

	if len(m.noteReadLines) == 0 {
		s += borderedRow("  (empty note)", iw, normalRowStyle) + "\n"
	} else {
		end := m.noteReadOffset + maxVisible
		if end > len(m.noteReadLines) {
			end = len(m.noteReadLines)
		}
		for i := m.noteReadOffset; i < end; i++ {
			s += borderedRow("  "+m.noteReadLines[i], iw, normalRowStyle) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"\u2191\u2193", "Scroll"},
		{"Esc", "Back"},
	}))
	return s
}

func (m Model) handleNoteReadKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 1
	}

	switch msg.String() {
	case "up", "k":
		if m.noteReadOffset > 0 {
			m.noteReadOffset--
		}
	case "down", "j":
		if m.noteReadOffset < len(m.noteReadLines)-maxVisible {
			m.noteReadOffset++
		}
	case "esc":
		m.view = viewNoteList
		m.noteReadLines = nil
		m.noteReadOffset = 0
	}
	return m, nil
}
