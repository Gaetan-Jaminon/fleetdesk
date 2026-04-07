package ssh

import (
	"fmt"
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

// ParseAccountLine parses a pipe-delimited account line:
// user|uid|groups|shell|last_login|pw_status|expiry
func ParseAccountLine(line string) config.Account {
	if line == "" {
		return config.Account{}
	}
	parts := strings.SplitN(line, "|", 7)
	if len(parts) < 2 {
		return config.Account{}
	}

	a := config.Account{
		User: parts[0],
	}
	fmt.Sscanf(parts[1], "%d", &a.UID)

	if len(parts) > 2 {
		a.Groups = parts[2]
	}
	if len(parts) > 3 {
		a.Shell = parts[3]
	}
	if len(parts) > 4 {
		a.LastLogin = parts[4]
	}
	if len(parts) > 5 {
		a.PasswordStatus = parts[5]
	}
	if len(parts) > 6 {
		a.Expiry = parts[6]
	}

	// detect sudo: wheel or sudo group
	groups := strings.Fields(a.Groups)
	for _, g := range groups {
		if g == "wheel" || g == "sudo" {
			a.IsSudo = true
			break
		}
	}

	// detect locked status
	a.IsLocked = a.PasswordStatus == "LK" || a.PasswordStatus == "L"

	return a
}

// AccountStateOrder returns sort priority for account states.
// Lower = shown first.
func AccountStateOrder(a config.Account) int {
	if a.IsLocked {
		return 0
	}
	return 1
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
