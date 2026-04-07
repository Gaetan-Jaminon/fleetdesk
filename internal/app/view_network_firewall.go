package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderNetworkFirewall() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredFirewallRules()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Network \u203a Firewall"
	s := m.renderHeader(breadcrumb, m.firewallCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	// backend info line
	var backendLine string
	switch m.firewallBackend {
	case "firewalld":
		backendLine = "  Firewall: firewalld (active)"
	case "nftables":
		backendLine = "  Firewall: nftables"
	case "iptables":
		backendLine = "  Firewall: iptables"
	default:
		if m.firewallRules != nil {
			backendLine = "  No firewall detected"
		}
	}
	if backendLine != "" {
		s += borderedRow(backendLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.filterText != "" {
		filterLine := fmt.Sprintf("  Filter: %s", m.filterText)
		s += borderedRow(filterLine, iw, lipgloss.NewStyle().Foreground(colorCyan)) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	}

	if m.firewallRules == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 && m.firewallBackend == "" {
		s += borderedRow("  No firewall detected", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		s += borderedRow("  No rules found", iw, normalRowStyle) + "\n"
	} else {
		// compute column widths
		zoneCol := len("ZONE")
		svcCol := len("SERVICE/PORT")
		protoCol := len("PROTOCOL")
		srcCol := len("SOURCE")
		actCol := len("ACTION")
		for _, r := range filtered {
			if len(r.Zone) > zoneCol {
				zoneCol = len(r.Zone)
			}
			if len(r.Service) > svcCol {
				svcCol = len(r.Service)
			}
			if len(r.Protocol) > protoCol {
				protoCol = len(r.Protocol)
			}
			if len(r.Source) > srcCol {
				srcCol = len(r.Source)
			}
			if len(r.Action) > actCol {
				actCol = len(r.Action)
			}
		}
		zoneCol += 2
		svcCol += 2
		protoCol += 2
		srcCol += 2
		actCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %-*s", zoneCol, "ZONE", svcCol, "SERVICE/PORT", protoCol, "PROTOCOL", srcCol, "SOURCE", actCol, "ACTION")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		dropStyle := lipgloss.NewStyle().Foreground(colorRed)

		maxVisible := m.height - 10
		if m.filterText != "" {
			maxVisible -= 2
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.firewallCursor >= offset+maxVisible {
			offset = m.firewallCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			r := filtered[i]

			cur := "   "
			if i == m.firewallCursor {
				cur = " \u25b8 "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s  %-*s", cur, zoneCol, r.Zone, svcCol, r.Service, protoCol, r.Protocol, srcCol, r.Source, actCol, r.Action)

			isDrop := strings.EqualFold(r.Action, "drop") || strings.EqualFold(r.Action, "reject")

			var style lipgloss.Style
			if i == m.firewallCursor {
				style = selectedRowStyle
			} else if isDrop {
				style = dropStyle
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
