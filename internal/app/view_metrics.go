package app

import (
	"fmt"
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderMetrics() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	// use sorted index if available, otherwise natural order
	hostIdx := m.metricsSortedIdx
	if len(hostIdx) != len(m.hosts) {
		hostIdx = make([]int, len(m.hosts))
		for i := range hostIdx {
			hostIdx[i] = i
		}
	}

	breadcrumb := f.Name + " \u203a Metrics"
	s := m.renderHeader(breadcrumb, m.metricsCursor+1, len(hostIdx)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	// compute host name column width
	hostCol := len("HOST")
	for _, h := range m.hosts {
		if len(h.Entry.Name) > hostCol {
			hostCol = len(h.Entry.Name)
		}
	}
	hostCol += 2

	hdr := fmt.Sprintf("     %-*s  %6s  %6s  %6s  %6s  %s",
		hostCol, "HOST"+m.sortIndicator(1),
		"CPU%"+m.sortIndicator(2),
		"MEM%"+m.sortIndicator(3),
		"DISK%"+m.sortIndicator(4),
		"LOAD"+m.sortIndicator(5),
		"UPTIME")
	s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

	// build group start map (only used when not sorting)
	sorted := m.sortColumn != 0
	groupStarts := make(map[int]string)
	if !sorted {
		lastGroup := ""
		for i, h := range m.hosts {
			if h.Group != "" && h.Group != lastGroup {
				groupStarts[i] = h.Group
				lastGroup = h.Group
			}
		}
	}

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}
	offset := 0
	if m.metricsCursor >= offset+maxVisible {
		offset = m.metricsCursor - maxVisible + 1
	}
	end := offset + maxVisible
	if end > len(hostIdx) {
		end = len(hostIdx)
	}

	warnStyle := lipgloss.NewStyle().Foreground(colorYellow)
	critStyle := lipgloss.NewStyle().Foreground(colorRed)

	for pos := offset; pos < end; pos++ {
		i := hostIdx[pos]

		// group header (only when not sorting by a column)
		if group, ok := groupStarts[i]; ok {
			groupLine := fmt.Sprintf("  \u2500\u2500 %s \u2500\u2500", group)
			s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
		}

		h := m.hosts[i]
		cur := "   "
		if pos == m.metricsCursor {
			cur = " \u25b8 "
		}

		if h.Status != config.HostOnline {
			line := fmt.Sprintf("%s  %-*s  %6s  %6s  %6s  %6s  %s", cur, hostCol, h.Entry.Name, "\u2014", "\u2014", "\u2014", "\u2014", "\u2014")
			var style lipgloss.Style
			if pos == m.metricsCursor {
				style = selectedRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
			continue
		}

		if m.metricErrors[i] {
			line := fmt.Sprintf("%s  %-*s  %6s  %6s  %6s  %6s  %s", cur, hostCol, h.Entry.Name, "err", "err", "err", "err", "err")
			var style lipgloss.Style
			if pos == m.metricsCursor {
				style = selectedRowStyle
			} else {
				style = critStyle
			}
			s += borderedRow(line, iw, style) + "\n"
			continue
		}

		met, ok := m.metrics[i]
		if !ok {
			// still loading
			line := fmt.Sprintf("%s  %-*s  %6s  %6s  %6s  %6s  %s", cur, hostCol, h.Entry.Name, "...", "...", "...", "...", "...")
			var style lipgloss.Style
			if pos == m.metricsCursor {
				style = selectedRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
			continue
		}

		// format with color
		cpuStr := fmt.Sprintf("%d", met.CPUPercent)
		memStr := fmt.Sprintf("%d", met.MemPercent)
		diskStr := fmt.Sprintf("%d", met.DiskPercent)

		line := fmt.Sprintf("%s  %-*s  %6s  %6s  %6s  %6s  %s",
			cur, hostCol, h.Entry.Name,
			cpuStr, memStr, diskStr, met.Load, met.Uptime)

		var style lipgloss.Style
		if pos == m.metricsCursor {
			style = selectedRowStyle
		} else if met.CPUPercent >= 90 || met.MemPercent >= 90 || met.DiskPercent >= 90 {
			style = critStyle
		} else if met.CPUPercent >= 80 || met.MemPercent >= 80 || met.DiskPercent >= 80 {
			style = warnStyle
		} else if pos%2 == 0 {
			style = altRowStyle
		} else {
			style = normalRowStyle
		}
		s += borderedRow(line, iw, style) + "\n"
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"\u2191\u2193", "Navigate"},
		{"1-5", "Sort"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	}))
	return s
}
