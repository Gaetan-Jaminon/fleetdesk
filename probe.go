package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// formatDateEU converts a date string to DD/MM/YYYY format.
// Handles: "2026-03-31 09:57", "2026-04-05 15:56:35", etc.
func formatDateEU(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || s == "unknown" {
		return s
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("02/01/2006")
		}
	}
	return s
}

// probeInfo holds the results of a host probe.
type probeInfo struct {
	FQDN           string
	OS             string
	UpSince        string
	ServiceCount   int
	ContainerCount int
	LastUpdate     string
	LastSecurity   string
	SystemdMode    string // the mode that actually worked
}

// probe runs a single SSH command to gather all host info in one roundtrip.
func probe(client *ssh.Client, systemdMode string) (probeInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return probeInfo{}, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// build the systemctl command based on mode
	sysctl := "systemctl"
	if systemdMode == "user" {
		sysctl = "systemctl --user"
	}

	cmd := fmt.Sprintf(
		`hostname -f 2>/dev/null || hostname && `+
			`uptime -s 2>/dev/null || echo unknown && `+
			`(grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"') || echo unknown && `+
			`%s list-units --type=service --no-pager -q 2>/dev/null | wc -l && `+
			`podman ps -q 2>/dev/null | wc -l && `+
			`(dnf history list 2>/dev/null | grep -E '\| update ' | grep -v mdatp | head -1 | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' || echo unknown) && `+
			`(dnf history list 2>/dev/null | grep -E '\| update --security' | head -1 | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' || echo unknown)`,
		sysctl,
	)

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		// try the opposite systemd mode
		session2, err2 := client.NewSession()
		if err2 != nil {
			return probeInfo{}, fmt.Errorf("probe failed: %w", err)
		}
		defer session2.Close()

		if systemdMode == "user" {
			sysctl = "systemctl"
			systemdMode = "system"
		} else {
			sysctl = "systemctl --user"
			systemdMode = "user"
		}

		cmd = fmt.Sprintf(
			`hostname -f 2>/dev/null || hostname && `+
				`uptime -s 2>/dev/null || echo unknown && `+
				`(grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"') || echo unknown && `+
				`%s list-units --type=service --no-pager -q 2>/dev/null | wc -l && `+
				`podman ps -q 2>/dev/null | wc -l`,
			sysctl,
		)

		out, err = session2.CombinedOutput(cmd)
		if err != nil {
			return probeInfo{}, fmt.Errorf("probe failed both modes: %w", err)
		}
	}

	return parseProbeOutput(string(out), systemdMode)
}

// parseProbeOutput parses the 7-line output from the probe command.
func parseProbeOutput(output string, systemdMode string) (probeInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		return probeInfo{}, fmt.Errorf("unexpected probe output (%d lines): %s", len(lines), output)
	}

	svcCount, _ := strconv.Atoi(strings.TrimSpace(lines[3]))
	ctnCount, _ := strconv.Atoi(strings.TrimSpace(lines[4]))

	lastUpdate := "—"
	lastSecurity := "—"
	if len(lines) > 5 {
		if v := strings.TrimSpace(lines[5]); v != "" && v != "unknown" {
			lastUpdate = formatDateEU(v)
		}
	}
	if len(lines) > 6 {
		if v := strings.TrimSpace(lines[6]); v != "" && v != "unknown" {
			lastSecurity = formatDateEU(v)
		}
	}

	return probeInfo{
		FQDN:           strings.TrimSpace(lines[0]),
		UpSince:        formatDateEU(strings.TrimSpace(lines[1])),
		OS:             strings.TrimSpace(lines[2]),
		ServiceCount:   svcCount,
		ContainerCount: ctnCount,
		LastUpdate:     lastUpdate,
		LastSecurity:   lastSecurity,
		SystemdMode:    systemdMode,
	}, nil
}
