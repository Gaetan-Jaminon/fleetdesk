package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderNetworkPorts() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredPorts()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Network \u203a Ports"
	s := m.renderHeader(breadcrumb, m.portCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.ports == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No listening ports found.", iw, normalRowStyle) + "\n"
	} else {
		// compute column widths
		portCol := len("PORT")
		protoCol := len("PROTOCOL")
		procCol := len("PROCESS")
		bindCol := len("BIND ADDRESS")
		for _, p := range filtered {
			ps := fmt.Sprintf("%d", p.Port)
			if len(ps) > portCol {
				portCol = len(ps)
			}
			if len(p.Protocol) > protoCol {
				protoCol = len(p.Protocol)
			}
			if len(p.Process) > procCol {
				procCol = len(p.Process)
			}
			if len(p.BindAddress) > bindCol {
				bindCol = len(p.BindAddress)
			}
		}
		portCol += 2
		protoCol += 2
		procCol += 2
		bindCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s", portCol, "PORT"+m.sortIndicator(1), protoCol, "PROTOCOL"+m.sortIndicator(2), procCol, "PROCESS"+m.sortIndicator(3), bindCol, "BIND ADDRESS"+m.sortIndicator(4))
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		dimmedStyle := lipgloss.NewStyle().Foreground(colorDimmed)

		maxVisible := m.height - 8
		if m.filterText != "" {
			maxVisible -= 2
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.portCursor >= offset+maxVisible {
			offset = m.portCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			p := filtered[i]

			cur := "   "
			if i == m.portCursor {
				cur = " \u25b8 "
			}

			portStr := fmt.Sprintf("%d", p.Port)
			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s", cur, portCol, portStr, protoCol, p.Protocol, procCol, p.Process, bindCol, p.BindAddress)

			isLocal := p.BindAddress == "127.0.0.1" || p.BindAddress == "::1"

			var style lipgloss.Style
			if i == m.portCursor {
				style = selectedRowStyle
			} else if isLocal {
				style = dimmedStyle
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
