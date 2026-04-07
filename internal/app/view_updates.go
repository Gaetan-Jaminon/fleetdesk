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

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Updates"
	s := m.renderHeader(breadcrumb, m.updateCursor+1, len(m.updates)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.updates) == 0 {
		if m.updates == nil {
			s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No pending updates.", iw, normalRowStyle) + "\n"
		}
	} else {
		pkgCol := len("PACKAGE")
		verCol := len("VERSION")
		for _, u := range m.updates {
			if len(u.Package) > pkgCol {
				pkgCol = len(u.Package)
			}
			if len(u.Version) > verCol {
				verCol = len(u.Version)
			}
		}
		pkgCol += 2
		verCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", pkgCol, "PACKAGE", verCol, "VERSION", "TYPE")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.updateCursor >= offset+maxVisible {
			offset = m.updateCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.updates) {
			end = len(m.updates)
		}

		lastType := ""
		for i := offset; i < end; i++ {
			u := m.updates[i]

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

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else {
		s += m.renderHintBar([][]string{
			{"u", "Update All"},
			{"p", "Security Only"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}
