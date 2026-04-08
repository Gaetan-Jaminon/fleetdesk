package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

func (m Model) renderHeader(breadcrumb string, current, total int) string {
	title := "fleetdesk"
	if breadcrumb != "" {
		title += " \u203a " + breadcrumb
	}
	left := headerStyle.Render(title)
	right := headerCountStyle.Render(fmt.Sprintf("%d/%d", current, total))
	if m.version != "" {
		ver := lipgloss.NewStyle().Foreground(colorDimmed).Render("FleetDesk " + m.version)
		right = ver + "  " + right
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderHintBar(hints [][]string) string {
	var parts []string
	for _, h := range hints {
		parts = append(parts, hintKeyStyle.Render("<"+h[0]+">")+" "+hintActionStyle.Render(h[1]))
	}
	left := strings.Join(parts, "  ")
	if m.flash != "" {
		style := flashStyle
		if m.flashError {
			style = flashErrorStyle
		}
		left += "  " + style.Render(m.flash)
	}
	return hintBarStyle.Width(m.width).Render(left)
}

// borderedRow wraps content with | on each side, clamped to exactly w display columns.
func borderedRow(content string, w int, style lipgloss.Style) string {
	dw := runewidth.StringWidth(content)
	if dw > w {
		truncated := ""
		col := 0
		for _, r := range content {
			rw := runewidth.RuneWidth(r)
			if col+rw >= w {
				break
			}
			truncated += string(r)
			col += rw
		}
		content = truncated + "\u2026"
		dw = runewidth.StringWidth(content)
	}
	if dw < w {
		content += strings.Repeat(" ", w-dw)
	}
	b := borderStyle.Render("\u2502")
	return b + style.Render(content) + b
}

func (m Model) padToBottom(s string, iw int) string {
	lines := strings.Count(s, "\n")
	for i := lines; i < m.height-3; i++ {
		s += borderedRow("", iw, normalRowStyle) + "\n"
	}
	return s
}
