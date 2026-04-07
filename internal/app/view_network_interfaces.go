package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderNetworkInterfaces() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredInterfaces()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Network \u203a Interfaces"
	s := m.renderHeader(breadcrumb, m.interfaceCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.interfaces == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No interfaces found.", iw, normalRowStyle) + "\n"
	} else {
		// compute column widths
		nameCol := len("INTERFACE")
		stateCol := len("STATE")
		ipCol := len("IP ADDRESS")
		mtuCol := len("MTU")
		for _, iface := range filtered {
			if len(iface.Name) > nameCol {
				nameCol = len(iface.Name)
			}
			if len(iface.State) > stateCol {
				stateCol = len(iface.State)
			}
			if len(iface.IPs) > ipCol {
				ipCol = len(iface.IPs)
			}
			if len(iface.MTU) > mtuCol {
				mtuCol = len(iface.MTU)
			}
		}
		nameCol += 2
		stateCol += 2
		ipCol += 2
		mtuCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s", nameCol, "INTERFACE", stateCol, "STATE", ipCol, "IP ADDRESS", mtuCol, "MTU")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		stateStyleUp := lipgloss.NewStyle().Foreground(colorGreen)
		stateStyleDown := lipgloss.NewStyle().Foreground(colorRed)
		stateStyleUnknown := lipgloss.NewStyle().Foreground(colorYellow)

		maxVisible := m.height - 8
		if m.filterText != "" {
			maxVisible -= 2
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.interfaceCursor >= offset+maxVisible {
			offset = m.interfaceCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			iface := filtered[i]

			cur := "   "
			if i == m.interfaceCursor {
				cur = " \u25b8 "
			}

			// color the state text
			var stateStr string
			switch iface.State {
			case "UP":
				stateStr = stateStyleUp.Render(fmt.Sprintf("%-*s", stateCol, iface.State))
			case "DOWN":
				stateStr = stateStyleDown.Render(fmt.Sprintf("%-*s", stateCol, iface.State))
			default:
				stateStr = stateStyleUnknown.Render(fmt.Sprintf("%-*s", stateCol, iface.State))
			}

			line := fmt.Sprintf("%s  %-*s  %s  %-*s  %-*s", cur, nameCol, iface.Name, stateStr, ipCol, iface.IPs, mtuCol, iface.MTU)

			var style lipgloss.Style
			if i == m.interfaceCursor {
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
		{"\u2191\u2193", "Navigate"},
		{"/", "Search"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
