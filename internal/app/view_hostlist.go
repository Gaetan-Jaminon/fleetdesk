package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func (m Model) renderHostList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader(f.Name, m.hostCursor+1, len(m.hosts)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.hosts) == 0 {
		s += borderedRow("  No hosts in fleet.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("HOST")
		osCol := len("OS")
		for _, h := range m.hosts {
			if len(h.Entry.Name) > nameCol {
				nameCol = len(h.Entry.Name)
			}
			if len(h.OS) > osCol {
				osCol = len(h.OS)
			}
		}
		nameCol += 2
		osCol += 2

		// compute dynamic column widths from actual data
		upCol := len("UP SINCE")
		for _, h := range m.hosts {
			if len(h.UpSince) > upCol {
				upCol = len(h.UpSince)
			}
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %s", nameCol, "HOST", osCol, "OS", upCol, "UP SINCE", "UPD")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.hostCursor >= offset+maxVisible {
			offset = m.hostCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.hosts) {
			end = len(m.hosts)
		}

		// build group start index map
		groupStarts := make(map[int]string)
		for i, h := range m.hosts {
			if h.Group != "" {
				if i == 0 || m.hosts[i-1].Group != h.Group {
					groupStarts[i] = h.Group
				}
			}
		}

		for i := offset; i < end; i++ {
			// render group header if this host starts a new group
			if groupName, ok := groupStarts[i]; ok {
				groupLine := fmt.Sprintf("  \u2500\u2500 %s \u2500\u2500", groupName)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
			}

			h := m.hosts[i]
			cur := "   "
			if i == m.hostCursor {
				cur = " \u25b8 "
			}

			var line string
			switch h.Status {
			case config.HostConnecting:
				line = fmt.Sprintf("%s  %-*s  connecting...", cur, nameCol, h.Entry.Name)
			case config.HostUnreachable:
				reason := h.Error
				if reason == "" {
					reason = "unknown"
				}
				line = fmt.Sprintf("%s  %-*s  unreachable (%s)", cur, nameCol, h.Entry.Name, reason)
			default:
				updStr := fmt.Sprintf("%d", h.UpdateCount)
				if h.UpdateCount == 0 {
					updStr = "—"
				}
				line = fmt.Sprintf("%s  %-*s  %-*s  %-*s  %s",
					cur, nameCol, h.Entry.Name, osCol, h.OS, upCol, h.UpSince, updStr)
			}

			var style lipgloss.Style
			if i == m.hostCursor {
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

	s += m.renderHintBar(hintWithHelp([][]string{
		{"↑↓", "Navigate"},
		{"Enter", "Drill In"},
		{"x", "Shell"},
		{"K", "Deploy Key"},
		{"d", "Metrics"},
		{"R", "Reboot"},
		{"r", "Refresh"},
		{"Esc", "Back"},
		{"q", "Quit"},
	}))
	return s
}
