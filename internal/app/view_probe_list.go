package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/probes"
)

func (m Model) renderProbeList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredProbeItems()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}

	cur := 0
	if len(filtered) > 0 {
		cur = m.probeCursor + 1
	}
	s := m.renderHeader(f.Name+filterInfo, cur, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(filtered) == 0 {
		msg := "  No probes"
		if m.filterText != "" {
			msg += " matching filter"
		}
		s += borderedRow(msg, iw, normalRowStyle) + "\n"
	} else {
		// Compute column widths
		nameCol := len("NAME")
		urlCol := len("URL")
		for _, p := range filtered {
			if len(p.Entry.Name) > nameCol {
				nameCol = len(p.Entry.Name)
			}
			if len(p.Entry.URL) > urlCol {
				urlCol = len(p.Entry.URL)
			}
		}
		nameCol += 2
		// Cap URL column to leave room for status/code/interval/latency
		maxURL := iw - nameCol - 58
		if maxURL < 10 {
			maxURL = 10
		}
		if urlCol > maxURL {
			urlCol = maxURL
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-8s  %-4s  %-10s  %-8s  %s",
			nameCol, "NAME"+m.sortIndicator(1),
			urlCol, "URL"+m.sortIndicator(2),
			"STATUS"+m.sortIndicator(3),
			"CODE"+m.sortIndicator(4),
			"TLS VERIFY",
			"INTERVAL"+m.sortIndicator(6),
			"LATENCY"+m.sortIndicator(5),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.probeCursor >= offset+maxVisible {
			offset = m.probeCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		// Build group start map
		groupStarts := make(map[int]string)
		lastGroup := ""
		for i, p := range filtered {
			if p.Group != "" && p.Group != lastGroup {
				groupStarts[i] = p.Group
				lastGroup = p.Group
			}
		}

		for i := offset; i < end; i++ {
			// Group header
			if groupName, ok := groupStarts[i]; ok {
				groupLine := fmt.Sprintf("  ── %s ──", groupName)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
			}

			p := filtered[i]
			cursor := "   "
			if i == m.probeCursor {
				cursor = " ▸ "
			}

			// Status display
			statusStr, statusColor := probeStatusDisplay(p.Result.Status)

			// Code
			codeStr := "---"
			if p.Result.Code > 0 {
				codeStr = fmt.Sprintf("%d", p.Result.Code)
			}

			// Latency
			latencyStr := "---"
			if p.Result.Status != probes.ProbeStatusPending && p.Result.Latency > 0 {
				latencyStr = formatLatency(p.Result.Latency)
			}

			// Interval (resolved: per-probe or fleet default)
			interval := p.Entry.Interval
			if interval == 0 {
				interval = m.fleets[m.selectedFleet].ProbeFleet.Defaults.Interval
			}
			intervalStr := fmt.Sprintf("%ds", int(interval.Seconds()))

			// Live indicator — green steady, green ring during probe, white pending
			live := ansiColor("●", "97") // white: pending
			if p.Result.Status != probes.ProbeStatusPending {
				if time.Since(p.Result.ProbeTime) < 2*time.Second {
					live = ansiColor("◉", "32") // green ring: just probed
				} else {
					live = ansiColor("●", "32") // green filled: steady
				}
			}

			// Truncate URL if needed
			urlStr := p.Entry.URL
			if len(urlStr) > urlCol {
				urlStr = urlStr[:urlCol-1] + "…"
			}

			// TLS verify indicator (pre-padded — ANSI codes break %-Ns)
			tlsStr := "✓         "
			if m.fleets[m.selectedFleet].ProbeFleet.Defaults.InsecureSkipVerify {
				tlsStr = ansiColor("Skipped", "33") + "   "
			}

			line := fmt.Sprintf("%s%s %-*s  %-*s  %s  %-4s  %s  %-8s  %s",
				cursor, live,
				nameCol, p.Entry.Name,
				urlCol, urlStr,
				statusColor(fmt.Sprintf("%-8s", statusStr)),
				codeStr,
				tlsStr,
				intervalStr,
				latencyStr,
			)

			var style lipgloss.Style
			if i == m.probeCursor {
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
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		s += m.renderHintBar(hintWithHelp([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Detail"},
			{"r", "Refresh"},
			{"/", "Filter"},
			{"1-6", "Sort"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}))
	}
	return s
}

func probeStatusDisplay(status probes.ProbeStatus) (string, func(string) string) {
	switch status {
	case probes.ProbeStatusUp:
		return "UP", func(s string) string { return "\033[32m" + s + "\033[0m" }
	case probes.ProbeStatusDown:
		return "DOWN", func(s string) string { return "\033[31m" + s + "\033[0m" }
	case probes.ProbeStatusDegraded:
		return "DEGRADED", func(s string) string { return "\033[33m" + s + "\033[0m" }
	default:
		return "---", func(s string) string { return "\033[90m" + s + "\033[0m" }
	}
}

func formatLatency(d interface{ Milliseconds() int64 }) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
