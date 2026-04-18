package app

import (
	"fmt"
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/k8s"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/notes"
)

func (m Model) renderK8sClusterList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader(f.Name, m.k8sClusterCursor+1, len(m.k8sClusters)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}

	nameCol := len("CLUSTER")
	for _, c := range m.k8sClusters {
		if len(c.Name) > nameCol {
			nameCol = len(c.Name)
		}
	}
	nameCol += 2

	fleetName := m.fleets[m.selectedFleet].Name
	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "CLUSTER", Width: nameCol, SortIndex: 1},
			{Label: "K8S VERSION", SortIndex: 2},
		},
		RowCount: len(m.k8sClusters),
		RowBuilder: func(i int) []string {
			c := m.k8sClusters[i]
			return []string{c.Name, c.K8sVersion}
		},
		RowPrefix: func(i int) string {
			return m.notePrefix(notes.ResourceRef{
				Fleet:    fleetName,
				Segments: []string{"k8s", m.k8sClusters[i].Name},
			})
		},
		RowOverride: func(i int) string {
			c := m.k8sClusters[i]
			switch c.Status {
			case k8s.ClusterChecking:
				return fmt.Sprintf("%-*s  checking...", nameCol, c.Name)
			case k8s.ClusterError:
				return fmt.Sprintf("%-*s  unavailable", nameCol, c.Name)
			}
			return ""
		},
		Cursor:        m.k8sClusterCursor,
		MaxVisible:    maxVisible,
		InnerWidth:    iw,
		SortIndicator: m.sortIndicator,
		EmptyMessage:  "  No clusters in fleet.",
	})

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		s += m.renderHintBar(hintWithHelp([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Contexts"},
			{"r", "Refresh"},
			{"/", "Filter"},
			{"1-2", "Sort"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}))
	}
	return s
}
