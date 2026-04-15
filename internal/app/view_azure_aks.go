package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatAKSDate extracts the date portion from an ISO timestamp (e.g. "2026-03-17T14:59:29Z" → "2026-03-17").
func formatAKSDate(s string) string {
	if s == "" {
		return "-"
	}
	if idx := strings.IndexByte(s, 'T'); idx > 0 {
		return s[:idx]
	}
	return s
}

// ansiColor wraps text with ANSI foreground color code only (no background reset).
// This preserves the row background applied by borderedRow.
func ansiColor(text string, code string) string {
	if code == "" {
		return text
	}
	return fmt.Sprintf("\033[%sm%s\033[39m", code, text)
}

func aksStatusColorCode(state string) string {
	switch strings.ToLower(state) {
	case "running":
		return "32" // green
	case "stopped":
		return "31" // red
	default:
		return ""
	}
}

func aksProvisioningColorCode(state string) string {
	switch strings.ToLower(state) {
	case "succeeded":
		return "32" // green
	case "starting", "stopping":
		return "33" // yellow
	case "failed":
		return "31" // red
	default:
		return ""
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
	displayTags := m.fleets[m.selectedFleet].DisplayTags
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " › " + m.azureSubs[m.selectedAzureSub].Name + " › AKS Clusters"
	s := m.renderHeader(breadcrumb+filterInfo, m.azureAKSCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(filtered) == 0 {
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
		provCol := len("PROVISIONING")
		createdCol := len("CREATED")
		nodesCol := 7

		// Compute tag column widths
		tagColWidths := make([]int, len(displayTags))
		for ti, tag := range displayTags {
			tagColWidths[ti] = len(strings.ToUpper(tag))
		}

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
			if len(c.ProvisioningState) > provCol {
				provCol = len(c.ProvisioningState)
			}
			if d := formatAKSDate(c.CreatedDate); len(d) > createdCol {
				createdCol = len(d)
			}
			for ti, tag := range displayTags {
				if c.Tags != nil {
					if v, ok := c.Tags[tag]; ok && len(v) > tagColWidths[ti] {
						tagColWidths[ti] = len(v)
					}
				}
			}
		}
		// Account for transition overlay display strings
		for k, t := range m.transitions {
			if strings.HasPrefix(k, "aks/") && len(t.Display) > provCol {
				provCol = len(t.Display)
			}
		}
		nameCol += 2
		rgCol += 2
		versionCol += 2
		provCol += 2
		createdCol += 2
		for ti := range tagColWidths {
			tagColWidths[ti] += 2
		}

		poolsCol := 7
		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
			nameCol, "NAME"+m.sortIndicator(1),
			rgCol, "RESOURCE GROUP"+m.sortIndicator(2),
			statusCol, "STATUS"+m.sortIndicator(3),
			provCol, "PROVISIONING"+m.sortIndicator(4),
			versionCol, "K8S VERSION"+m.sortIndicator(5),
			nodesCol, "NODES"+m.sortIndicator(6),
			poolsCol, "POOLS"+m.sortIndicator(7),
			createdCol, "CREATED"+m.sortIndicator(8),
		)
		for ti, tag := range displayTags {
			colNum := 9 + ti
			header := strings.ToUpper(tag) + m.sortIndicator(colNum)
			if ti == len(displayTags)-1 {
				hdr += fmt.Sprintf("  %s", header)
			} else {
				hdr += fmt.Sprintf("  %-*s", tagColWidths[ti], header)
			}
		}
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
			prov := c.ProvisioningState
			if t, ok := m.transitions["aks/"+c.Name]; ok {
				prov = t.Display
			}
			// Pad before coloring so ANSI codes don't affect width
			statusPadded := fmt.Sprintf("%-*s", statusCol, status)
			provPadded := fmt.Sprintf("%-*s", provCol, prov)
			statusColored := ansiColor(statusPadded, aksStatusColorCode(status))
			provColored := ansiColor(provPadded, aksProvisioningColorCode(prov))
			created := formatAKSDate(c.CreatedDate)
			nodes := fmt.Sprintf("%d", c.NodeCount)
			pools := fmt.Sprintf("%d", len(c.Pools))

			line := fmt.Sprintf("%s  %-*s  %-*s  %s  %s  %-*s  %-*s  %-*s  %-*s",
				cur, nameCol, c.Name, rgCol, c.ResourceGroup,
				statusColored, provColored,
				versionCol, c.KubernetesVersion, nodesCol, nodes,
				poolsCol, pools, createdCol, created)

			for ti, tag := range displayTags {
				val := "-"
				if c.Tags != nil {
					if v, ok := c.Tags[tag]; ok {
						val = v
					}
				}
				if ti == len(displayTags)-1 {
					line += fmt.Sprintf("  %s", val)
				} else {
					line += fmt.Sprintf("  %-*s", tagColWidths[ti], val)
				}
			}

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

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		maxSortCol := 8 + len(displayTags)
		sortLabel := fmt.Sprintf("1-%d", maxSortCol)
		s += m.renderHintBar(hintWithHelp([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Detail"},
			{"s", "Start"},
			{"o", "Stop"},
			{"d", "Delete"},
			{"/", "Filter"},
			{sortLabel, "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}))
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
		{"Provisioning", d.ProvisioningState},
		{"Created", formatAKSDate(d.CreatedDate)},
		{"K8s Version", d.KubernetesVersion},
		{"Network Plugin", d.NetworkPlugin},
		{"Total Nodes", fmt.Sprintf("%d", d.NodeCount)},
	}

	for _, tag := range m.fleets[m.selectedFleet].DisplayTags {
		val := "-"
		if d.Tags != nil {
			if v, ok := d.Tags[tag]; ok {
				val = v
			}
		}
		items = append(items, struct{ key, val string }{"Tag: " + tag, val})
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
			val = ansiColor(val, aksStatusColorCode(d.PowerState))
		}
		if item.key == "Provisioning" {
			val = ansiColor(val, aksProvisioningColorCode(d.ProvisioningState))
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
	s += m.renderActivityLog(iw)

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	s += m.renderHintBar(hintWithHelp([][]string{
		{"↑↓", "Scroll"},
		{"a", "Activity Log"},
		{"Esc", "Back"},
		{"q", "Quit"},
	}))
	return s
}
