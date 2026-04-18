package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderContainerList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	// detail view
	if m.showContainerDetail {
		d := m.containerDetail
		name := d.ID
		if len(name) > 12 {
			name = name[:12]
		}
		breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Containers \u203a " + name

		lines := m.containerDetailLines()

		s := m.renderHeader(breadcrumb, m.containerDetailCursor+1, len(lines)) + "\n"
		s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

		maxVisible := m.height - 6
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.containerDetailCursor >= offset+maxVisible {
			offset = m.containerDetailCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(lines) {
			end = len(lines)
		}
		for i := offset; i < end; i++ {
			cur := "  "
			if i == m.containerDetailCursor {
				cur = "\u25b8 "
			}
			line := "  " + cur + lines[i]

			var style lipgloss.Style
			if i == m.containerDetailCursor {
				style = selectedRowStyle
			} else if strings.HasPrefix(lines[i], "---") {
				style = colHeaderStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}

		s = m.padToBottom(s, iw)
		s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
		s += m.renderHintBar(hintWithHelp([][]string{
			{"\u2191\u2193", "Scroll"},
			{"Esc", "Back"},
		}))
		return s
	}

	filtered := m.filteredContainers()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Containers"
	s := m.renderHeader(breadcrumb, m.containerCursor+1, len(filtered)) + "\n"
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

	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "CONTAINER", SortIndex: 1},
			{Label: "IMAGE", SortIndex: 2},
			{Label: "STATUS", SortIndex: 3},
		},
		RowCount: len(filtered),
		RowBuilder: func(i int) []string {
			c := filtered[i]
			return []string{c.Name, c.Image, c.Status}
		},
		RowPrefix: func(i int) string {
			c := filtered[i]
			if !strings.HasPrefix(c.Status, "Up") && !strings.HasPrefix(c.Status, "Exited (0)") && c.Status != "Created" {
				return "\u2717 "
			}
			return ""
		},
		Cursor:        m.containerCursor,
		MaxVisible:    maxVisible,
		InnerWidth:    iw,
		SortIndicator: m.sortIndicator,
		EmptyMessage:  "  No containers found.",
	})

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"↑↓", "Navigate"},
		{"Enter", "Detail"},
		{"1-3", "Sort"},
		{"/", "Search"},
		{"l", "Logs"},
		{"i", "Inspect"},
		{"e", "Exec"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	}))
	return s
}
