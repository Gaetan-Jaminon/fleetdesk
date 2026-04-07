package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// parseNumericPrefix extracts the leading integer from a string like "200G" or "90%".
func parseNumericPrefix(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

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

func (m Model) filteredServiceLogs() []string {
	if m.filterText == "" {
		return m.serviceLogLines
	}
	filter := strings.ToLower(m.filterText)
	var filtered []string
	for _, line := range m.serviceLogLines {
		if strings.Contains(strings.ToLower(line), filter) {
			filtered = append(filtered, line)
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

func (m Model) filteredFailedLogins() []config.FailedLogin {
	if m.filterText == "" {
		return m.failedLogins
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.FailedLogin
	for _, fl := range m.failedLogins {
		line := strings.ToLower(fl.Time + " " + fl.User + " " + fl.Source + " " + fl.Method)
		if strings.Contains(line, filter) {
			filtered = append(filtered, fl)
		}
	}
	return filtered
}

func (m Model) filteredSudoEntries() []config.SudoEntry {
	if m.filterText == "" {
		return m.sudoEntries
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.SudoEntry
	for _, se := range m.sudoEntries {
		line := strings.ToLower(se.Time + " " + se.User + " " + se.Command + " " + se.Result)
		if strings.Contains(line, filter) {
			filtered = append(filtered, se)
		}
	}
	return filtered
}

func (m Model) filteredSELinuxDenials() []config.SELinuxDenial {
	if m.filterText == "" {
		return m.selinuxDenials
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.SELinuxDenial
	for _, d := range m.selinuxDenials {
		line := strings.ToLower(d.Time + " " + d.Action + " " + d.Source + " " + d.Target + " " + d.Class)
		if strings.Contains(line, filter) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (m Model) filteredAuditEvents() []config.AuditEvent {
	if m.filterText == "" {
		return m.auditEvents
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.AuditEvent
	for _, ae := range m.auditEvents {
		line := strings.ToLower(ae.Time + " " + ae.Type + " " + ae.User + " " + ae.Result + " " + ae.Message)
		if strings.Contains(line, filter) {
			filtered = append(filtered, ae)
		}
	}
	return filtered
}

func (m Model) filteredContainers() []config.Container {
	if m.filterText == "" {
		return m.containers
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.Container
	for _, c := range m.containers {
		line := strings.ToLower(c.Name + " " + c.Image + " " + c.Status)
		if strings.Contains(line, filter) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (m Model) filteredUpdates() []config.Update {
	if m.filterText == "" {
		return m.updates
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.Update
	for _, u := range m.updates {
		line := strings.ToLower(u.Package + " " + u.Version + " " + u.Type)
		if strings.Contains(line, filter) {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

func (m Model) filteredCronJobs() []config.CronJob {
	if m.filterText == "" {
		return m.cronJobs
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.CronJob
	for _, cj := range m.cronJobs {
		line := strings.ToLower(cj.Schedule + " " + cj.Source + " " + cj.Command)
		if strings.Contains(line, filter) {
			filtered = append(filtered, cj)
		}
	}
	return filtered
}

func (m Model) filteredDisks() []config.Disk {
	if m.filterText == "" {
		return m.disks
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.Disk
	for _, d := range m.disks {
		line := strings.ToLower(d.Filesystem + " " + d.Mount + " " + d.UsePercent)
		if strings.Contains(line, filter) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (m Model) filteredInterfaces() []config.NetInterface {
	if m.filterText == "" {
		return m.interfaces
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.NetInterface
	for _, ni := range m.interfaces {
		line := strings.ToLower(ni.Name + " " + ni.State + " " + ni.IPs)
		if strings.Contains(line, filter) {
			filtered = append(filtered, ni)
		}
	}
	return filtered
}

func (m Model) filteredRoutes() []config.Route {
	if m.filterText == "" {
		return m.routes
	}
	filter := strings.ToLower(m.filterText)
	var filtered []config.Route
	for _, r := range m.routes {
		line := strings.ToLower(r.Destination + " " + r.Gateway + " " + r.Interface)
		if strings.Contains(line, filter) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// sortIndicator returns ▲ or ▼ if the given column is the active sort column.
func (m Model) sortIndicator(col int) string {
	if m.sortColumn != col {
		return ""
	}
	if m.sortAsc {
		return " \u25b2"
	}
	return " \u25bc"
}

// sortView applies user-selected column sort to the current view's data.
func (m *Model) sortView() {
	if m.sortColumn == 0 {
		return
	}
	switch m.view {
	case viewServiceList:
		m.sortServices()
	case viewContainerList:
		m.sortContainers()
	case viewCronList:
		m.sortCronJobs()
	case viewErrorLogList:
		m.sortErrorLogs()
	case viewUpdateList:
		m.sortUpdates()
	case viewDiskList:
		m.sortDisks()
	case viewAccountList:
		m.sortAccounts()
	case viewNetworkInterfaces:
		m.sortInterfaces()
	case viewNetworkPorts:
		m.sortPorts()
	case viewNetworkRoutes:
		m.sortRoutes()
	case viewNetworkFirewall:
		m.sortFirewallRules()
	case viewSecurityFailedLogins:
		m.sortFailedLogins()
	case viewSecuritySudo:
		m.sortSudoEntries()
	case viewSecuritySELinux:
		m.sortSELinuxDenials()
	case viewSecurityAudit:
		m.sortAuditEvents()
	}
}

func (m *Model) sortServices() {
	sort.Slice(m.services, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.services[i].Name, m.services[j].Name
		case 2: vi, vj = m.services[i].State, m.services[j].State
		case 3: vi, vj = m.services[i].Enabled, m.services[j].Enabled
		case 4: vi, vj = m.services[i].Description, m.services[j].Description
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortContainers() {
	sort.Slice(m.containers, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.containers[i].Name, m.containers[j].Name
		case 2: vi, vj = m.containers[i].Image, m.containers[j].Image
		case 3: vi, vj = m.containers[i].Status, m.containers[j].Status
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortCronJobs() {
	sort.Slice(m.cronJobs, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.cronJobs[i].Schedule, m.cronJobs[j].Schedule
		case 2: vi, vj = m.cronJobs[i].Source, m.cronJobs[j].Source
		case 3: vi, vj = m.cronJobs[i].Command, m.cronJobs[j].Command
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortErrorLogs() {
	sort.Slice(m.errorLogs, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.errorLogs[i].Time, m.errorLogs[j].Time
		case 2: vi, vj = m.errorLogs[i].Unit, m.errorLogs[j].Unit
		case 3: vi, vj = m.errorLogs[i].Message, m.errorLogs[j].Message
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortUpdates() {
	sort.Slice(m.updates, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.updates[i].Package, m.updates[j].Package
		case 2: vi, vj = m.updates[i].Version, m.updates[j].Version
		case 3: vi, vj = m.updates[i].Type, m.updates[j].Type
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortDisks() {
	sort.Slice(m.disks, func(i, j int) bool {
		switch m.sortColumn {
		case 1:
			vi, vj := m.disks[i].Filesystem, m.disks[j].Filesystem
			if m.sortAsc { return vi < vj }
			return vi > vj
		case 2:
			ni, nj := parseNumericPrefix(m.disks[i].Size), parseNumericPrefix(m.disks[j].Size)
			if m.sortAsc { return ni < nj }
			return ni > nj
		case 3:
			ni, nj := parseNumericPrefix(m.disks[i].Used), parseNumericPrefix(m.disks[j].Used)
			if m.sortAsc { return ni < nj }
			return ni > nj
		case 4:
			ni, nj := parseNumericPrefix(m.disks[i].Avail), parseNumericPrefix(m.disks[j].Avail)
			if m.sortAsc { return ni < nj }
			return ni > nj
		case 5:
			ni, nj := parseNumericPrefix(m.disks[i].UsePercent), parseNumericPrefix(m.disks[j].UsePercent)
			if m.sortAsc { return ni < nj }
			return ni > nj
		case 6:
			vi, vj := m.disks[i].Mount, m.disks[j].Mount
			if m.sortAsc { return vi < vj }
			return vi > vj
		}
		return false
	})
}

func (m *Model) sortAccounts() {
	sort.Slice(m.accounts, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.accounts[i].User, m.accounts[j].User
		case 2: vi, vj = m.accounts[i].Groups, m.accounts[j].Groups
		case 3: vi, vj = m.accounts[i].Shell, m.accounts[j].Shell
		case 4: vi, vj = m.accounts[i].LastLogin, m.accounts[j].LastLogin
		case 5: vi, vj = m.accounts[i].PasswordStatus, m.accounts[j].PasswordStatus
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortInterfaces() {
	sort.Slice(m.interfaces, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.interfaces[i].Name, m.interfaces[j].Name
		case 2: vi, vj = m.interfaces[i].State, m.interfaces[j].State
		case 3: vi, vj = m.interfaces[i].IPs, m.interfaces[j].IPs
		case 4: vi, vj = m.interfaces[i].MTU, m.interfaces[j].MTU
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortPorts() {
	sort.Slice(m.ports, func(i, j int) bool {
		switch m.sortColumn {
		case 1:
			if m.sortAsc { return m.ports[i].Port < m.ports[j].Port }
			return m.ports[i].Port > m.ports[j].Port
		case 2:
			vi, vj := m.ports[i].Protocol, m.ports[j].Protocol
			if m.sortAsc { return vi < vj }
			return vi > vj
		case 3:
			vi, vj := m.ports[i].Process, m.ports[j].Process
			if m.sortAsc { return vi < vj }
			return vi > vj
		case 4:
			vi, vj := m.ports[i].BindAddress, m.ports[j].BindAddress
			if m.sortAsc { return vi < vj }
			return vi > vj
		}
		return false
	})
}

func (m *Model) sortRoutes() {
	sort.Slice(m.routes, func(i, j int) bool {
		switch m.sortColumn {
		case 1:
			vi, vj := m.routes[i].Destination, m.routes[j].Destination
			if m.sortAsc { return vi < vj }
			return vi > vj
		case 2:
			vi, vj := m.routes[i].Gateway, m.routes[j].Gateway
			if m.sortAsc { return vi < vj }
			return vi > vj
		case 3:
			vi, vj := m.routes[i].Interface, m.routes[j].Interface
			if m.sortAsc { return vi < vj }
			return vi > vj
		case 4:
			ni, nj := parseNumericPrefix(m.routes[i].Metric), parseNumericPrefix(m.routes[j].Metric)
			if m.sortAsc { return ni < nj }
			return ni > nj
		}
		return false
	})
}

func (m *Model) sortFirewallRules() {
	sort.Slice(m.firewallRules, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.firewallRules[i].Zone, m.firewallRules[j].Zone
		case 2: vi, vj = m.firewallRules[i].Service, m.firewallRules[j].Service
		case 3: vi, vj = m.firewallRules[i].Protocol, m.firewallRules[j].Protocol
		case 4: vi, vj = m.firewallRules[i].Source, m.firewallRules[j].Source
		case 5: vi, vj = m.firewallRules[i].Action, m.firewallRules[j].Action
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortFailedLogins() {
	sort.Slice(m.failedLogins, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.failedLogins[i].Time, m.failedLogins[j].Time
		case 2: vi, vj = m.failedLogins[i].User, m.failedLogins[j].User
		case 3: vi, vj = m.failedLogins[i].Source, m.failedLogins[j].Source
		case 4: vi, vj = m.failedLogins[i].Method, m.failedLogins[j].Method
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortSudoEntries() {
	sort.Slice(m.sudoEntries, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.sudoEntries[i].Time, m.sudoEntries[j].Time
		case 2: vi, vj = m.sudoEntries[i].User, m.sudoEntries[j].User
		case 3: vi, vj = m.sudoEntries[i].Result, m.sudoEntries[j].Result
		case 4: vi, vj = m.sudoEntries[i].Command, m.sudoEntries[j].Command
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortSELinuxDenials() {
	sort.Slice(m.selinuxDenials, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.selinuxDenials[i].Time, m.selinuxDenials[j].Time
		case 2: vi, vj = m.selinuxDenials[i].Action, m.selinuxDenials[j].Action
		case 3: vi, vj = m.selinuxDenials[i].Source, m.selinuxDenials[j].Source
		case 4: vi, vj = m.selinuxDenials[i].Target, m.selinuxDenials[j].Target
		case 5: vi, vj = m.selinuxDenials[i].Class, m.selinuxDenials[j].Class
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
}

func (m *Model) sortAuditEvents() {
	sort.Slice(m.auditEvents, func(i, j int) bool {
		var vi, vj string
		switch m.sortColumn {
		case 1: vi, vj = m.auditEvents[i].Time, m.auditEvents[j].Time
		case 2: vi, vj = m.auditEvents[i].Type, m.auditEvents[j].Type
		case 3: vi, vj = m.auditEvents[i].User, m.auditEvents[j].User
		case 4: vi, vj = m.auditEvents[i].Result, m.auditEvents[j].Result
		case 5: vi, vj = m.auditEvents[i].Message, m.auditEvents[j].Message
		default: return false
		}
		if m.sortAsc { return vi < vj }
		return vi > vj
	})
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
	m.hosts[idx].FailedLoginCount = info.FailedLoginCount
	m.hosts[idx].SudoEventCount = info.SudoEventCount
	m.hosts[idx].SELinuxDenyCount = info.SELinuxDenyCount
	m.hosts[idx].AuditEventCount = info.AuditEventCount
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

// containerDetailLines builds a flat line list for the container detail view.
func (m Model) containerDetailLines() []string {
	d := m.containerDetail
	var lines []string
	lines = append(lines, "--- Details ---")
	lines = append(lines, fmt.Sprintf("  %-12s  %s", "ID", d.ID))
	lines = append(lines, fmt.Sprintf("  %-12s  %s", "Image", d.Image))
	lines = append(lines, fmt.Sprintf("  %-12s  %s", "Status", d.Status))
	lines = append(lines, fmt.Sprintf("  %-12s  %s", "Created", d.Created))
	lines = append(lines, fmt.Sprintf("  %-12s  %s", "Command", d.Command))
	if len(d.Ports) > 0 {
		lines = append(lines, "--- Ports ---")
		for _, p := range d.Ports {
			lines = append(lines, "    "+p)
		}
	}
	if len(d.Mounts) > 0 {
		lines = append(lines, "--- Mounts ---")
		for _, mt := range d.Mounts {
			lines = append(lines, "    "+mt)
		}
	}
	if len(d.Env) > 0 {
		lines = append(lines, "--- Environment ---")
		for _, e := range d.Env {
			lines = append(lines, "    "+e)
		}
	}
	return lines
}

// retryWithPassword attempts to connect a specific host using password auth.
func retryWithPassword(sm *ssh.Manager, idx int, h config.Host, password string) tea.Cmd {
	return func() tea.Msg {
		return sm.ConnectWithPassword(idx, h, password)
	}
}
