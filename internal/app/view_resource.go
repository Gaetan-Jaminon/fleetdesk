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
	resourceCount := 7
	s := m.renderHeader(breadcrumb, m.resourceCursor+1, resourceCount) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	nameCol := len("RESOURCE") + 4

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
	svcTotal, svcRunning, svcFailed := h.ServiceCount, h.ServiceRunning, h.ServiceFailed
	ctnTotal, ctnRunning := h.ContainerCount, h.ContainerRunning
	if len(m.services) > 0 {
		svcTotal = len(m.services)
		svcRunning = 0
		svcFailed = 0
		for _, s := range m.services {
			switch s.State {
			case "running":
				svcRunning++
			case "failed":
				svcFailed++
			}
		}
	}
	ctnFailed := 0
	if len(m.containers) > 0 {
		ctnTotal = len(m.containers)
		ctnRunning = 0
		for _, c := range m.containers {
			if strings.HasPrefix(c.Status, "Up") {
				ctnRunning++
			} else if !strings.HasPrefix(c.Status, "Exited (0)") && c.Status != "Created" {
				ctnFailed++
			}
		}
	}

	updTotal, updFailed := h.UpdateCount, 0
	if len(m.updates) > 0 {
		updTotal = 0
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
		{"Accounts", h.UserCount, 0, h.LockedUsers},
		{"Network", h.InterfacesTotal, h.InterfacesUp, h.ListeningPorts},
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
	s += m.renderHintBar([][]string{
		{"Enter", "Select"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})
	return s
}
