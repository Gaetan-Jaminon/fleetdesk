package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderFleetPicker() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader("", m.fleetCursor+1, len(m.fleets)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.fleets) == 0 {
		noFleetMsg := "  No fleet files found"
		if m.appCfg.FleetDir != "" {
			noFleetMsg += " in " + m.appCfg.FleetDir
		}
		s += borderedRow(noFleetMsg, iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("FLEET")
		for _, f := range m.fleets {
			if len(f.Name) > nameCol {
				nameCol = len(f.Name)
			}
		}
		nameCol += 2

		hdr := fmt.Sprintf("     %-*s  %-6s  %s", nameCol, "FLEET", "TYPE", "TARGETS")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.fleetCursor >= offset+maxVisible {
			offset = m.fleetCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.fleets) {
			end = len(m.fleets)
		}

		// build type group start map
		typeStarts := make(map[int]string)
		lastType := ""
		for i, f := range m.fleets {
			if f.Type != lastType {
				label := f.Type
				if label == "kubernetes" {
					label = "Kubernetes"
				} else if label == "azure" {
					label = "Azure"
				} else {
					label = "VM"
				}
				typeStarts[i] = label
				lastType = f.Type
			}
		}

		for i := offset; i < end; i++ {
			// type group header
			if typeName, ok := typeStarts[i]; ok {
				groupLine := fmt.Sprintf("  \u2500\u2500 %s \u2500\u2500", typeName)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
			}

			f := m.fleets[i]
			cur := "   "
			if i == m.fleetCursor {
				cur = " \u25b8 "
			}

			ftype := f.Type
			if ftype == "kubernetes" {
				ftype = "k8s"
			}
			var targetCount int
			if f.Type == "vm" {
				targetCount = m.fleetHostCount(f)
			} else {
				targetCount = len(f.Groups)
			}
			line := fmt.Sprintf("%s  %-*s  %-6s  %d", cur, nameCol, f.Name, ftype, targetCount)

			var style lipgloss.Style
			if i == m.fleetCursor {
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
	s += m.renderSudoPromptOrHintBar([][]string{
		{"Enter", "Select"},
		{"e", "Edit"},
		{"c", "Config"},
		{"r", "Reload"},
		{"q", "Quit"},
	})
	return s
}
