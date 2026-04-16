package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

func (m Model) renderProcessList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredProcesses()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Processes"
	cur := 0
	if len(filtered) > 0 {
		cur = m.processCursor + 1
	}
	s := m.renderHeader(breadcrumb+filterInfo, cur, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(filtered) == 0 {
		msg := "  No processes"
		if m.filterText != "" {
			msg += " matching filter"
		}
		s += borderedRow(msg, iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("PROCESS")
		stateCol := len("STATE")
		uptimeCol := len("UPTIME")
		pidCol := len("PID")
		for _, p := range filtered {
			if len(p.Name) > nameCol {
				nameCol = len(p.Name)
			}
			if len(p.State) > stateCol {
				stateCol = len(p.State)
			}
		}
		nameCol += 2
		stateCol += 2
		uptimeCol += 2
		pidCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %s",
			nameCol, "PROCESS"+m.sortIndicator(1),
			stateCol, "STATE"+m.sortIndicator(2),
			uptimeCol, "UPTIME"+m.sortIndicator(3),
			"PID"+m.sortIndicator(4),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.processCursor >= offset+maxVisible {
			offset = m.processCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			p := filtered[i]
			cursor := "   "
			if i == m.processCursor {
				cursor = " ▸ "
			}

			// State prefix for failures
			nameDisplay := p.Name
			if p.State == "FATAL" || p.State == "BACKOFF" {
				nameDisplay = "✗ " + nameDisplay
			}

			// Color state
			stateStr := processStateColor(p.State)

			line := fmt.Sprintf("%s  %-*s  %s  %-*s  %-*s",
				cursor,
				nameCol, nameDisplay,
				stateStr,
				uptimeCol, p.Uptime,
				pidCol, p.PID,
			)

			var style lipgloss.Style
			if i == m.processCursor {
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
			{"s", "Start"},
			{"o", "Stop"},
			{"t", "Restart"},
			{"l", "Logs"},
			{"/", "Filter"},
			{"1-4", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		}))
	}
	return s
}

func processStateColor(state string) string {
	// Pre-pad to fixed width for alignment (ANSI codes break %-Ns)
	padded := fmt.Sprintf("%-10s", state)
	switch state {
	case "RUNNING":
		return ansiColor(padded, "32") // green
	case "FATAL", "BACKOFF":
		return ansiColor(padded, "31") // red
	case "STOPPED", "EXITED":
		return ansiColor(padded, "33") // yellow
	case "STARTING", "STOPPING":
		return ansiColor(padded, "36") // cyan
	default:
		return ansiColor(padded, "90") // dim
	}
}

func (m Model) filteredProcesses() []config.Process {
	if m.filterText == "" {
		return m.processes
	}
	f := strings.ToLower(m.filterText)
	var result []config.Process
	for _, p := range m.processes {
		if strings.Contains(strings.ToLower(p.Name), f) ||
			strings.Contains(strings.ToLower(p.State), f) {
			result = append(result, p)
		}
	}
	return result
}

func (m *Model) sortProcesses() {
	sort.Slice(m.processes, func(i, j int) bool {
		if m.sortColumn == 0 {
			oi := ssh.ProcessStateOrder(m.processes[i].State)
			oj := ssh.ProcessStateOrder(m.processes[j].State)
			if oi != oj {
				return oi < oj
			}
			return m.processes[i].Name < m.processes[j].Name
		}
		var less bool
		switch m.sortColumn {
		case 1:
			less = strings.ToLower(m.processes[i].Name) < strings.ToLower(m.processes[j].Name)
		case 2:
			less = ssh.ProcessStateOrder(m.processes[i].State) < ssh.ProcessStateOrder(m.processes[j].State)
		case 3:
			less = m.processes[i].Uptime < m.processes[j].Uptime
		case 4:
			less = m.processes[i].PID < m.processes[j].PID
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}
