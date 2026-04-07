package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderNetworkPicker() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Network"
	s := m.renderHeader(breadcrumb, m.networkCursor+1, 4) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	nameCol := len("RESOURCE") + 4

	hdr := fmt.Sprintf("     %-*s  %7s  %s", nameCol, "RESOURCE", "COUNT", "STATUS")
	s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

	type netRow struct {
		name   string
		count  int
		status string
	}

	fwStatus := m.firewallType
	if fwStatus == "" {
		fwStatus = "…"
	}

	rows := []netRow{
		{"Interfaces", h.InterfacesTotal, fmt.Sprintf("%d UP", h.InterfacesUp)},
		{"Ports", h.ListeningPorts, "\u2014"},
		{"Routes", m.routeCount, "\u2014"},
		{"Firewall", m.firewallCount, fwStatus},
	}

	for i, r := range rows {
		cur := "   "
		if i == m.networkCursor {
			cur = " \u25b8 "
		}
		line := fmt.Sprintf("%s  %-*s  %7d  %s", cur, nameCol, r.name, r.count, r.status)

		var style lipgloss.Style
		if i == m.networkCursor {
			style = selectedRowStyle
		} else if i%2 == 0 {
			style = altRowStyle
		} else {
			style = normalRowStyle
		}
		s += borderedRow(line, iw, style) + "\n"
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar([][]string{
		{"\u2191\u2193", "Navigate"},
		{"Enter", "Select"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
