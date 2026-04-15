package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderUpdateList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredUpdates()

	// detail view
	if m.showUpdateDetail {
		breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Updates \u203a Detail"
		s := m.renderHeader(breadcrumb, m.updateDetailCursor+1, len(m.updateDetailLines)) + "\n"
		s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

		if len(m.updateDetailLines) == 0 {
			s += borderedRow("  No details available.", iw, normalRowStyle) + "\n"
		} else {
			maxVisible := m.height - 6
			if maxVisible < 1 {
				maxVisible = 1
			}
			offset := 0
			if m.updateDetailCursor >= offset+maxVisible {
				offset = m.updateDetailCursor - maxVisible + 1
			}
			end := offset + maxVisible
			if end > len(m.updateDetailLines) {
				end = len(m.updateDetailLines)
			}
			for i := offset; i < end; i++ {
				cur := "  "
				if i == m.updateDetailCursor {
					cur = "\u25b8 "
				}
				line := "  " + cur + m.updateDetailLines[i]
				var style lipgloss.Style
				if i == m.updateDetailCursor {
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
			{"\u2191\u2193", "Scroll"},
			{"Esc", "Back"},
		})
		return s
	}

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Updates"
	s := m.renderHeader(breadcrumb, m.updateCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.updates == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No pending updates.", iw, normalRowStyle) + "\n"
	} else {
		pkgCol := len("PACKAGE")
		verCol := len("VERSION")
		for _, u := range filtered {
			if len(u.Package) > pkgCol {
				pkgCol = len(u.Package)
			}
			if len(u.Version) > verCol {
				verCol = len(u.Version)
			}
		}
		pkgCol += 2
		verCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", pkgCol, "PACKAGE"+m.sortIndicator(1), verCol, "VERSION"+m.sortIndicator(2), "TYPE"+m.sortIndicator(3))
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
		if m.updateCursor >= offset+maxVisible {
			offset = m.updateCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		lastType := ""
		for i := offset; i < end; i++ {
			u := filtered[i]

			// group header when type changes
			if u.Type != lastType {
				label := strings.ToUpper(u.Type[:1]) + u.Type[1:]
				groupLine := fmt.Sprintf("  \u2500\u2500 %s \u2500\u2500", label)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
				lastType = u.Type
			}

			cur := "   "
			if i == m.updateCursor {
				cur = " \u25b8 "
			}
			line := fmt.Sprintf("%s  %-*s  %-*s  %s", cur, pkgCol, u.Package, verCol, u.Version, u.Type)

			var style lipgloss.Style
			if i == m.updateCursor {
				style = selectedRowStyle
			} else if (u.Type == "security" || u.Type == "error") && i != m.updateCursor {
				style = lipgloss.NewStyle().Foreground(colorRed)
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
		{"u", "Update All"},
		{"p", "Security Only"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
