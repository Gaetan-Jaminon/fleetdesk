package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderNetworkRoutes() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredRoutes()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Network \u203a Routes & DNS"
	s := m.renderHeader(breadcrumb, m.routeCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	// DNS header
	dnsLine := "  DNS: "
	if len(m.dnsNameservers) > 0 {
		dnsLine += strings.Join(m.dnsNameservers, ", ")
	} else {
		dnsLine += "—"
	}
	if m.dnsSearch != "" {
		dnsLine += "  Search: " + m.dnsSearch
	}
	s += borderedRow(dnsLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
	s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

	if m.routes == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No routes found.", iw, normalRowStyle) + "\n"
	} else {
		// compute column widths
		destCol := len("DESTINATION")
		gwCol := len("GATEWAY")
		ifCol := len("INTERFACE")
		metCol := len("METRIC")
		for _, r := range filtered {
			if len(r.Destination) > destCol {
				destCol = len(r.Destination)
			}
			if len(r.Gateway) > gwCol {
				gwCol = len(r.Gateway)
			}
			if len(r.Interface) > ifCol {
				ifCol = len(r.Interface)
			}
			if len(r.Metric) > metCol {
				metCol = len(r.Metric)
			}
		}
		destCol += 2
		gwCol += 2
		ifCol += 2
		metCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s", destCol, "DESTINATION", gwCol, "GATEWAY", ifCol, "INTERFACE", metCol, "METRIC")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		boldStyle := lipgloss.NewStyle().Foreground(colorWhite).Bold(true)

		maxVisible := m.height - 10 // account for DNS header + table header
		if m.filterText != "" {
			maxVisible -= 2
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.routeCursor >= offset+maxVisible {
			offset = m.routeCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			r := filtered[i]

			cur := "   "
			if i == m.routeCursor {
				cur = " \u25b8 "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s", cur, destCol, r.Destination, gwCol, r.Gateway, ifCol, r.Interface, metCol, r.Metric)

			var style lipgloss.Style
			if i == m.routeCursor {
				style = selectedRowStyle
			} else if r.IsDefault {
				style = boldStyle
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
