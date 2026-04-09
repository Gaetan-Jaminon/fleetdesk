package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/k8s"
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

	if len(m.k8sClusters) == 0 {
		s += borderedRow("  No clusters in fleet.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("CLUSTER")
		verCol := len("K8S VERSION")
		for _, c := range m.k8sClusters {
			if len(c.Name) > nameCol {
				nameCol = len(c.Name)
			}
		}
		nameCol += 2
		verCol += 2

		hdr := fmt.Sprintf("     %-*s  %s",
			nameCol, "CLUSTER"+m.sortIndicator(1),
			"K8S VERSION"+m.sortIndicator(2),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.k8sClusterCursor >= offset+maxVisible {
			offset = m.k8sClusterCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.k8sClusters) {
			end = len(m.k8sClusters)
		}

		for i := offset; i < end; i++ {
			c := m.k8sClusters[i]
			cur := "   "
			if i == m.k8sClusterCursor {
				cur = " ▸ "
			}

			var line string
			switch c.Status {
			case k8s.ClusterChecking:
				line = fmt.Sprintf("%s  %-*s  checking...", cur, nameCol, c.Name)
			case k8s.ClusterError:
				line = fmt.Sprintf("%s  %-*s  unavailable", cur, nameCol, c.Name)
			case k8s.ClusterOnline:
				line = fmt.Sprintf("%s  %-*s  %s", cur, nameCol, c.Name, c.K8sVersion)
			}

			var style lipgloss.Style
			if i == m.k8sClusterCursor {
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
		s += m.renderHintBar([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Contexts"},
			{"r", "Refresh"},
			{"/", "Filter"},
			{"1-2", "Sort"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}
