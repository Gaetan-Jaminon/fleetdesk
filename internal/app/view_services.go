package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderServiceList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredServices()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Services"
	s := m.renderHeader(breadcrumb+filterInfo, m.serviceCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.services == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No services found.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("SERVICE")
		enabledCol := len("ENABLED")
		for _, svc := range filtered {
			if len(svc.Name) > nameCol {
				nameCol = len(svc.Name)
			}
			if len(svc.Enabled) > enabledCol {
				enabledCol = len(svc.Enabled)
			}
		}
		nameCol += 2
		enabledCol += 2

		hdr := fmt.Sprintf("     %-*s  %-10s  %-*s  %s", nameCol, "SERVICE", "STATE", enabledCol, "ENABLED", "DESCRIPTION")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.serviceCursor >= offset+maxVisible {
			offset = m.serviceCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			svc := filtered[i]
			cur := "   "
			if i == m.serviceCursor {
				cur = " \u25b8 "
			}
			prefix := ""
			if svc.State == "failed" {
				prefix = "\u2717 "
			}
			desc := svc.Description
			if desc == "" {
				desc = "\u2014"
			}
			line := fmt.Sprintf("%s  %s%-*s  %-10s  %-*s  %s", cur, prefix, nameCol, svc.Name, svc.State, enabledCol, svc.Enabled, desc)

			var style lipgloss.Style
			if i == m.serviceCursor {
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

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s\u2588", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"/", "Search"},
			{"s", "Start"},
			{"o", "Stop"},
			{"t", "Restart"},
			{"l", "Logs"},
			{"i", "Status"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}
