package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/probes"
)

func (m Model) renderProbeDetail() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredProbeItems()
	if m.selectedProbe >= len(filtered) || len(filtered) == 0 {
		return m.renderProbeList()
	}
	p := filtered[m.selectedProbe]
	r := p.Result

	f := m.fleets[m.selectedFleet]
	breadcrumb := fmt.Sprintf("%s › %s", f.Name, p.Entry.Name)
	s := m.renderHeader(breadcrumb, 0, 0) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	// Build content lines
	var content strings.Builder

	// Summary section
	content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")
	content.WriteString(borderedRow("  ── Summary ──", iw, colHeaderStyle) + "\n")
	content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")

	statusStr, statusColor := probeStatusDisplay(r.Status)

	tlsVerify := "Yes"
	if m.fleets[m.selectedFleet].ProbeFleet.Defaults.InsecureSkipVerify {
		tlsVerify = ansiColor("Skipped", "33")
	}

	kvSummary := []struct{ k, v string }{
		{"Name", p.Entry.Name},
		{"URL", p.Entry.URL},
		{"Protocol", p.Entry.Protocol},
		{"Expected Code", fmt.Sprintf("%d", p.Entry.ExpectedCode)},
		{"TLS Verify", tlsVerify},
		{"Status", statusColor(statusStr)},
	}
	if !r.ProbeTime.IsZero() {
		kvSummary = append(kvSummary, struct{ k, v string }{"Last Check", r.ProbeTime.Format(time.TimeOnly)})
	}
	keyWidth := 0
	for _, kv := range kvSummary {
		if len(kv.k) > keyWidth {
			keyWidth = len(kv.k)
		}
	}
	for _, kv := range kvSummary {
		line := fmt.Sprintf("    %-*s  %s", keyWidth, kv.k, kv.v)
		content.WriteString(borderedRow(line, iw, normalRowStyle) + "\n")
	}

	// Timing section
	content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")
	content.WriteString(borderedRow("  ── Timing ──", iw, colHeaderStyle) + "\n")
	content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")

	latencyStr := "---"
	if r.Latency > 0 {
		latencyStr = formatLatency(r.Latency)
	}
	ttfbStr := "---"
	if r.TTFB > 0 {
		ttfbStr = formatLatency(r.TTFB)
	}
	kvTiming := []struct{ k, v string }{
		{"Latency", latencyStr},
		{"TTFB", ttfbStr},
	}
	for _, kv := range kvTiming {
		line := fmt.Sprintf("    %-*s  %s", keyWidth, kv.k, kv.v)
		content.WriteString(borderedRow(line, iw, normalRowStyle) + "\n")
	}

	// TLS section (conditional — hidden on plain HTTP)
	if r.TLS != nil {
		content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")
		content.WriteString(borderedRow("  ── TLS ──", iw, colHeaderStyle) + "\n")
		content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")

		expiryStr := r.TLS.Expiry.Format("2006-01-02")
		daysStr := fmt.Sprintf("%d days", r.TLS.DaysToExpiry)
		if r.TLS.DaysToExpiry < probes.CertWarnDays {
			daysStr = "\033[33m" + daysStr + "\033[0m" // yellow warning
		}

		kvTLS := []struct{ k, v string }{
			{"Version", r.TLS.Version},
			{"Issuer", r.TLS.Issuer},
			{"Subject", r.TLS.Subject},
			{"Expires", fmt.Sprintf("%s (%s)", expiryStr, daysStr)},
		}
		for _, kv := range kvTLS {
			line := fmt.Sprintf("    %-*s  %s", keyWidth, kv.k, kv.v)
			content.WriteString(borderedRow(line, iw, normalRowStyle) + "\n")
		}
	}

	// Response section
	if r.Code > 0 || r.BodyPreview != "" {
		content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")
		content.WriteString(borderedRow("  ── Response ──", iw, colHeaderStyle) + "\n")
		content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")

		if r.Code > 0 {
			codeLine := fmt.Sprintf("    %-*s  %d", keyWidth, "Status Code", r.Code)
			content.WriteString(borderedRow(codeLine, iw, normalRowStyle) + "\n")
		}

		if r.BodyPreview != "" {
			content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")
			bodyLines := strings.Split(r.BodyPreview, "\n")
			for _, bl := range bodyLines {
				content.WriteString(borderedRow("    "+bl, iw, normalRowStyle) + "\n")
			}
		}
	}

	// Error section (conditional)
	if r.Error != probes.ErrorClassNone && r.ErrorMsg != "" {
		content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")
		content.WriteString(borderedRow("  ── Error ──", iw, colHeaderStyle) + "\n")
		content.WriteString(borderedRow("", iw, normalRowStyle) + "\n")

		kvErr := []struct{ k, v string }{
			{"Class", string(r.Error)},
			{"Detail", r.ErrorMsg},
		}
		for _, kv := range kvErr {
			line := fmt.Sprintf("    %-*s  %s", keyWidth, kv.k, kv.v)
			content.WriteString(borderedRow(line, iw, normalRowStyle) + "\n")
		}
	}

	// Apply scroll
	contentStr := content.String()
	contentLines := strings.Split(contentStr, "\n")
	if len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}

	maxVisible := m.height - 4
	if maxVisible < 3 {
		maxVisible = 3
	}
	startLine := m.probeDetailScroll
	if startLine >= len(contentLines) {
		startLine = max(0, len(contentLines)-1)
	}
	endLine := startLine + maxVisible
	if endLine > len(contentLines) {
		endLine = len(contentLines)
	}

	for _, line := range contentLines[startLine:endLine] {
		s += line + "\n"
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"↑↓", "Scroll"},
		{"r", "Refresh"},
		{"Esc", "Back"},
		{"q", "Quit"},
	}))
	return s
}
