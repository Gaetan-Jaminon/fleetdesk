package ssh

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

// ServiceStateOrder returns a sort priority for service states.
// Lower = shown first.
func ServiceStateOrder(state string) int {
	switch state {
	case "failed":
		return 0
	case "running":
		return 1
	case "exited":
		return 2
	case "waiting":
		return 3
	case "inactive":
		return 4
	default:
		return 5
	}
}

// ContainerStateOrder returns a sort priority for container states.
func ContainerStateOrder(status string) int {
	if strings.HasPrefix(status, "Up") {
		return 0
	}
	if strings.HasPrefix(status, "Exited") {
		return 1
	}
	return 2
}

// ParseServiceLine parses a single line from systemctl list-units output.
// Format: UNIT LOAD ACTIVE SUB DESCRIPTION...
func ParseServiceLine(line string) config.Service {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return config.Service{}
	}

	name := strings.TrimSuffix(fields[0], ".service")
	state := fields[2]
	sub := fields[3]

	display := state
	if state == "active" && sub != "" {
		display = sub
	}

	desc := ""
	if len(fields) > 4 {
		desc = strings.Join(fields[4:], " ")
	}

	return config.Service{
		Name:        name,
		State:       display,
		Enabled:     "—",
		Description: desc,
	}
}

// MatchesFilter returns true if the service name matches any of the filter patterns.
// If no filters are defined, all services match.
func MatchesFilter(name string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, pattern := range filters {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// ExtractPkgName extracts the package name from an NVRA string.
func ExtractPkgName(nvra string) string {
	if idx := strings.LastIndex(nvra, "."); idx > 0 {
		tail := nvra[idx+1:]
		if tail == "x86_64" || tail == "noarch" || tail == "i686" || tail == "aarch64" || tail == "src" {
			nvra = nvra[:idx]
		}
	}
	lastDash := strings.LastIndex(nvra, "-")
	if lastDash <= 0 {
		return nvra
	}
	secondDash := strings.LastIndex(nvra[:lastDash], "-")
	if secondDash <= 0 {
		return nvra
	}
	return nvra[:secondDash]
}

// IsAuthError checks if an error is an SSH authentication failure.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unable to authenticate") ||
		strings.Contains(s, "no supported methods remain") ||
		strings.Contains(s, "handshake failed")
}

// ExpandPath expands ~ in paths.
func ExpandPath(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
