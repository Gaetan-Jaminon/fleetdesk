package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/azure"
)

func (m Model) renderAzureSubList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name
	if ver := m.azure.Version(); ver != "" {
		breadcrumb += " (az " + ver + ")"
	}
	s := m.renderHeader(breadcrumb, m.azureSubCursor+1, len(m.azureSubs)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.azureSubs) == 0 {
		s += borderedRow("  No subscriptions in fleet.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("SUBSCRIPTION")
		tenantCol := len("TENANT")
		userCol := len("USER")
		for _, sub := range m.azureSubs {
			if len(sub.Name) > nameCol {
				nameCol = len(sub.Name)
			}
			if len(sub.Tenant) > tenantCol {
				tenantCol = len(sub.Tenant)
			}
			if len(sub.User) > userCol {
				userCol = len(sub.User)
			}
		}
		nameCol += 2
		tenantCol += 2
		userCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s",
			nameCol, "SUBSCRIPTION"+m.sortIndicator(1),
			tenantCol, "TENANT"+m.sortIndicator(2),
			userCol, "USER"+m.sortIndicator(3),
		)
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.azureSubCursor >= offset+maxVisible {
			offset = m.azureSubCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.azureSubs) {
			end = len(m.azureSubs)
		}

		for i := offset; i < end; i++ {
			sub := m.azureSubs[i]
			cur := "   "
			if i == m.azureSubCursor {
				cur = " ▸ "
			}

			var line string
			switch sub.Status {
			case azure.SubConnecting:
				line = fmt.Sprintf("%s  %-*s  checking...", cur, nameCol, sub.Name)
			case azure.SubError:
				reason := sub.Error
				if reason == "" {
					reason = "unknown"
				}
				line = fmt.Sprintf("%s  %-*s  error (%s)", cur, nameCol, sub.Name, reason)
			case azure.SubOnline:
				line = fmt.Sprintf("%s  %-*s  %-*s  %-*s",
					cur, nameCol, sub.Name, tenantCol, sub.Tenant, userCol, sub.User)
			}

			var style lipgloss.Style
			if i == m.azureSubCursor {
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
		s += m.renderHintBar(hintWithHelp([][]string{
			{"↑↓", "Navigate"},
			{"Enter", "Drill In"},
			{"r", "Refresh"},
			{"/", "Filter"},
			{"1-3", "Sort"},
			{"Esc", "Back"},
			{"q", "Quit"},
		}))
	}
	return s
}
