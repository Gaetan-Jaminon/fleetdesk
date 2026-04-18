package app

import (
	"fmt"
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/notes"
)

func (m Model) renderFleetPicker() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader("", m.fleetCursor+1, len(m.fleets)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	maxVisible := m.height - 8
	if maxVisible < 1 {
		maxVisible = 1
	}

	emptyMsg := "  No fleet files found"
	if m.appCfg.FleetDir != "" {
		emptyMsg += " in " + m.appCfg.FleetDir
	}

	typeLabel := func(t string) string {
		switch t {
		case "kubernetes":
			return "Kubernetes"
		case "azure":
			return "Azure"
		case "probes":
			return "Probes"
		default:
			return "VM"
		}
	}

	s += renderList(ListConfig{
		Columns: []ListColumn{
			{Label: "FLEET"},
			{Label: "TYPE", Width: 6},
			{Label: "TARGETS"},
		},
		RowCount: len(m.fleets),
		RowBuilder: func(i int) []string {
			f := m.fleets[i]
			ftype := f.Type
			if ftype == "kubernetes" {
				ftype = "k8s"
			}
			var targetCount int
			switch f.Type {
			case "vm":
				targetCount = m.fleetHostCount(f)
			case "probes":
				targetCount = fleetProbeCount(f)
			default:
				targetCount = len(f.Groups)
			}
			return []string{f.Name, ftype, fmt.Sprintf("%d", targetCount)}
		},
		RowPrefix: func(i int) string {
			return m.notePrefix(notes.ResourceRef{Fleet: m.fleets[i].Name})
		},
		GroupHeader: func(i int) (string, bool) {
			if i > 0 && m.fleets[i-1].Type == m.fleets[i].Type {
				return "", false
			}
			return typeLabel(m.fleets[i].Type), true
		},
		Cursor:       m.fleetCursor,
		MaxVisible:   maxVisible,
		InnerWidth:   iw,
		EmptyMessage: emptyMsg,
	})

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"Enter", "Select"},
		{"n", "Notes"},
		{"a", "About"},
		{"e", "Edit"},
		{"c", "Config"},
		{"r", "Reload"},
		{"q", "Quit"},
	}))
	return s
}
