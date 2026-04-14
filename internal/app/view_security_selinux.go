package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSELinuxDenials() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredSELinuxDenials()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a SELinux Denials"
	s := m.renderHeader(breadcrumb, m.selinuxCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.selinuxDenials == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No SELinux denials found.", iw, normalRowStyle) + "\n"
	} else {
		timeCol := len("TIME") + 2
		actionCol := len("ACTION") + 2
		sourceCol := len("SOURCE") + 2
		targetCol := len("TARGET") + 2
		classCol := len("CLASS") + 2
		for _, d := range filtered {
			if len(d.Time)+2 > timeCol {
				timeCol = len(d.Time) + 2
			}
			if len(d.Action)+2 > actionCol {
				actionCol = len(d.Action) + 2
			}
			if len(d.Source)+2 > sourceCol {
				sourceCol = len(d.Source) + 2
			}
			if len(d.Target)+2 > targetCol {
				targetCol = len(d.Target) + 2
			}
			if len(d.Class)+2 > classCol {
				classCol = len(d.Class) + 2
			}
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %-*s", timeCol, "TIME"+m.sortIndicator(1), actionCol, "ACTION"+m.sortIndicator(2), sourceCol, "SOURCE"+m.sortIndicator(3), targetCol, "TARGET"+m.sortIndicator(4), classCol, "CLASS"+m.sortIndicator(5))
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
		if m.selinuxCursor >= offset+maxVisible {
			offset = m.selinuxCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			d := filtered[i]
			cur := "   "
			if i == m.selinuxCursor {
				cur = " \u25b8 "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s  %-*s", cur, timeCol, d.Time, actionCol, d.Action, sourceCol, d.Source, targetCol, d.Target, classCol, d.Class)

			var style lipgloss.Style
			if i == m.selinuxCursor {
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
	s += m.renderSudoPromptOrHintBar([][]string{
		{"\u2191\u2193", "Navigate"},
		{"1-5", "Sort"},
		{"/", "Search"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
