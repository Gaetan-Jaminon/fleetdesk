package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func azureVMStatusStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	case "deallocated":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	default:
		return lipgloss.NewStyle()
	}
}

func (m Model) renderAzureVMList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredAzureVMs()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " › " + m.azureSubs[m.selectedAzureSub].Name + " › VMs"
	s := m.renderHeader(breadcrumb+filterInfo, m.azureVMCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if m.azureVMs == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No VMs in subscription.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("NAME")
		rgCol := len("RESOURCE GROUP")
		statusCol := len("STATUS")
		sizeCol := len("SIZE")
		osCol := len("OS")
		ipCol := len("PRIVATE IP")
		hostCol := len("HOSTNAME")
		for _, vm := range filtered {
			if len(vm.Name) > nameCol {
				nameCol = len(vm.Name)
			}
			if len(vm.ResourceGroup) > rgCol {
				rgCol = len(vm.ResourceGroup)
			}
			if len(vm.PowerState) > statusCol {
				statusCol = len(vm.PowerState)
			}
			if len(vm.VMSize) > sizeCol {
				sizeCol = len(vm.VMSize)
			}
			if len(vm.OSType) > osCol {
				osCol = len(vm.OSType)
			}
			if len(vm.PrivateIP) > ipCol {
				ipCol = len(vm.PrivateIP)
			}
			if len(vm.Hostname) > hostCol {
				hostCol = len(vm.Hostname)
			}
		}
		nameCol += 2
		rgCol += 2
		statusCol += 2
		sizeCol += 2
		osCol += 2
		ipCol += 2
		hostCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
			nameCol, "NAME"+m.sortIndicator(1),
			rgCol, "RESOURCE GROUP"+m.sortIndicator(2),
			statusCol, "STATUS"+m.sortIndicator(3),
			sizeCol, "SIZE"+m.sortIndicator(4),
			osCol, "OS"+m.sortIndicator(5),
			ipCol, "PRIVATE IP"+m.sortIndicator(6),
			"HOSTNAME"+m.sortIndicator(7),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.azureVMCursor >= offset+maxVisible {
			offset = m.azureVMCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			vm := filtered[i]
			cur := "   "
			if i == m.azureVMCursor {
				cur = " ▸ "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
				cur, nameCol, vm.Name, rgCol, vm.ResourceGroup,
				statusCol, vm.PowerState, sizeCol, vm.VMSize, osCol, vm.OSType,
				ipCol, vm.PrivateIP, vm.Hostname)

			var style lipgloss.Style
			if i == m.azureVMCursor {
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
			{"s", "Start"},
			{"o", "Stop"},
			{"t", "Restart"},
			{"/", "Filter"},
			{"1-7", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}

func (m Model) renderAzureVMDetail() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	vmName := m.azureVMDetail.Name
	if vmName == "" {
		vmName = "unknown"
	}
	breadcrumb := f.Name + " › " + m.azureSubs[m.selectedAzureSub].Name + " › VMs › " + vmName
	s := m.renderHeader(breadcrumb, 0, 0) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	d := m.azureVMDetail

	// Build tags display
	tagsDisplay := "—"
	if len(d.Tags) > 0 {
		var tagParts []string
		for k, v := range d.Tags {
			tagParts = append(tagParts, k+"="+v)
		}
		tagsDisplay = strings.Join(tagParts, ", ")
	}

	osDiskSize := "—"
	if d.OSDiskSizeGB > 0 {
		osDiskSize = fmt.Sprintf("%d GB", d.OSDiskSizeGB)
	}

	items := []struct{ key, val string }{
		{"Name", d.Name},
		{"Hostname", d.Hostname},
		{"Resource Group", d.ResourceGroup},
		{"Location", d.Location},
		{"Status", d.PowerState},
		{"Size", d.VMSize},
		{"OS Type", d.OSType},
		{"OS Disk", d.OSDisk},
		{"Private IP", d.PrivateIP},
		{"Public IP", d.PublicIP},
		{"VNet", d.VNet},
		{"Subnet", d.Subnet},
		{"OS Disk Name", d.OSDiskName},
		{"OS Disk Size", osDiskSize},
		{"NIC", d.NICName},
		{"Created", d.CreatedTime},
		{"Tags", tagsDisplay},
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
			val = azureVMStatusStyle(d.PowerState).Render(val)
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

	// Activity log section
	s += borderedRow("", iw, normalRowStyle) + "\n"
	s += borderedRow("  ── Recent Activity (Resource Group) ──", iw, colHeaderStyle) + "\n"
	s += borderedRow("", iw, normalRowStyle) + "\n"

	if m.azureActivityLog == nil {
		s += borderedRow("  Press 'a' to load activity log", iw, normalRowStyle) + "\n"
	} else if len(m.azureActivityLog) == 0 {
		s += borderedRow("  No recent activity.", iw, normalRowStyle) + "\n"
	} else {
		timeCol := len("TIME")
		opCol := len("OPERATION")
		statusCol := len("STATUS")
		for _, e := range m.azureActivityLog {
			if len(e.Timestamp) > timeCol {
				timeCol = len(e.Timestamp)
			}
			if len(e.Operation) > opCol {
				opCol = len(e.Operation)
			}
			if len(e.Status) > statusCol {
				statusCol = len(e.Status)
			}
		}
		timeCol += 2
		opCol += 2
		statusCol += 2

		logHdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %s",
			timeCol, "TIME", opCol, "OPERATION", statusCol, "STATUS", "CALLER")
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
			logLine := fmt.Sprintf("%s   %-*s  %-*s  %-*s  %s",
				cur, timeCol, e.Timestamp, opCol, e.Operation, statusCol, e.Status, e.Caller)
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
		{"a", "Activity Log"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})
	return s
}
