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
		s := m.renderHeader(breadcrumb, 0, 0) + "\n"
		s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

		s += borderedRow("  Details", iw, colHeaderStyle) + "\n"

		items := []struct{ key, val string }{
			{"ID", d.ID},
			{"Image", d.Image},
			{"Status", d.Status},
			{"Created", d.Created},
			{"Command", d.Command},
		}
		for _, item := range items {
			line := fmt.Sprintf("  %-12s  %s", item.key, item.val)
			s += borderedRow(line, iw, normalRowStyle) + "\n"
		}

		if len(d.Ports) > 0 {
			s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
			s += borderedRow("  Ports", iw, colHeaderStyle) + "\n"
			for _, p := range d.Ports {
				s += borderedRow("    "+p, iw, normalRowStyle) + "\n"
			}
		}

		if len(d.Mounts) > 0 {
			s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
			s += borderedRow("  Mounts", iw, colHeaderStyle) + "\n"
			for _, mt := range d.Mounts {
				s += borderedRow("    "+mt, iw, normalRowStyle) + "\n"
			}
		}

		if len(d.Env) > 0 {
			s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
			s += borderedRow("  Environment", iw, colHeaderStyle) + "\n"
			for _, e := range d.Env {
				s += borderedRow("    "+e, iw, normalRowStyle) + "\n"
			}
		}

		s = m.padToBottom(s, iw)
		s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
		s += m.renderHintBar([][]string{
			{"any key", "Back"},
		})
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

	if m.containers == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No containers found.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("CONTAINER")
		imgCol := len("IMAGE")
		for _, c := range filtered {
			if len(c.Name) > nameCol {
				nameCol = len(c.Name)
			}
			if len(c.Image) > imgCol {
				imgCol = len(c.Image)
			}
		}
		nameCol += 2
		imgCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", nameCol, "CONTAINER"+m.sortIndicator(1), imgCol, "IMAGE"+m.sortIndicator(2), "STATUS"+m.sortIndicator(3))
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if m.filterText != "" {
			maxVisible -= 2
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.containerCursor >= offset+maxVisible {
			offset = m.containerCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			c := filtered[i]
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
		{"↑↓", "Navigate"},
		{"Enter", "Detail"},
		{"1-3", "Sort"},
		{"/", "Search"},
		{"l", "Logs"},
		{"i", "Inspect"},
		{"e", "Exec"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
