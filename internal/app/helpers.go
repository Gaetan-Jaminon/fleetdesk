package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// shellQuote escapes single quotes for safe shell interpolation inside single-quoted strings.
func shellQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func (m Model) filteredServices() []config.Service {
	if m.filterText == "" {
		return m.services
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.Service
	for _, s := range m.services {
		line := strings.ToLower(s.Name + " " + s.State + " " + s.Enabled + " " + s.Description)
		if strings.Contains(line, filter) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (m Model) findServiceIndex(name string) int {
	for i, s := range m.services {
		if s.Name == name {
			return i
		}
	}
	return 0
}

// filteredErrorLogs returns the error logs matching the current filter.
func (m Model) filteredErrorLogs() []config.ErrorLog {
	if m.filterText == "" {
		return m.errorLogs
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.ErrorLog
	for _, e := range m.errorLogs {
		line := strings.ToLower(e.Time + " " + e.Unit + " " + e.Message)
		if strings.Contains(line, filter) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (m Model) filteredAccounts() []config.Account {
	if m.filterText == "" {
		return m.accounts
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.Account
	for _, a := range m.accounts {
		line := strings.ToLower(a.User + " " + a.Groups + " " + a.Shell + " " + a.PasswordStatus)
		if strings.Contains(line, filter) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (m Model) filteredPorts() []config.ListeningPort {
	if m.filterText == "" {
		return m.ports
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.ListeningPort
	for _, p := range m.ports {
		line := strings.ToLower(fmt.Sprintf("%d %s %s", p.Port, p.Process, p.BindAddress))
		if strings.Contains(line, filter) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (m Model) filteredFirewallRules() []config.FirewallRule {
	if m.filterText == "" {
		return m.firewallRules
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.FirewallRule
	for _, r := range m.firewallRules {
		line := strings.ToLower(r.Zone + " " + r.Service + " " + r.Protocol + " " + r.Action)
		if strings.Contains(line, filter) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// parseLogFields parses structured key=value pairs from a log message.
// Handles both simple key=value and key="quoted value" formats.
func parseLogFields(msg string) [][2]string {
	var pairs [][2]string
	remaining := msg

	for len(remaining) > 0 {
		remaining = strings.TrimLeft(remaining, " ")
		if remaining == "" {
			break
		}

		// find key=
		eqIdx := strings.Index(remaining, "=")
		if eqIdx < 0 {
			break
		}

		key := remaining[:eqIdx]
		// key should not contain spaces
		if strings.Contains(key, " ") {
			// not a key=value pattern -- treat the rest as plain text
			break
		}

		remaining = remaining[eqIdx+1:]

		var value string
		if len(remaining) > 0 && remaining[0] == '"' {
			// quoted value -- find closing quote
			endQuote := strings.Index(remaining[1:], "\"")
			if endQuote >= 0 {
				value = remaining[1 : endQuote+1]
				remaining = remaining[endQuote+2:]
			} else {
				value = remaining[1:]
				remaining = ""
			}
		} else {
			// unquoted value -- until next space
			spIdx := strings.Index(remaining, " ")
			if spIdx >= 0 {
				value = remaining[:spIdx]
				remaining = remaining[spIdx+1:]
			} else {
				value = remaining
				remaining = ""
			}
		}

		pairs = append(pairs, [2]string{key, value})
	}

	// only return if we found at least 2 key-value pairs (otherwise it's plain text)
	if len(pairs) >= 2 {
		return pairs
	}
	return nil
}

// applyProbeInfo updates a host with successful probe results.
func (m *Model) applyProbeInfo(idx int, info ssh.ProbeInfo) {
	m.hosts[idx].Status = config.HostOnline
	m.hosts[idx].FQDN = info.FQDN
	m.hosts[idx].OS = info.OS
	m.hosts[idx].UpSince = info.UpSince
	m.hosts[idx].ServiceCount = info.ServiceCount
	m.hosts[idx].ServiceRunning = info.ServiceRunning
	m.hosts[idx].ServiceFailed = info.ServiceFailed
	m.hosts[idx].ContainerCount = info.ContainerCount
	m.hosts[idx].ContainerRunning = info.ContainerRunning
	m.hosts[idx].CronCount = info.CronCount
	m.hosts[idx].ErrorCount = info.ErrorCount
	m.hosts[idx].UpdateCount = info.UpdateCount
	m.hosts[idx].DiskCount = info.DiskCount
	m.hosts[idx].DiskHighCount = info.DiskHighCount
	m.hosts[idx].UserCount = info.UserCount
	m.hosts[idx].LockedUsers = info.LockedUsers
	m.hosts[idx].InterfacesUp = info.InterfacesUp
	m.hosts[idx].InterfacesTotal = info.InterfacesTotal
	m.hosts[idx].ListeningPorts = info.ListeningPorts
	m.hosts[idx].LastUpdate = info.LastUpdate
	m.hosts[idx].LastSecurity = info.LastSecurity
}

// retryRemainingPasswordHosts retries connection for hosts that still need a password.
// Uses the password from the last successful retry (stored temporarily in sshManager).
func (m Model) retryRemainingPasswordHosts() tea.Cmd {
	var cmds []tea.Cmd
	for i, h := range m.hosts {
		if h.NeedsPassword && h.Status == config.HostUnreachable {
			idx := i
			hh := h
			sm := m.ssh
			cmds = append(cmds, func() tea.Msg {
				return sm.RetryWithCachedPassword(idx, hh)
			})
		}
	}
	if len(cmds) == 0 {
		// all done -- clear the cached password
		m.ssh.ClearPassword()
		return nil
	}
	return tea.Batch(cmds...)
}

// startProbe launches parallel SSH connections and probes for all hosts.
// Returns a batch of commands, one per host, that will send hostProbeResult messages.
func (m Model) startProbe() tea.Cmd {
	var cmds []tea.Cmd
	for i, h := range m.hosts {
		idx := i
		hh := h
		sm := m.ssh
		cmds = append(cmds, func() tea.Msg {
			return sm.ConnectAndProbe(idx, hh)
		})
	}
	return tea.Batch(cmds...)
}

// buildHostList creates the runtime host list from a fleet definition.
func buildHostList(f config.Fleet) []config.Host {
	errorLogSince := f.Defaults.ErrorLogSince
	var hosts []config.Host
	for _, g := range f.Groups {
		for _, e := range g.Hosts {
			hosts = append(hosts, config.Host{
				Entry:         e,
				Group:         g.Name,
				Status:        config.HostConnecting,
				ErrorLogSince: errorLogSince,
			})
		}
	}
	for _, e := range f.Hosts {
		hosts = append(hosts, config.Host{
			Entry:         e,
			Status:        config.HostConnecting,
			ErrorLogSince: errorLogSince,
		})
	}
	return hosts
}

// fleetHostCount returns the total number of hosts in a fleet.
func (m Model) fleetHostCount(f config.Fleet) int {
	count := len(f.Hosts)
	for _, g := range f.Groups {
		count += len(g.Hosts)
	}
	return count
}

// retryWithPassword attempts to connect a specific host using password auth.
func retryWithPassword(sm *ssh.Manager, idx int, h config.Host, password string) tea.Cmd {
	return func() tea.Msg {
		return sm.ConnectWithPassword(idx, h, password)
	}
}
