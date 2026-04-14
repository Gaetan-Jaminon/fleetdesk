package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSubscription() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Subscription"
	s := m.renderHeader(breadcrumb, m.subscriptionCursor+1, len(m.subscriptions)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.subscriptions) == 0 {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else {
		fieldCol := len("FIELD")
		for _, sub := range m.subscriptions {
			if len(sub.Field) > fieldCol {
				fieldCol = len(sub.Field)
			}
		}
		fieldCol += 2

		hdr := fmt.Sprintf("     %-*s  %s", fieldCol, "FIELD", "VALUE")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.subscriptionCursor >= offset+maxVisible {
			offset = m.subscriptionCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.subscriptions) {
			end = len(m.subscriptions)
		}

		for i := offset; i < end; i++ {
			sub := m.subscriptions[i]
			cur := "   "
			if i == m.subscriptionCursor {
				cur = " \u25b8 "
			}
			line := fmt.Sprintf("%s  %-*s  %s", cur, fieldCol, sub.Field, sub.Value)

			var style lipgloss.Style
			if i == m.subscriptionCursor {
				style = selectedRowStyle
			} else if sub.Value == "ERROR" {
				style = lipgloss.NewStyle().Foreground(colorRed)
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

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else {
		s += m.renderSudoPromptOrHintBar([][]string{
			{"↑↓", "Navigate"},
			{"u", "Unregister"},
			{"g", "Register CDN"},
			{"d", "Disable Repo"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}
