package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderFleetPicker() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader("", m.fleetCursor+1, len(m.fleets)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.fleets) == 0 {
		s += borderedRow("  No fleet files found in ~/.config/fleetdesk/", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("FLEET")
		for _, f := range m.fleets {
			if len(f.Name) > nameCol {
				nameCol = len(f.Name)
			}
		}
		nameCol += 2

		hdr := fmt.Sprintf("     %-*s  %-6s  %s", nameCol, "FLEET", "TYPE", "HOSTS")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.fleetCursor >= offset+maxVisible {
			offset = m.fleetCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.fleets) {
			end = len(m.fleets)
		}

		for i := offset; i < end; i++ {
			f := m.fleets[i]
			cur := "   "
			if i == m.fleetCursor {
				cur = " \u25b8 "
			}

			ftype := f.Type
			if ftype == "" {
				ftype = "vm"
			}
			hostCount := m.fleetHostCount(f)
			line := fmt.Sprintf("%s  %-*s  %-6s  %d", cur, nameCol, f.Name, ftype, hostCount)

			var style lipgloss.Style
			if i == m.fleetCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar([][]string{
		{"Enter", "Select"},
		{"e", "Edit"},
		{"r", "Reload"},
		{"q", "Quit"},
	})
	return s
}
