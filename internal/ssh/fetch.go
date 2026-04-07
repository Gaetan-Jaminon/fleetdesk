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
		// filter out the primary group (same name as user)
		var filtered []string
		for _, g := range strings.Fields(parts[2]) {
			if g != a.User {
				filtered = append(filtered, g)
			}
		}
		a.Groups = strings.Join(filtered, " ")
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

// ParseInterfaceLine parses a line from `ip -br addr` output.
// Format: INTERFACE STATE [IP/PREFIX ...]
func ParseInterfaceLine(line string) config.NetInterface {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return config.NetInterface{}
	}
	iface := config.NetInterface{
		Name:  fields[0],
		State: fields[1],
	}
	// strip CIDR prefix from IPs
	var ips []string
	for _, f := range fields[2:] {
		if idx := strings.Index(f, "/"); idx > 0 {
			ips = append(ips, f[:idx])
		} else {
			ips = append(ips, f)
		}
	}
	iface.IPs = strings.Join(ips, " ")
	return iface
}

// InterfaceStateOrder returns sort priority for interface states.
// Lower = shown first.
func InterfaceStateOrder(state string) int {
	switch state {
	case "UP":
		return 0
	case "UNKNOWN":
		return 1
	case "DOWN":
		return 2
	default:
		return 3
	}
}

// ParsePortLine parses a line from `ss -tlnp` output.
// Format: State Recv-Q Send-Q Local_Address:Port Peer_Address:Port [Process]
func ParsePortLine(line string) config.ListeningPort {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return config.ListeningPort{}
	}

	local := fields[3] // e.g. "0.0.0.0:22", "[::]:80", "127.0.0.1:5432"
	var bindAddr string
	var port int

	if strings.HasPrefix(local, "[") {
		// IPv6: [::]:80 or [::1]:9090
		bracket := strings.LastIndex(local, "]:")
		if bracket > 0 {
			bindAddr = local[1:bracket]
			fmt.Sscanf(local[bracket+2:], "%d", &port)
		}
	} else {
		// IPv4: 0.0.0.0:22
		lastColon := strings.LastIndex(local, ":")
		if lastColon > 0 {
			bindAddr = local[:lastColon]
			fmt.Sscanf(local[lastColon+1:], "%d", &port)
		}
	}

	// extract process name from users:(("name",pid=X,fd=Y))
	process := "—"
	for _, f := range fields[5:] {
		if strings.HasPrefix(f, "users:") {
			if start := strings.Index(f, "((\""); start >= 0 {
				end := strings.Index(f[start+3:], "\"")
				if end > 0 {
					process = f[start+3 : start+3+end]
				}
			}
			break
		}
	}

	return config.ListeningPort{
		Port:        port,
		Protocol:    "tcp",
		Process:     process,
		BindAddress: bindAddr,
	}
}

// ParseRouteLine parses a line from `ip route` output.
func ParseRouteLine(line string) config.Route {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return config.Route{}
	}

	r := config.Route{
		Destination: fields[0],
		Gateway:     "direct",
		Metric:      "—",
		IsDefault:   fields[0] == "default",
	}

	for i := 1; i < len(fields)-1; i++ {
		switch fields[i] {
		case "via":
			r.Gateway = fields[i+1]
		case "dev":
			r.Interface = fields[i+1]
		case "metric":
			r.Metric = fields[i+1]
		}
	}

	return r
}

// DetectFirewallBackend returns the firewall backend from probe output markers.
func DetectFirewallBackend(output string) string {
	if strings.Contains(output, "---FIREWALLD---") {
		return "firewalld"
	}
	if strings.Contains(output, "---NFTABLES---") {
		return "nftables"
	}
	if strings.Contains(output, "---IPTABLES---") {
		return "iptables"
	}
	return ""
}

// ParseFirewalldOutput parses `firewall-cmd --list-all-zones` output into rules.
// Skips zones with no services and no ports.
func ParseFirewalldOutput(output string) []config.FirewallRule {
	var rules []config.FirewallRule
	var zoneName string
	var services, ports []string

	flush := func() {
		if zoneName == "" {
			return
		}
		for _, svc := range services {
			rules = append(rules, config.FirewallRule{
				Zone: zoneName, Service: svc, Protocol: "—", Source: "—",
				Action: "allow", Backend: "firewalld",
			})
		}
		for _, port := range ports {
			rules = append(rules, config.FirewallRule{
				Zone: zoneName, Service: port, Protocol: "—", Source: "—",
				Action: "allow", Backend: "firewalld",
			})
		}
		zoneName = ""
		services = nil
		ports = nil
	}

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		// zone header: starts at column 0 (not indented)
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			flush()
			// strip "(active)" suffix
			name := strings.TrimSpace(strings.Replace(trimmed, "(active)", "", 1))
			if name != "" {
				zoneName = name
			}
			continue
		}

		if strings.HasPrefix(trimmed, "services:") {
			svcLine := strings.TrimPrefix(trimmed, "services:")
			for _, s := range strings.Fields(svcLine) {
				services = append(services, s)
			}
		} else if strings.HasPrefix(trimmed, "ports:") {
			portLine := strings.TrimPrefix(trimmed, "ports:")
			for _, p := range strings.Fields(portLine) {
				ports = append(ports, p)
			}
		}
	}
	flush()

	return rules
}

// ParseIptablesOutput parses `iptables -L -n --line-numbers` output into rules.
func ParseIptablesOutput(output string) []config.FirewallRule {
	var rules []config.FirewallRule
	var chain string

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// chain header: "Chain INPUT (policy ACCEPT)"
		if strings.HasPrefix(trimmed, "Chain ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				chain = parts[1]
			}
			continue
		}

		// skip column header line
		if strings.HasPrefix(trimmed, "num") || strings.HasPrefix(trimmed, "target") {
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) < 4 {
			continue
		}

		// fields: num target prot opt source destination [extras]
		action := fields[1]
		protocol := fields[2]
		source := fields[4]
		if source == "0.0.0.0/0" {
			source = "—"
		}

		// extract dpt:PORT if present
		service := "—"
		for _, f := range fields[6:] {
			if strings.HasPrefix(f, "dpt:") {
				service = strings.TrimPrefix(f, "dpt:")
				break
			}
		}

		rules = append(rules, config.FirewallRule{
			Zone:     chain,
			Service:  service,
			Protocol: protocol,
			Source:   source,
			Action:   action,
			Backend:  "iptables",
		})
	}

	return rules
}

// ParseNftablesOutput provides basic parsing of `nft list ruleset` output.
func ParseNftablesOutput(output string) []config.FirewallRule {
	var rules []config.FirewallRule
	var chain string

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "chain ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				chain = parts[1]
			}
			continue
		}

		// rule lines contain keywords like "accept", "drop", "reject"
		if chain != "" && (strings.Contains(trimmed, "accept") || strings.Contains(trimmed, "drop") || strings.Contains(trimmed, "reject")) {
			action := "—"
			if strings.Contains(trimmed, "accept") {
				action = "accept"
			} else if strings.Contains(trimmed, "drop") {
				action = "drop"
			} else if strings.Contains(trimmed, "reject") {
				action = "reject"
			}

			rules = append(rules, config.FirewallRule{
				Zone:    chain,
				Service: trimmed,
				Action:  action,
				Backend: "nftables",
			})
		}
	}

	return rules
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
