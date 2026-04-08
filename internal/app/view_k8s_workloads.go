package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	if m.k8sWorkloads == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No workloads.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("NAME")
		readyCol := 7
		utdCol := len("UP-TO-DATE")
		availCol := len("AVAILABLE")
		for _, wl := range filtered {
			if len(wl.Name) > nameCol {
				nameCol = len(wl.Name)
			}
			if len(wl.Ready) > readyCol {
				readyCol = len(wl.Ready)
			}
		}
		nameCol += 2
		readyCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %*s  %*s  %s",
			nameCol, "NAME"+m.sortIndicator(1),
			readyCol, "READY"+m.sortIndicator(2),
			utdCol, "UP-TO-DATE"+m.sortIndicator(3),
			availCol, "AVAILABLE"+m.sortIndicator(4),
			"AGE"+m.sortIndicator(5),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.k8sWorkloadCursor >= offset+maxVisible {
			offset = m.k8sWorkloadCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		// build group start index map by kind
		groupStarts := make(map[int]string)
		kindLabels := map[string]string{
			"Deployment":  "Deployments",
			"StatefulSet": "StatefulSets",
			"DaemonSet":   "DaemonSets",
		}
		for i, wl := range filtered {
			if i == 0 || filtered[i-1].Kind != wl.Kind {
				label := kindLabels[wl.Kind]
				if label == "" {
					label = wl.Kind
				}
				groupStarts[i] = label
			}
		}

		for i := offset; i < end; i++ {
			// render group header if this workload starts a new kind
			if groupLabel, ok := groupStarts[i]; ok {
				groupLine := fmt.Sprintf("  \u2500\u2500 %s \u2500\u2500", groupLabel)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
			}

			wl := filtered[i]
			cur := "   "
			if i == m.k8sWorkloadCursor {
				cur = " \u25b8 "
			}

			utd := "\u2014"
			avail := "\u2014"
			if wl.Kind == "Deployment" {
				utd = fmt.Sprintf("%d", wl.UpToDate)
				avail = fmt.Sprintf("%d", wl.Available)
			}

			line := fmt.Sprintf("%s  %-*s  %-*s  %*s  %*s  %s",
				cur, nameCol, wl.Name,
				readyCol, wl.Ready,
				utdCol, utd,
				availCol, avail,
				wl.Age,
			)

			var style lipgloss.Style
			if i == m.k8sWorkloadCursor {
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
			{"Enter", "Pods"},
			{"/", "Filter"},
			{"1-5", "Sort"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}
