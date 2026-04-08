package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func k8sPodStatusStyle(status string) lipgloss.Style {
	switch status {
	case "Running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	case "Succeeded":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	case "Pending":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	case "Failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	}
}

func (m Model) renderK8sPodList() string {
	f := m.fleets[m.selectedFleet]
	cluster := m.k8sClusters[m.selectedK8sCluster]
	ns := m.k8sNamespaces[m.selectedK8sNamespace]
	wl := m.k8sWorkloads[m.selectedK8sWorkload]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredK8sPodList()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " \u203a " + cluster.Name + " \u203a " + m.selectedK8sContext + " \u203a " + ns.Name + " \u203a " + wl.Name
	s := m.renderHeader(breadcrumb+filterInfo, m.k8sPodCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.k8sPodList == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No pods.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("NAME")
		statusCol := len("STATUS")
		readyCol := len("READY")
		restartsCol := len("RESTARTS")
		nodeCol := len("NODE")
		for _, p := range filtered {
			if len(p.Name) > nameCol {
				nameCol = len(p.Name)
			}
			if len(p.Status) > statusCol {
				statusCol = len(p.Status)
			}
			if len(p.Ready) > readyCol {
				readyCol = len(p.Ready)
			}
			if len(p.Node) > nodeCol {
				nodeCol = len(p.Node)
			}
		}
		nameCol += 2
		statusCol += 2
		readyCol += 2
		restartsCol += 2
		nodeCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %*s  %-*s  %s",
			nameCol, "NAME"+m.sortIndicator(1),
			statusCol, "STATUS"+m.sortIndicator(2),
			readyCol, "READY"+m.sortIndicator(3),
			restartsCol, "RESTARTS"+m.sortIndicator(4),
			nodeCol, "NODE"+m.sortIndicator(5),
			"AGE"+m.sortIndicator(6),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.k8sPodCursor >= offset+maxVisible {
			offset = m.k8sPodCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			p := filtered[i]
			cur := "   "
			if i == m.k8sPodCursor {
				cur = " \u25b8 "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %*d  %-*s  %s",
				cur, nameCol, p.Name,
				statusCol, p.Status,
				readyCol, p.Ready,
				restartsCol, p.Restarts,
				nodeCol, p.Node,
				p.Age,
			)

			var style lipgloss.Style
			if i == m.k8sPodCursor {
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

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s\u2588", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"\u2191\u2193", "Navigate"},
			{"Enter", "Detail"},
			{"/", "Filter"},
			{"1-6", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}

func (m Model) renderK8sPodDetail() string {
	f := m.fleets[m.selectedFleet]
	cluster := m.k8sClusters[m.selectedK8sCluster]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	d := m.k8sPodDetail

	podName := d.Name
	if podName == "" {
		podName = "unknown"
	}
	breadcrumb := f.Name + " \u203a " + cluster.Name + " \u203a " + m.selectedK8sContext + " \u203a " + d.Namespace + " \u203a " + podName
	s := m.renderHeader(breadcrumb, 0, 0) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	type kv struct{ key, val string }

	renderSection := func(title string, items []kv) string {
		var lines []string
		lines = append(lines, fmt.Sprintf("\u2500\u2500 %s \u2500\u2500", title))
		keyW := 0
		for _, item := range items {
			if len(item.key) > keyW {
				keyW = len(item.key)
			}
		}
		for _, item := range items {
			val := item.val
			if val == "" {
				val = "\u2014"
			}
			lines = append(lines, fmt.Sprintf("%-*s  %s", keyW, item.key, val))
		}
		return strings.Join(lines, "\n")
	}

	// Compute resource totals from containers
	cpuReq, cpuLim, memReq, memLim := "\u2014", "\u2014", "\u2014", "\u2014"
	if len(d.Containers) > 0 {
		// Use first container values for single-container pods, show totals for multi
		if len(d.Containers) == 1 {
			c := d.Containers[0]
			if c.CPUReq != "" {
				cpuReq = c.CPUReq
			}
			if c.CPULim != "" {
				cpuLim = c.CPULim
			}
			if c.MemReq != "" {
				memReq = c.MemReq
			}
			if c.MemLim != "" {
				memLim = c.MemLim
			}
		} else {
			// For multi-container, show "see containers"
			cpuReq = "see containers"
			cpuLim = "see containers"
			memReq = "see containers"
			memLim = "see containers"
		}
	}

	// Left column: Pod Info
	left := renderSection("Pod Info", []kv{
		{"Name", d.Name},
		{"Namespace", d.Namespace},
		{"Node", d.Node},
		{"IP", d.IP},
	})

	// Middle column: Status
	mid := renderSection("Status", []kv{
		{"Status", k8sPodStatusStyle(d.Status).Render(d.Status)},
		{"Ready", d.Ready},
		{"Restarts", fmt.Sprintf("%d", d.Restarts)},
		{"Age", d.Age},
	})

	// Right column: Resources
	right := renderSection("Resources", []kv{
		{"CPU Req", cpuReq},
		{"CPU Lim", cpuLim},
		{"Mem Req", memReq},
		{"Mem Lim", memLim},
	})

	// Three-column layout
	colW := iw / 3
	leftStyle := lipgloss.NewStyle().Width(colW)
	midStyle := lipgloss.NewStyle().Width(colW)
	rightStyle := lipgloss.NewStyle().Width(iw - 2*colW)

	threeCol := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(left),
		midStyle.Render(mid),
		rightStyle.Render(right),
	)

	for _, line := range strings.Split(threeCol, "\n") {
		s += borderedRow(line, iw, normalRowStyle) + "\n"
	}

	// Blank row before container table
	s += borderedRow("", iw, normalRowStyle) + "\n"

	// Container table section
	s += borderedRow(fmt.Sprintf("  \u2500\u2500 Containers (%d) \u2500\u2500", len(d.Containers)), iw, groupHeaderStyle) + "\n"

	if len(d.Containers) == 0 {
		s += borderedRow("  No containers.", iw, normalRowStyle) + "\n"
	} else {
		cNameCol := len("NAME")
		cImageCol := len("IMAGE")
		cStateCol := len("STATE")
		for _, c := range d.Containers {
			if len(c.Name) > cNameCol {
				cNameCol = len(c.Name)
			}
			if len(c.Image) > cImageCol {
				cImageCol = len(c.Image)
			}
			if len(c.State) > cStateCol {
				cStateCol = len(c.State)
			}
		}
		cNameCol += 2
		cImageCol += 2
		cStateCol += 2

		cHdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-7s  %-10s  %-10s  %-10s  %-10s  %s",
			cNameCol, "NAME"+m.sortIndicator(1),
			cImageCol, "IMAGE"+m.sortIndicator(2),
			cStateCol, "STATE"+m.sortIndicator(3),
			"READY"+m.sortIndicator(4),
			"RESTARTS"+m.sortIndicator(5),
			"CPU REQ"+m.sortIndicator(6),
			"CPU LIM"+m.sortIndicator(7),
			"MEM REQ"+m.sortIndicator(8),
			"MEM LIM"+m.sortIndicator(9),
		)
		s += borderedRow(cHdr, iw, colHeaderStyle) + "\n"

		linesUsed := strings.Count(s, "\n")
		maxVisible := m.height - linesUsed - 4
		if maxVisible < 3 {
			maxVisible = 3
		}
		offset := 0
		if m.k8sPodContainerCursor >= offset+maxVisible {
			offset = m.k8sPodContainerCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(d.Containers) {
			end = len(d.Containers)
		}

		for i := offset; i < end; i++ {
			c := d.Containers[i]
			cur := "  "
			if i == m.k8sPodContainerCursor {
				cur = " \u25b8"
			}

			readyStr := fmt.Sprintf("%v", c.Ready)
			cpuR := c.CPUReq
			if cpuR == "" {
				cpuR = "\u2014"
			}
			cpuL := c.CPULim
			if cpuL == "" {
				cpuL = "\u2014"
			}
			mR := c.MemReq
			if mR == "" {
				mR = "\u2014"
			}
			mL := c.MemLim
			if mL == "" {
				mL = "\u2014"
			}

			cLine := fmt.Sprintf("%s   %-*s  %-*s  %-*s  %-7s  %-10d  %-10s  %-10s  %-10s  %s",
				cur, cNameCol, c.Name, cImageCol, c.Image, cStateCol, c.State,
				readyStr, c.Restarts, cpuR, cpuL, mR, mL)

			var style lipgloss.Style
			if i == m.k8sPodContainerCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(cLine, iw, style) + "\n"
		}
	}

	// Blank row before conditions
	s += borderedRow("", iw, normalRowStyle) + "\n"

	// Conditions section
	s += borderedRow("  \u2500\u2500 Conditions \u2500\u2500", iw, groupHeaderStyle) + "\n"

	if len(d.Conditions) == 0 {
		s += borderedRow("  No conditions.", iw, normalRowStyle) + "\n"
	} else {
		typeW := 0
		for _, c := range d.Conditions {
			if len(c.Type) > typeW {
				typeW = len(c.Type)
			}
		}
		for i, c := range d.Conditions {
			condLine := fmt.Sprintf("     %-*s  %s", typeW, c.Type, c.Status)
			var style lipgloss.Style
			if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(condLine, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s\u2588", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"\u2191\u2193", "Scroll"},
			{"/", "Filter"},
			{"1-9", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}
