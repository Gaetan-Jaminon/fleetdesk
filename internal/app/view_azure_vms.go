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

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}

	emptyMsg := "  No VMs in subscription."
	if m.filterText != "" {
		emptyMsg = fmt.Sprintf("  No matches for '%s'", m.filterText)
	}

	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "NAME", SortIndex: 1},
			{Label: "RESOURCE GROUP", SortIndex: 2},
			{Label: "STATUS", SortIndex: 3},
			{Label: "SIZE", SortIndex: 4},
			{Label: "OS", SortIndex: 5},
			{Label: "PRIVATE IP", SortIndex: 6},
			{Label: "HOSTNAME", SortIndex: 7},
		},
		RowCount: len(filtered),
		RowBuilder: func(i int) []string {
			vm := filtered[i]
			status := vm.PowerState
			if t, ok := m.transitions["vm/"+vm.Name]; ok {
				status = t.Display
			}
			return []string{vm.Name, vm.ResourceGroup, status, vm.VMSize, vm.OSType, vm.PrivateIP, vm.Hostname}
		},
		Cursor:        m.azureVMCursor,
		MaxVisible:    maxVisible,
		InnerWidth:    iw,
		SortIndicator: m.sortIndicator,
		EmptyMessage:  emptyMsg,
	})

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		s += m.renderHintBar(hintWithHelp([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Detail"},
			{"s", "Start"},
			{"o", "Stop"},
			{"t", "Restart"},
			{"/", "Filter"},
			{"1-7", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		}))
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
	s += m.renderActivityLog(iw)

	// Apply scrollable viewport
	contentLines := strings.Split(s, "\n")
	// headerLines = 2 (breadcrumb + top border), footerLines = 2 (bottom border + hint bar)
	maxVisible := m.height - 4
	if maxVisible < 5 {
		maxVisible = 5
	}
	totalLines := len(contentLines)
	maxScroll := totalLines - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	scrollOffset := m.azureVMDetailScroll
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	endLine := scrollOffset + maxVisible
	if endLine > totalLines {
		endLine = totalLines
	}
	s = strings.Join(contentLines[scrollOffset:endLine], "\n") + "\n"

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	s += m.renderHintBar(hintWithHelp([][]string{
		{"↑↓", "Scroll"},
		{"a", "Activity Log"},
		{"r", "Refresh"},
		{"Esc", "Back"},
		{"q", "Quit"},
	}))
	return s
}
