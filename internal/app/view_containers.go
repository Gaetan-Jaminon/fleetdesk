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

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Containers"
	s := m.renderHeader(breadcrumb, m.containerCursor+1, len(m.containers)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.containers) == 0 {
		s += borderedRow("  No containers found.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("CONTAINER")
		imgCol := len("IMAGE")
		for _, c := range m.containers {
			if len(c.Name) > nameCol {
				nameCol = len(c.Name)
			}
			if len(c.Image) > imgCol {
				imgCol = len(c.Image)
			}
		}
		nameCol += 2
		imgCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", nameCol, "CONTAINER", imgCol, "IMAGE", "STATUS")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.containerCursor >= offset+maxVisible {
			offset = m.containerCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.containers) {
			end = len(m.containers)
		}

		for i := offset; i < end; i++ {
			c := m.containers[i]
			cur := "   "
			if i == m.containerCursor {
				cur = " \u25b8 "
			}
			prefix := ""
			if !strings.HasPrefix(c.Status, "Up") && !strings.HasPrefix(c.Status, "Exited (0)") && c.Status != "Created" {
				prefix = "\u2717 "
			}
			line := fmt.Sprintf("%s  %s%-*s  %-*s  %s", cur, prefix, nameCol, c.Name, imgCol, c.Image, c.Status)

			var style lipgloss.Style
			if i == m.containerCursor {
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
		{"l", "Logs"},
		{"i", "Inspect"},
		{"e", "Exec"},
		{"Esc", "Back"},
	})
	return s
}
