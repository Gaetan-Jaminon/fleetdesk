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
	FQDN              string
	OS                string
	UpSince           string
	ServiceCount      int
	ServiceRunning    int
	ServiceFailed     int
	ContainerCount    int
	ContainerRunning  int
	CronCount         int
	ErrorCount        int
	UpdateCount       int
	DiskCount         int
	DiskHighCount     int
	LastUpdate        string
	LastSecurity      string
	SystemdMode       string // the mode that actually worked
}

// probe runs a single SSH command to gather all host info in one roundtrip.
func probe(client *ssh.Client, systemdMode string, errorLogSince string) (probeInfo, error) {
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
			`%s list-units --type=service --state=running --no-pager -q 2>/dev/null | wc -l && `+
			`%s list-units --type=service --state=failed --no-pager -q 2>/dev/null | wc -l && `+
			`podman ps -q 2>/dev/null | wc -l && `+
			`podman ps -a -q 2>/dev/null | wc -l && `+
			`(dnf history list 2>/dev/null | grep -E '\| update ' | grep -v mdatp | head -1 | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' || echo unknown) && `+
			`(dnf history list 2>/dev/null | grep -E '\| update --security' | head -1 | awk -F'|' '{gsub(/^ +| +$/,"",$3); print $3}' || echo unknown) && `+
			`echo $(( $(crontab -l 2>/dev/null | grep -v '^#' | grep -v '^$' | wc -l) + $(ls /etc/cron.d/ 2>/dev/null | wc -l) )) && `+
			`sudo journalctl -p err --since '%s' --no-pager -q 2>/dev/null | wc -l && `+
			`dnf check-update --quiet 2>/dev/null | grep -E '^\S+\.\S+\s' | wc -l && `+
			`df -h --output=pcent -x tmpfs -x devtmpfs 2>/dev/null | tail -n+2 | wc -l && `+
			`df -h --output=pcent -x tmpfs -x devtmpfs 2>/dev/null | tail -n+2 | awk '{gsub(/%%/,""); if ($1 >= 80) print}' | wc -l`,
		sysctl, sysctl, sysctl, errorLogSince,
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

// parseProbeOutput parses the probe command output.
// Lines: hostname, uptime, os, svc_total, svc_running, svc_failed, ctn_running, ctn_total,
//        last_update, last_security, cron_count, error_count, update_count, disk_count, disk_high
func parseProbeOutput(output string, systemdMode string) (probeInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 7 {
		return probeInfo{}, fmt.Errorf("unexpected probe output (%d lines): %s", len(lines), output)
	}

	getInt := func(idx int) int {
		if idx < len(lines) {
			v, _ := strconv.Atoi(strings.TrimSpace(lines[idx]))
			return v
		}
		return 0
	}

	getDate := func(idx int) string {
		if idx < len(lines) {
			if v := strings.TrimSpace(lines[idx]); v != "" && v != "unknown" {
				return formatDateEU(v)
			}
		}
		return "—"
	}

	return probeInfo{
		FQDN:             strings.TrimSpace(lines[0]),
		UpSince:          formatDateEU(strings.TrimSpace(lines[1])),
		OS:               strings.TrimSpace(lines[2]),
		ServiceCount:     getInt(3),
		ServiceRunning:   getInt(4),
		ServiceFailed:    getInt(5),
		ContainerRunning: getInt(6),
		ContainerCount:   getInt(7),
		LastUpdate:       getDate(8),
		LastSecurity:     getDate(9),
		CronCount:        getInt(10),
		ErrorCount:       getInt(11),
		UpdateCount:      getInt(12),
		DiskCount:        getInt(13),
		DiskHighCount:    getInt(14),
		SystemdMode:      systemdMode,
	}, nil
}
