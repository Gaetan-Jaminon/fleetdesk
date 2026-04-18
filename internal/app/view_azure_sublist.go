package app

import (
	"fmt"
	"strings"

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

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}

	nameCol := len("SUBSCRIPTION")
	for _, sub := range m.azureSubs {
		if len(sub.Name) > nameCol {
			nameCol = len(sub.Name)
		}
	}
	nameCol += 2

	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "SUBSCRIPTION", Width: nameCol, SortIndex: 1},
			{Label: "TENANT", SortIndex: 2},
			{Label: "USER", SortIndex: 3},
		},
		RowCount: len(m.azureSubs),
		RowBuilder: func(i int) []string {
			sub := m.azureSubs[i]
			return []string{sub.Name, sub.Tenant, sub.User}
		},
		RowOverride: func(i int) string {
			sub := m.azureSubs[i]
			switch sub.Status {
			case azure.SubConnecting:
				return fmt.Sprintf("%-*s  checking...", nameCol, sub.Name)
			case azure.SubError:
				reason := sub.Error
				if reason == "" {
					reason = "unknown"
				}
				return fmt.Sprintf("%-*s  error (%s)", nameCol, sub.Name, reason)
			}
			return ""
		},
		Cursor:        m.azureSubCursor,
		MaxVisible:    maxVisible,
		InnerWidth:    iw,
		SortIndicator: m.sortIndicator,
		EmptyMessage:  "  No subscriptions in fleet.",
	})

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
