package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderFailedLogins() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredFailedLogins()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Failed Logins"
	s := m.renderHeader(breadcrumb, m.failedLoginCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.failedLogins == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No failed logins found.", iw, normalRowStyle) + "\n"
	} else {
		timeCol := len("TIME") + 2
		userCol := len("USER") + 2
		sourceCol := len("SOURCE") + 2
		methodCol := len("METHOD") + 2
		for _, fl := range filtered {
			if len(fl.Time)+2 > timeCol {
				timeCol = len(fl.Time) + 2
			}
			if len(fl.User)+2 > userCol {
				userCol = len(fl.User) + 2
			}
			if len(fl.Source)+2 > sourceCol {
				sourceCol = len(fl.Source) + 2
			}
			if len(fl.Method)+2 > methodCol {
				methodCol = len(fl.Method) + 2
			}
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s", timeCol, "TIME"+m.sortIndicator(1), userCol, "USER"+m.sortIndicator(2), sourceCol, "SOURCE"+m.sortIndicator(3), methodCol, "METHOD"+m.sortIndicator(4))
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
		if m.failedLoginCursor >= offset+maxVisible {
			offset = m.failedLoginCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			fl := filtered[i]
			cur := "   "
			if i == m.failedLoginCursor {
				cur = " \u25b8 "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s", cur, timeCol, fl.Time, userCol, fl.User, sourceCol, fl.Source, methodCol, fl.Method)

			var style lipgloss.Style
			if i == m.failedLoginCursor {
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
	s += m.renderHintBar(hintWithHelp([][]string{
		{"\u2191\u2193", "Navigate"},
		{"1-4", "Sort"},
		{"/", "Search"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	}))
	return s
}
