package main

import (
	"fmt"

	"golang.org/x/crypto/ssh"

	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// probeInfo is an alias for internal/ssh.ProbeInfo.
type probeInfo = issh.ProbeInfo

// formatDateEU delegates to internal/ssh.FormatDateEU.
var formatDateEU = issh.FormatDateEU

// parseProbeOutput delegates to internal/ssh.ParseProbeOutput.
var parseProbeOutput = issh.ParseProbeOutput

// probe runs a single SSH command to gather all host info in one roundtrip.
func probe(client *ssh.Client, systemdMode string, errorLogSince string) (probeInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return probeInfo{}, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

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
