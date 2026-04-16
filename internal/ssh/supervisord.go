package ssh

import (
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

// ParseSupervisordStatus parses the output of `supervisorctl status`.
// Each line: "name   STATE   pid NNN, uptime H:MM:SS"
// Or for FATAL/STOPPED: "name   STATE   description"
func ParseSupervisordStatus(output string) []config.Process {
	var processes []config.Process
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		p := config.Process{
			Name:  fields[0],
			State: strings.ToUpper(fields[1]),
		}

		// Parse PID and uptime from remaining fields
		rest := strings.Join(fields[2:], " ")
		if strings.HasPrefix(rest, "pid ") {
			parts := strings.SplitN(rest, ",", 2)
			pidStr := strings.TrimPrefix(parts[0], "pid ")
			p.PID = strings.TrimSpace(pidStr)
			if len(parts) > 1 {
				uptime := strings.TrimSpace(parts[1])
				uptime = strings.TrimPrefix(uptime, "uptime ")
				p.Uptime = uptime
			}
		} else {
			p.PID = "-"
		}

		processes = append(processes, p)
	}
	return processes
}

// ProcessStateOrder returns the sort priority for supervisord process states.
// Lower = higher priority (shown first).
func ProcessStateOrder(state string) int {
	switch strings.ToUpper(state) {
	case "FATAL":
		return 0
	case "BACKOFF":
		return 1
	case "STOPPED":
		return 2
	case "EXITED":
		return 3
	case "STARTING":
		return 4
	case "STOPPING":
		return 5
	case "RUNNING":
		return 6
	default:
		return 7
	}
}
