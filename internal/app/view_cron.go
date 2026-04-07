package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderCronList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredCronJobs()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Cron Jobs"
	s := m.renderHeader(breadcrumb, m.cronCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.cronJobs == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No cron jobs found.", iw, normalRowStyle) + "\n"
	} else {
		schedCol := len("SCHEDULE")
		srcCol := len("SOURCE")
		for _, j := range filtered {
			if len(j.Schedule) > schedCol {
				schedCol = len(j.Schedule)
			}
			if len(j.Source) > srcCol {
				srcCol = len(j.Source)
			}
		}
		schedCol += 2
		srcCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", schedCol, "SCHEDULE"+m.sortIndicator(1), srcCol, "SOURCE"+m.sortIndicator(2), "COMMAND"+m.sortIndicator(3))
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
		if m.cronCursor >= offset+maxVisible {
			offset = m.cronCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		lastGroup := ""
		for i := offset; i < end; i++ {
			j := filtered[i]

			// group header: crontab = User, anything else = System
			group := "System"
			if j.Source == "crontab" {
				group = "User"
			}
			if group != lastGroup {
				groupLine := fmt.Sprintf("  \u2500\u2500 %s \u2500\u2500", group)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
				lastGroup = group
			}

			cur := "   "
			if i == m.cronCursor {
				cur = " \u25b8 "
			}
			line := fmt.Sprintf("%s  %-*s  %-*s  %s", cur, schedCol, j.Schedule, srcCol, j.Source, j.Command)

			var style lipgloss.Style
			if i == m.cronCursor {
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
		{"1-3", "Sort"},
		{"/", "Search"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
