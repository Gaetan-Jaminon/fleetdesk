package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderK8sResourcePicker() string {
	f := m.fleets[m.selectedFleet]
	cluster := m.k8sClusters[m.selectedK8sCluster]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + cluster.Name + " › " + m.selectedK8sContext
	s := m.renderHeader(breadcrumb, m.k8sResourceCursor+1, 3) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	nameCol := len("Workloads (namespaces)") + 2

	hdr := fmt.Sprintf("     %-*s  %7s", nameCol, "RESOURCE", "TOTAL")
	s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

	resources := []struct {
		name  string
		count int
	}{
		{"Workloads (namespaces)", m.k8sResourceCounts.Namespaces},
		{"Nodes", m.k8sResourceCounts.Nodes},
		{"ArgoCD Apps", m.k8sResourceCounts.ArgoApps},
	}

	errorPrefixes := []string{"namespaces:", "nodes:", "argocd:"}

	for i, r := range resources {
		cur := "   "
		if i == m.k8sResourceCursor {
			cur = " ▸ "
		}

		countStr := "..."
		if m.k8sCountsLoaded {
			if hasK8sResourceError(m.k8sResourceErrors, errorPrefixes[i]) {
				countStr = "err"
			} else {
				countStr = fmt.Sprintf("%d", r.count)
			}
		}

		line := fmt.Sprintf("%s  %-*s  %7s", cur, nameCol, r.name, countStr)

		var style lipgloss.Style
		if i == m.k8sResourceCursor {
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

func hasK8sResourceError(errs []string, prefix string) bool {
	for _, e := range errs {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}
