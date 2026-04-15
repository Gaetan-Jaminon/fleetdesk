package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAuditSummary() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredAuditEvents()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Audit Summary"
	s := m.renderHeader(breadcrumb, m.auditCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.auditEvents == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No audit events found.", iw, normalRowStyle) + "\n"
	} else {
		timeCol := len("TIME") + 2
		userCol := len("USER") + 2
		resultCol := len("RESULT") + 2
		for _, ae := range filtered {
			if len(ae.Time)+2 > timeCol {
				timeCol = len(ae.Time) + 2
			}
			if len(ae.User)+2 > userCol {
				userCol = len(ae.User) + 2
			}
			if len(ae.Result)+2 > resultCol {
				resultCol = len(ae.Result) + 2
			}
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %s", timeCol, "TIME"+m.sortIndicator(1), userCol, "USER"+m.sortIndicator(2), resultCol, "RESULT"+m.sortIndicator(3), "MESSAGE"+m.sortIndicator(4))
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
		if m.auditCursor >= offset+maxVisible {
			offset = m.auditCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		failedStyle := lipgloss.NewStyle().Foreground(colorRed)

		for i := offset; i < end; i++ {
			ae := filtered[i]
			cur := "   "
			if i == m.auditCursor {
				cur = " \u25b8 "
			}

			// truncate message to fit
			msgMaxLen := iw - timeCol - userCol - resultCol - 16
			msg := ae.Message
			if msgMaxLen > 0 && len(msg) > msgMaxLen {
				msg = msg[:msgMaxLen-1] + "\u2026"
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %s", cur, timeCol, ae.Time, userCol, ae.User, resultCol, ae.Result, msg)

			var style lipgloss.Style
			if i == m.auditCursor {
				style = selectedRowStyle
			} else if ae.Result == "failed" {
				style = failedStyle
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
