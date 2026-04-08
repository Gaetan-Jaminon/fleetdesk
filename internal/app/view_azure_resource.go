package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func hasAzureResourceError(errs []string, prefix string) bool {
	for _, e := range errs {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

func (m Model) renderAzureResourcePicker() string {
	f := m.fleets[m.selectedFleet]
	sub := m.azureSubs[m.selectedAzureSub]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + sub.Name
	s := m.renderHeader(breadcrumb, m.azureResourceCursor+1, 3) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	nameCol := len("Resource Groups") + 2

	hdr := fmt.Sprintf("     %-*s  %7s", nameCol, "RESOURCE", "TOTAL")
	s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

	resources := []struct {
		name  string
		count int
	}{
		{"VMs", m.azureResourceCounts.VMs},
		{"Resource Groups", m.azureResourceCounts.RGs},
		{"AKS Clusters", m.azureResourceCounts.AKS},
	}

	errorPrefixes := []string{"vms:", "rgs:", "aks:"}

	for i, r := range resources {
		cur := "   "
		if i == m.azureResourceCursor {
			cur = " ▸ "
		}

		countStr := "..."
		if m.azureCountsLoaded {
			if hasAzureResourceError(m.azureResourceErrors, errorPrefixes[i]) {
				countStr = "err"
			} else {
				countStr = fmt.Sprintf("%d", r.count)
			}
		}

		line := fmt.Sprintf("%s  %-*s  %7s", cur, nameCol, r.name, countStr)

		var style lipgloss.Style
		if i == m.azureResourceCursor {
			style = selectedRowStyle
		} else if i%2 == 0 {
			style = altRowStyle
		} else {
			style = normalRowStyle
		}
		s += borderedRow(line, iw, style) + "\n"
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar([][]string{
		{"↑↓", "Navigate"},
		{"Enter", "Select"},
		{"r", "Refresh"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})
	return s
}
