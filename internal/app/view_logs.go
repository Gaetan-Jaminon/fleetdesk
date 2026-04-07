package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderLogLevelPicker() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Logs"
	s := m.renderHeader(breadcrumb, m.logLevelCursor+1, len(m.logLevels)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.logLevels) == 0 {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("LEVEL") + 6

		hdr := fmt.Sprintf("     %-*s  %8s", nameCol, "LEVEL", "COUNT")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		for i, l := range m.logLevels {
			cur := "   "
			if i == m.logLevelCursor {
				cur = " \u25b8 "
			}
			line := fmt.Sprintf("%s  %-*s  %8d", cur, nameCol, l.Level, l.Count)

			var style lipgloss.Style
			if i == m.logLevelCursor {
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
		{"Enter", "View Logs"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}

func (m Model) renderErrorLogList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredErrorLogs()

	// log detail view
	if m.showLogDetail && m.errorCursor < len(filtered) {
		e := filtered[m.errorCursor]
		breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Log Detail"
		s := m.renderHeader(breadcrumb, 0, 0) + "\n"
		s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

		s += borderedRow(fmt.Sprintf("  Time: %s", e.Time), iw, colHeaderStyle) + "\n"
		s += borderedRow(fmt.Sprintf("  Unit: %s", e.Unit), iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		// parse structured key=value fields from the message
		kvPairs := parseLogFields(e.Message)
		if len(kvPairs) > 0 {
			fieldCol := 0
			for _, kv := range kvPairs {
				if len(kv[0]) > fieldCol {
					fieldCol = len(kv[0])
				}
			}
			fieldCol += 2

			for i, kv := range kvPairs {
				line := fmt.Sprintf("  %-*s  %s", fieldCol, kv[0], kv[1])
				var style lipgloss.Style
				if kv[0] == "level" && (kv[1] == "error" || kv[1] == "crit") {
					style = lipgloss.NewStyle().Foreground(colorRed)
				} else if kv[0] == "err" || kv[0] == "error" {
					style = lipgloss.NewStyle().Foreground(colorRed)
				} else if i%2 == 0 {
					style = altRowStyle
				} else {
					style = normalRowStyle
				}
				s += borderedRow(line, iw, style) + "\n"
			}
		} else {
			// plain message -- word-wrap
			msg := e.Message
			lineWidth := iw - 4
			for len(msg) > 0 {
				end := lineWidth
				if end > len(msg) {
					end = len(msg)
				}
				s += borderedRow("  "+msg[:end], iw, normalRowStyle) + "\n"
				msg = msg[end:]
			}
		}

		s = m.padToBottom(s, iw)
		s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
		s += hintBarStyle.Width(m.width).Render("  Press any key to return")
		return s
	}

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Logs"
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	s := m.renderHeader(breadcrumb+filterInfo, m.errorCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.errorLogs == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No errors found.", iw, normalRowStyle) + "\n"
		}
	} else {
		timeCol := len("TIME")
		unitCol := len("UNIT")
		for _, e := range filtered {
			if len(e.Time) > timeCol {
				timeCol = len(e.Time)
			}
			if len(e.Unit) > unitCol {
				unitCol = len(e.Unit)
			}
		}
		timeCol += 2
		if unitCol > 40 {
			unitCol = 40
		}
		unitCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", timeCol, "TIME", unitCol, "UNIT", "MESSAGE")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.errorCursor >= offset+maxVisible {
			offset = m.errorCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			e := filtered[i]
			cur := "   "
			if i == m.errorCursor {
				cur = " \u25b8 "
			}
			unit := e.Unit
			if len(unit) > unitCol-2 {
				unit = unit[:unitCol-3] + "\u2026"
			}
			line := fmt.Sprintf("%s  %-*s  %-*s  %s", cur, timeCol, e.Time, unitCol, unit, e.Message)

			var style lipgloss.Style
			if i == m.errorCursor {
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

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s\u2588", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"Enter", "Detail"},
			{"/", "Search"},
			{"l", "Full Log"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}
