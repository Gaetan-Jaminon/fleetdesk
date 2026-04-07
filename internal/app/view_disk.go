package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDiskList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " \u203a " + h.Entry.Name + " \u203a Disk"
	s := m.renderHeader(breadcrumb, m.diskCursor+1, len(m.disks)) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	if len(m.disks) == 0 {
		s += borderedRow("  No partitions found.", iw, normalRowStyle) + "\n"
	} else {
		fsCol := len("FILESYSTEM")
		mountCol := len("MOUNT")
		for _, d := range m.disks {
			if len(d.Filesystem) > fsCol {
				fsCol = len(d.Filesystem)
			}
			if len(d.Mount) > mountCol {
				mountCol = len(d.Mount)
			}
		}
		fsCol += 2
		mountCol += 2

		hdr := fmt.Sprintf("     %-*s  %6s  %6s  %6s  %5s  %-*s", fsCol, "FILESYSTEM", "SIZE", "USED", "AVAIL", "USE%", mountCol, "MOUNT")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.diskCursor >= offset+maxVisible {
			offset = m.diskCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.disks) {
			end = len(m.disks)
		}

		for i := offset; i < end; i++ {
			d := m.disks[i]
			cur := "   "
			if i == m.diskCursor {
				cur = " \u25b8 "
			}
			line := fmt.Sprintf("%s  %-*s  %6s  %6s  %6s  %5s  %-*s", cur, fsCol, d.Filesystem, d.Size, d.Used, d.Avail, d.UsePercent, mountCol, d.Mount)

			var style lipgloss.Style
			if i == m.diskCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}

			// highlight high disk usage
			pct := strings.TrimSuffix(d.UsePercent, "%")
			var pctVal int
			fmt.Sscanf(pct, "%d", &pctVal)
			if pctVal >= 90 && i != m.diskCursor {
				style = flashErrorStyle
			} else if pctVal >= 80 && i != m.diskCursor {
				style = lipgloss.NewStyle().Foreground(colorYellow)
			}

			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"

	s += m.renderHintBar([][]string{
		{"↑↓", "Navigate"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}
