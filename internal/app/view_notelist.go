package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// filteredNoteList returns indices into m.noteList matching m.filterText
// (case-insensitive substring on the preview). When no filter is active
// all indices are returned in order.
func (m Model) filteredNoteList() []int {
	if m.filterText == "" {
		out := make([]int, len(m.noteList))
		for i := range m.noteList {
			out[i] = i
		}
		return out
	}
	filter := strings.ToLower(m.filterText)
	out := []int{}
	for i, n := range m.noteList {
		if strings.Contains(strings.ToLower(n.Preview), filter) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) renderNoteList() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := "Notes \u203a " + m.noteRef.Key()
	filtered := m.filteredNoteList()

	s := m.renderHeader(breadcrumb, m.noteCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	maxVisible := m.height - 8
	if m.filterText != "" {
		maxVisible -= 2
	}
	if maxVisible < 1 {
		maxVisible = 1
	}

	emptyMsg := "  No notes for this resource. Press n to create one."
	if m.filterText != "" {
		emptyMsg = fmt.Sprintf("  No notes match '%s'", m.filterText)
	}

	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "DATE"},
			{Label: "PREVIEW"},
		},
		RowCount: len(filtered),
		RowBuilder: func(i int) []string {
			n := m.noteList[filtered[i]]
			date := n.CreatedAt.Local().Format("02/01/2006 15:04")
			preview := n.Preview
			if preview == "" {
				preview = "(empty)"
			}
			return []string{date, preview}
		},
		Cursor:       m.noteCursor,
		MaxVisible:   maxVisible,
		InnerWidth:   iw,
		EmptyMessage: emptyMsg,
	})

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s\u2588", m.filterText))
	} else {
		s += m.renderHintBar(hintWithHelp([][]string{
			{"\u2191\u2193", "Navigate"},
			{"Enter", "Read"},
			{"n", "New"},
			{"e", "Edit"},
			{"d", "Delete"},
			{"/", "Filter"},
			{"Esc", "Back"},
		}))
	}
	return s
}

func (m Model) handleNoteListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredNoteList()

	switch msg.String() {
	case "up", "k":
		if m.noteCursor > 0 {
			m.noteCursor--
		}
	case "down", "j":
		if m.noteCursor < len(filtered)-1 {
			m.noteCursor++
		}
	case "/":
		m.filterActive = true
		m.noteCursor = 0
	case "esc":
		m.view = m.previousView
		m.filterText = ""
	case "n":
		// Create a new note for the current resource.
		return m, m.createNoteCmd(m.noteRef)
	case "enter":
		if m.noteCursor < len(filtered) {
			path := m.noteList[filtered[m.noteCursor]].Path
			return m, m.loadNoteReadCmd(path)
		}
	case "e":
		if m.noteCursor < len(filtered) {
			path := m.noteList[filtered[m.noteCursor]].Path
			return m, m.editNoteCmd(path, false)
		}
	case "d":
		if m.noteCursor >= len(filtered) {
			return m, nil
		}
		note := m.noteList[filtered[m.noteCursor]]
		preview := note.Preview
		if preview == "" {
			preview = "(empty)"
		}
		date := note.CreatedAt.Local().Format("02/01/2006 15:04")
		path := note.Path
		m.modal = NewConfirmModal(
			"Delete note",
			fmt.Sprintf("Delete note from %s?\n\n  %s", date, preview),
			func() tea.Msg {
				return noteDeleteConfirmedMsg{path: path}
			},
		)
	}
	return m, nil
}

// noteDeleteConfirmedMsg is sent from the delete confirm modal.
type noteDeleteConfirmedMsg struct {
	path string
}
