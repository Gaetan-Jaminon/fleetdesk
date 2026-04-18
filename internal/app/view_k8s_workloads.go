package app

import (
	"fmt"
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/notes"
)

func (m Model) renderK8sWorkloadList() string {
	f := m.fleets[m.selectedFleet]
	cluster := m.k8sClusters[m.selectedK8sCluster]
	ns := m.k8sNamespaces[m.selectedK8sNamespace]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredK8sWorkloads()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " \u203a " + cluster.Name + " \u203a " + m.selectedK8sContext + " \u203a " + ns.Name
	s := m.renderHeader(breadcrumb+filterInfo, m.k8sWorkloadCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}

	emptyMsg := "  No workloads."
	if m.filterText != "" {
		emptyMsg = fmt.Sprintf("  No matches for '%s'", m.filterText)
	}

	kindLabels := map[string]string{
		"Deployment":  "Deployments",
		"StatefulSet": "StatefulSets",
		"DaemonSet":   "DaemonSets",
	}

	fleetName := m.fleets[m.selectedFleet].Name
	clusterName := m.k8sClusters[m.selectedK8sCluster].Name
	nsName := m.k8sNamespaces[m.selectedK8sNamespace].Name
	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "NAME", SortIndex: 1},
			{Label: "READY", SortIndex: 2},
			{Label: "AGE", SortIndex: 3},
		},
		RowCount: len(filtered),
		RowBuilder: func(i int) []string {
			wl := filtered[i]
			return []string{wl.Name, wl.Ready, wl.Age}
		},
		RowPrefix: func(i int) string {
			return m.notePrefix(notes.ResourceRef{
				Fleet:    fleetName,
				Segments: []string{"k8s", clusterName, m.selectedK8sContext, nsName, filtered[i].Name},
			})
		},
		GroupHeader: func(i int) (string, bool) {
			if i > 0 && filtered[i-1].Kind == filtered[i].Kind {
				return "", false
			}
			label := kindLabels[filtered[i].Kind]
			if label == "" {
				label = filtered[i].Kind
			}
			return label, true
		},
		Cursor:        m.k8sWorkloadCursor,
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
			{"Enter", "Pods"},
			{"/", "Filter"},
			{"1-3", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}))
	}
	return s
}
