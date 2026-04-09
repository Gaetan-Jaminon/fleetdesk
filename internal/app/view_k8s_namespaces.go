package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderK8sNamespaceList() string {
	f := m.fleets[m.selectedFleet]
	cluster := m.k8sClusters[m.selectedK8sCluster]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredK8sNamespaces()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " \u203a " + cluster.Name + " \u203a " + m.selectedK8sContext + " \u203a Workloads"
	s := m.renderHeader(breadcrumb+filterInfo, m.k8sNamespaceCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.k8sNamespaces == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No namespaces.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("NAMESPACE")
		statusCol := len("STATUS")
		podsCol := 5
		deployCol := 6
		stsCol := 4
		dsCol := 3
		for _, ns := range filtered {
			if len(ns.Name) > nameCol {
				nameCol = len(ns.Name)
			}
			if len(ns.Status) > statusCol {
				statusCol = len(ns.Status)
			}
		}
		nameCol += 2
		statusCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %*s  %*s  %*s  %*s  %s",
			nameCol, "NAMESPACE"+m.sortIndicator(1),
			statusCol, "STATUS"+m.sortIndicator(2),
			podsCol, "PODS"+m.sortIndicator(3),
			deployCol, "DEPLOY"+m.sortIndicator(4),
			stsCol, "STS"+m.sortIndicator(5),
			dsCol, "DS"+m.sortIndicator(6),
			"AGE"+m.sortIndicator(7),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.k8sNamespaceCursor >= offset+maxVisible {
			offset = m.k8sNamespaceCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			ns := filtered[i]
			cur := "   "
			if i == m.k8sNamespaceCursor {
				cur = " \u25b8 "
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %*d  %*d  %*d  %*d  %s",
				cur, nameCol, ns.Name,
				statusCol, ns.Status,
				podsCol, ns.PodCount,
				deployCol, ns.DeployCount,
				stsCol, ns.STSCount,
				dsCol, ns.DSCount,
				ns.Age,
			)

			var style lipgloss.Style
			if i == m.k8sNamespaceCursor {
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
			{"Enter", "Workloads"},
			{"/", "Filter"},
			{"1-7", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}
