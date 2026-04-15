package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderResourcePicker() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " \u203a " + h.Entry.Name
	s := m.renderHeader(breadcrumb, m.resourceCursor+1, resourceCount) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	nameCol := len("SELinux Denials") + 2

	hdr := fmt.Sprintf("     %-*s  %7s  %7s  %7s", nameCol, "RESOURCE", "TOTAL", "RUNNING", "FAILED")
	s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"

	type resRow struct {
		name    string
		total   int
		running int
		failed  int
	}

	// use fetched (filtered) data if available, otherwise probe counts
	svcTotal, svcRunning, svcFailed := 0, 0, 0
	if len(m.services) > 0 {
		svcTotal = len(m.services)
		for _, s := range m.services {
			switch s.State {
			case "running":
				svcRunning++
			case "failed":
				svcFailed++
			}
		}
	}
	ctnTotal, ctnRunning, ctnFailed := 0, 0, 0
	if len(m.containers) > 0 {
		ctnTotal = len(m.containers)
		for _, c := range m.containers {
			if strings.HasPrefix(c.Status, "Up") {
				ctnRunning++
			} else if !strings.HasPrefix(c.Status, "Exited (0)") && c.Status != "Created" {
				ctnFailed++
			}
		}
	}

	updTotal, updFailed := 0, 0
	if len(m.updates) > 0 {
		for _, u := range m.updates {
			if u.Type == "error" {
				updFailed++
			} else {
				updTotal++
			}
		}
	}

	rows := []resRow{
		{"Services", svcTotal, svcRunning, svcFailed},
		{"Containers", ctnTotal, ctnRunning, ctnFailed},
		{"Cron Jobs", h.CronCount, 0, 0},
		{"System Logs", h.ErrorCount, 0, 0},
		{"Updates", updTotal, 0, updFailed},
		{"Disk", h.DiskCount, 0, h.DiskHighCount},
		{"Subscription", 0, 0, 0},
		{"Accounts", h.UserCount, 0, 0},
		{"Network", h.InterfacesTotal, h.InterfacesUp, 0},
	}
	for i, r := range rows {
		cur := "   "
		if i == m.resourceCursor {
			cur = " \u25b8 "
		}
		failedStr := fmt.Sprintf("%d", r.failed)
		line := fmt.Sprintf("%s  %-*s  %7d  %7d  %7s", cur, nameCol, r.name, r.total, r.running, failedStr)

		var style lipgloss.Style
		if i == m.resourceCursor {
			style = selectedRowStyle
		} else if i%2 == 0 {
			style = altRowStyle
		} else {
			style = normalRowStyle
		}
		s += borderedRow(line, iw, style) + "\n"
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"\u2191\u2193", "Navigate"},
		{"Enter", "Select"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	}))
	return s
}
