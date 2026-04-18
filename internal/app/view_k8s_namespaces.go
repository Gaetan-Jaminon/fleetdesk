package app

import (
	"fmt"
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/notes"
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

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}

	emptyMsg := "  No namespaces."
	if m.filterText != "" {
		emptyMsg = fmt.Sprintf("  No matches for '%s'", m.filterText)
	}

	fleetName := m.fleets[m.selectedFleet].Name
	clusterName := m.k8sClusters[m.selectedK8sCluster].Name
	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "NAMESPACE", SortIndex: 1},
			{Label: "STATUS", SortIndex: 2},
			{Label: "PODS", Width: 5, SortIndex: 3, RightAlign: true},
			{Label: "DEPLOY", Width: 6, SortIndex: 4, RightAlign: true},
			{Label: "STS", Width: 4, SortIndex: 5, RightAlign: true},
			{Label: "DS", Width: 3, SortIndex: 6, RightAlign: true},
			{Label: "AGE", SortIndex: 7},
		},
		RowCount: len(filtered),
		RowBuilder: func(i int) []string {
			ns := filtered[i]
			return []string{
				ns.Name,
				ns.Status,
				fmt.Sprintf("%d", ns.PodCount),
				fmt.Sprintf("%d", ns.DeployCount),
				fmt.Sprintf("%d", ns.STSCount),
				fmt.Sprintf("%d", ns.DSCount),
				ns.Age,
			}
		},
		RowPrefix: func(i int) string {
			return m.notePrefix(notes.ResourceRef{
				Fleet:    fleetName,
				Segments: []string{"k8s", clusterName, m.selectedK8sContext, filtered[i].Name},
			})
		},
		Cursor:        m.k8sNamespaceCursor,
		MaxVisible:    maxVisible,
		InnerWidth:    iw,
		SortIndicator: m.sortIndicator,
		EmptyMessage:  emptyMsg,
	})

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s\u2588", m.filterText))
	} else {
		s += m.renderHintBar(hintWithHelp([][]string{
			{"\u2191\u2193", "Navigate"},
			{"Enter", "Workloads"},
			{"/", "Filter"},
			{"1-7", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}))
	}
	return s
}
