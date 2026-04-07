package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAccountList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	accts := m.filteredAccounts()

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Accounts"
	s := m.renderHeader(breadcrumb, m.accountCursor+1, len(accts)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if m.filterActive {
		s += borderedRow(fmt.Sprintf("  / %s\u2588", m.filterText), iw, normalRowStyle) + "\n"
	}

	if accts == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(accts) == 0 {
		s += borderedRow("  No accounts found.", iw, normalRowStyle) + "\n"
	} else {
		// compute dynamic column widths
		userCol := len("USER")
		groupCol := len("GROUPS")
		shellCol := len("SHELL")
		loginCol := len("LAST LOGIN")
		for _, a := range accts {
			if len(a.User) > userCol {
				userCol = len(a.User)
			}
			g := a.Groups
			if a.IsSudo {
				g = "\u2605 " + g
			}
			if len(g) > groupCol {
				groupCol = len(g)
			}
			if len(a.Shell) > shellCol {
				shellCol = len(a.Shell)
			}
			if len(a.LastLogin) > loginCol {
				loginCol = len(a.LastLogin)
			}
		}
		userCol += 2
		groupCol += 2
		shellCol += 2
		loginCol += 2

		hdr := fmt.Sprintf("     %-*s  %5s  %-*s  %-*s  %-*s  %s", userCol, "USER", "UID", groupCol, "GROUPS", shellCol, "SHELL", loginCol, "LAST LOGIN", "PASSWORD")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if m.filterActive {
			maxVisible--
		}
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.accountCursor >= offset+maxVisible {
			offset = m.accountCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(accts) {
			end = len(accts)
		}

		for i := offset; i < end; i++ {
			a := accts[i]

			cur := "   "
			if i == m.accountCursor {
				cur = " \u25b8 "
			}

			user := a.User
			if a.IsLocked {
				user = "\U0001f512 " + user
			}

			groups := a.Groups
			if a.IsSudo {
				groups = "\u2605 " + groups
			}

			pw := a.PasswordStatus
			switch pw {
			case "PS":
				pw = "Set"
			case "LK", "L":
				pw = "Locked"
			case "NP":
				pw = "No Password"
			}

			line := fmt.Sprintf("%s  %-*s  %5d  %-*s  %-*s  %-*s  %s", cur, userCol, user, a.UID, groupCol, groups, shellCol, a.Shell, loginCol, a.LastLogin, pw)

			var style lipgloss.Style
			if i == m.accountCursor {
				style = selectedRowStyle
			} else if a.IsLocked {
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
	s += m.renderHintBar([][]string{
		{"\u2191\u2193", "Navigate"},
		{"i", "Detail"},
		{"/", "Search"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
