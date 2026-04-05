package main

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// probeInfo holds the results of a host probe.
type probeInfo struct {
	FQDN           string
	OS             string
	UpSince        string
	ServiceCount   int
	ContainerCount int
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
			`podman ps -q 2>/dev/null | wc -l`,
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

// parseProbeOutput parses the 5-line output from the probe command.
func parseProbeOutput(output string, systemdMode string) (probeInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		return probeInfo{}, fmt.Errorf("unexpected probe output (%d lines): %s", len(lines), output)
	}

	svcCount, _ := strconv.Atoi(strings.TrimSpace(lines[3]))
	ctnCount, _ := strconv.Atoi(strings.TrimSpace(lines[4]))

	return probeInfo{
		FQDN:           strings.TrimSpace(lines[0]),
		UpSince:        strings.TrimSpace(lines[1]),
		OS:             strings.TrimSpace(lines[2]),
		ServiceCount:   svcCount,
		ContainerCount: ctnCount,
		SystemdMode:    systemdMode,
	}, nil
}
