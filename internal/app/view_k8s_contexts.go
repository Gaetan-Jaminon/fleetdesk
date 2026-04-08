package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderK8sContextList() string {
	f := m.fleets[m.selectedFleet]
	cluster := m.k8sClusters[m.selectedK8sCluster]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + cluster.Name
	s := m.renderHeader(breadcrumb, m.k8sContextCursor+1, len(m.k8sContexts)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if m.k8sContexts == nil {
		s += borderedRow("  Loading contexts...", iw, normalRowStyle) + "\n"
	} else if len(m.k8sContexts) == 0 {
		s += borderedRow("  No contexts found for this cluster.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("CONTEXT")
		userCol := len("USER")
		for _, ctx := range m.k8sContexts {
			if len(ctx.Name) > nameCol {
				nameCol = len(ctx.Name)
			}
			if len(ctx.User) > userCol {
				userCol = len(ctx.User)
			}
		}
		nameCol += 2
		userCol += 2

		hdr := fmt.Sprintf("     %-*s  %s",
			nameCol, "CONTEXT"+m.sortIndicator(1),
			"USER"+m.sortIndicator(2),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.k8sContextCursor >= offset+maxVisible {
			offset = m.k8sContextCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.k8sContexts) {
			end = len(m.k8sContexts)
		}

		for i := offset; i < end; i++ {
			ctx := m.k8sContexts[i]
			cur := "   "
			if i == m.k8sContextCursor {
				cur = " ▸ "
			}

			prefix := ""
			if ctx.Current {
				prefix = "* "
			}

			line := fmt.Sprintf("%s  %s%-*s  %s",
				cur, prefix, nameCol-len(prefix), ctx.Name, ctx.User)

			var style lipgloss.Style
			if i == m.k8sContextCursor {
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
	} else {
		s += m.renderHintBar([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Resources"},
			{"d", "Delete Context"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}
