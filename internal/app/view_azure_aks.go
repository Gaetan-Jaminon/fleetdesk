package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func aksStatusStyle(state string) lipgloss.Style {
	switch strings.ToLower(state) {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	default:
		return lipgloss.NewStyle()
	}
}

func (m Model) renderAzureAKSList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredAzureAKS()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " › " + m.azureSubs[m.selectedAzureSub].Name + " › AKS Clusters"
	s := m.renderHeader(breadcrumb+filterInfo, m.azureAKSCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if m.azureAKSClusters == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No AKS clusters in subscription.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("NAME")
		rgCol := len("RESOURCE GROUP")
		versionCol := len("K8S VERSION")
		statusCol := 10
		nodesCol := 7

		for _, c := range filtered {
			if len(c.Name) > nameCol {
				nameCol = len(c.Name)
			}
			if len(c.ResourceGroup) > rgCol {
				rgCol = len(c.ResourceGroup)
			}
			if len(c.KubernetesVersion) > versionCol {
				versionCol = len(c.KubernetesVersion)
			}
		}
		nameCol += 2
		rgCol += 2
		versionCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %-*s  %s",
			nameCol, "NAME"+m.sortIndicator(1),
			rgCol, "RESOURCE GROUP"+m.sortIndicator(2),
			statusCol, "STATUS"+m.sortIndicator(3),
			versionCol, "K8S VERSION"+m.sortIndicator(4),
			nodesCol, "NODES"+m.sortIndicator(5),
			"POOLS"+m.sortIndicator(6),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.azureAKSCursor >= offset+maxVisible {
			offset = m.azureAKSCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			c := filtered[i]
			cur := "   "
			if i == m.azureAKSCursor {
				cur = " ▸ "
			}

			status := c.PowerState
			nodes := fmt.Sprintf("%d", c.NodeCount)
			pools := fmt.Sprintf("%d", len(c.Pools))

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
				cur, nameCol, c.Name, rgCol, c.ResourceGroup,
				statusCol, status, versionCol, c.KubernetesVersion,
				nodesCol, nodes, pools)

			var style lipgloss.Style
			if i == m.azureAKSCursor {
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

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"↑↓", "Navigate"},
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

func (m Model) renderAzureAKSDetail() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	clusterName := m.azureAKSDetail.Name
	if clusterName == "" {
		clusterName = "unknown"
	}
	breadcrumb := f.Name + " › " + m.azureSubs[m.selectedAzureSub].Name + " › AKS Clusters › " + clusterName
	s := m.renderHeader(breadcrumb, 0, 0) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	d := m.azureAKSDetail

	items := []struct{ key, val string }{
		{"Name", d.Name},
		{"Resource Group", d.ResourceGroup},
		{"Location", d.Location},
		{"Status", d.PowerState},
		{"K8s Version", d.KubernetesVersion},
		{"Network Plugin", d.NetworkPlugin},
		{"Total Nodes", fmt.Sprintf("%d", d.NodeCount)},
	}

	keyWidth := 0
	for _, item := range items {
		if len(item.key) > keyWidth {
			keyWidth = len(item.key)
		}
	}

	for i, item := range items {
		val := item.val
		if val == "" {
			val = "—"
		}
		if item.key == "Status" {
			val = aksStatusStyle(d.PowerState).Render(val)
		}
		line := fmt.Sprintf("    %-*s  %s", keyWidth, item.key, val)
		var style lipgloss.Style
		if i%2 == 0 {
			style = altRowStyle
		} else {
			style = normalRowStyle
		}
		s += borderedRow(line, iw, style) + "\n"
	}

	// Node pools section
	s += borderedRow("", iw, normalRowStyle) + "\n"
	s += borderedRow("  ── Node Pools ──", iw, colHeaderStyle) + "\n"
	s += borderedRow("", iw, normalRowStyle) + "\n"

	if len(d.Pools) == 0 {
		s += borderedRow("  No node pools.", iw, normalRowStyle) + "\n"
	} else {
		poolCol := len("POOL")
		modeCol := len("MODE")
		sizeCol := len("VM SIZE")
		nodesCol := 7
		minCol := 5
		maxCol := 5
		verCol := len("VERSION")

		for _, p := range d.Pools {
			if len(p.Name) > poolCol {
				poolCol = len(p.Name)
			}
			if len(p.Mode) > modeCol {
				modeCol = len(p.Mode)
			}
			if len(p.VMSize) > sizeCol {
				sizeCol = len(p.VMSize)
			}
			if len(p.Version) > verCol {
				verCol = len(p.Version)
			}
		}
		poolCol += 2
		modeCol += 2
		sizeCol += 2
		verCol += 2

		poolHdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
			poolCol, "POOL", modeCol, "MODE", sizeCol, "VM SIZE",
			nodesCol, "NODES", minCol, "MIN", maxCol, "MAX",
			verCol, "VERSION", "AUTOSCALE")
		s += borderedRow(poolHdr, iw, colHeaderStyle) + "\n"

		for i, p := range d.Pools {
			autoScale := "no"
			if p.AutoScale {
				autoScale = "yes"
			}
			poolLine := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*d  %-*d  %-*d  %-*s  %s",
				poolCol, p.Name, modeCol, p.Mode, sizeCol, p.VMSize,
				nodesCol, p.Count, minCol, p.MinCount, maxCol, p.MaxCount,
				verCol, p.Version, autoScale)
			var style lipgloss.Style
			if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(poolLine, iw, style) + "\n"
		}
	}

	// Activity log section
	s += borderedRow("", iw, normalRowStyle) + "\n"
	s += borderedRow("  ── Recent Activity (Resource Group) ──", iw, colHeaderStyle) + "\n"
	s += borderedRow("", iw, normalRowStyle) + "\n"

	if m.azureActivityLog == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(m.azureActivityLog) == 0 {
		s += borderedRow("  No recent activity.", iw, normalRowStyle) + "\n"
	} else {
		timeCol := len("TIME")
		opCol := len("OPERATION")
		resCol := len("RESOURCE")
		statusCol := len("STATUS")
		for _, e := range m.azureActivityLog {
			if len(e.Timestamp) > timeCol {
				timeCol = len(e.Timestamp)
			}
			if len(e.Operation) > opCol {
				opCol = len(e.Operation)
			}
			if len(e.Resource) > resCol {
				resCol = len(e.Resource)
			}
			if len(e.Status) > statusCol {
				statusCol = len(e.Status)
			}
		}
		timeCol += 2
		opCol += 2
		resCol += 2
		statusCol += 2

		logHdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %s",
			timeCol, "TIME", opCol, "OPERATION", resCol, "RESOURCE", statusCol, "STATUS", "CALLER")
		s += borderedRow(logHdr, iw, colHeaderStyle) + "\n"

		maxVisible := m.height - 24
		if maxVisible < 3 {
			maxVisible = 3
		}
		offset := 0
		if m.azureActivityCursor >= offset+maxVisible {
			offset = m.azureActivityCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.azureActivityLog) {
			end = len(m.azureActivityLog)
		}

		for i := offset; i < end; i++ {
			e := m.azureActivityLog[i]
			cur := "  "
			if i == m.azureActivityCursor {
				cur = " ▸"
			}
			logLine := fmt.Sprintf("%s   %-*s  %-*s  %-*s  %-*s  %s",
				cur, timeCol, e.Timestamp, opCol, e.Operation, resCol, e.Resource, statusCol, e.Status, e.Caller)
			var style lipgloss.Style
			if i == m.azureActivityCursor {
				style = selectedRowStyle
			} else if strings.Contains(strings.ToLower(e.Status), "fail") {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(logLine, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	s += m.renderHintBar([][]string{
		{"↑↓", "Scroll"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})
	return s
}
