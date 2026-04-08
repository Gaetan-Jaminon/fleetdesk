package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/azure"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/k8s"
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
	case viewMetrics:
		m.sortMetricsIdx()
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
	case viewAzureSubList:
		m.sortAzureSubs()
	case viewAzureVMList:
		m.sortAzureVMs()
	case viewAzureAKSList:
		m.sortAzureAKS()
	case viewK8sClusterList:
		m.sortK8sClusters()
	case viewK8sNodeList:
		m.sortK8sNodes()
	case viewK8sNodeDetail:
		m.sortK8sNodePods()
	case viewK8sNamespaceList:
		m.sortK8sNamespaces()
	case viewK8sWorkloadList:
		m.sortK8sWorkloads()
	case viewK8sPodList:
		m.sortK8sPodList()
	}
}

func (m *Model) sortMetricsIdx() {
	// build index slice
	m.metricsSortedIdx = make([]int, len(m.hosts))
	for i := range m.hosts {
		m.metricsSortedIdx[i] = i
	}
	if m.sortColumn == 0 {
		return
	}
	sort.Slice(m.metricsSortedIdx, func(a, b int) bool {
		ia, ib := m.metricsSortedIdx[a], m.metricsSortedIdx[b]
		ma, mb := m.metrics[ia], m.metrics[ib]
		var less bool
		switch m.sortColumn {
		case 1: // HOST
			less = m.hosts[ia].Entry.Name < m.hosts[ib].Entry.Name
		case 2: // CPU%
			less = ma.CPUPercent < mb.CPUPercent
		case 3: // MEM%
			less = ma.MemPercent < mb.MemPercent
		case 4: // DISK%
			less = ma.DiskPercent < mb.DiskPercent
		case 5: // LOAD
			var la, lb float64
			fmt.Sscanf(ma.Load, "%f", &la)
			fmt.Sscanf(mb.Load, "%f", &lb)
			less = la < lb
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
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
			m.logger.Debug("retryRemainingPasswordHosts", "host_idx", idx, "host", hh.Entry.Name)
			cmds = append(cmds, func() tea.Msg {
				return sm.RetryWithCachedPassword(idx, hh)
			})
		}
	}
	if len(cmds) == 0 {
		m.logger.Debug("retryRemainingPasswordHosts done, no hosts remaining")
		// all done -- clear the cached password
		m.ssh.ClearPassword()
		return nil
	}
	m.logger.Debug("retryRemainingPasswordHosts", "retry_count", len(cmds))
	return tea.Batch(cmds...)
}

// startProbe launches parallel SSH connections and probes for all hosts.
// Returns a batch of commands, one per host, that will send hostProbeResult messages.
func (m Model) startProbe() tea.Cmd {
	m.logger.Debug("startProbe", "host_count", len(m.hosts))
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

// buildAzureSubList creates the runtime subscription list from an Azure fleet definition.
// Groups in the fleet config represent Azure subscriptions.
func buildAzureSubList(f config.Fleet) []azure.AzureSubscriptionItem {
	var subs []azure.AzureSubscriptionItem
	for _, g := range f.Groups {
		subs = append(subs, azure.AzureSubscriptionItem{
			Name:     g.Name,
			TenantID: f.TenantID,
			Status:   azure.SubConnecting,
		})
	}
	return subs
}

// startAzureProbe launches parallel access checks for all subscriptions.
func (m Model) startAzureProbe() tea.Cmd {
	m.logger.Debug("startAzureProbe", "sub_count", len(m.azureSubs))
	var cmds []tea.Cmd
	for i, sub := range m.azureSubs {
		idx := i
		ss := sub
		am := m.azure
		logger := m.logger
		cmds = append(cmds, func() tea.Msg {
			info, err := azure.CheckSubscriptionAccess(am, ss.Name, ss.TenantID, logger)
			return azure.SubscriptionProbeResult{Index: idx, Info: info, Err: err}
		})
	}
	return tea.Batch(cmds...)
}

// applyAzureProbeInfo updates a subscription item with successful access check results.
func (m *Model) applyAzureProbeInfo(idx int, info azure.SubscriptionProbeInfo) {
	m.azureSubs[idx].Status = azure.SubOnline
	m.azureSubs[idx].ID = info.ID
	m.azureSubs[idx].State = info.State
	m.azureSubs[idx].Tenant = info.Tenant
	m.azureSubs[idx].User = info.User
}

// isAzureTransitioningState returns true for Azure VM power states that indicate
// an in-progress transition (not a final state like running, deallocated, stopped).
func isAzureTransitioningState(state string) bool {
	switch state {
	case "starting", "stopping", "deallocating", "restarting":
		return true
	}
	return false
}

// filteredAzureVMs returns VMs matching the current filter.
func (m Model) filteredAzureVMs() []azure.VM {
	if m.azureVMs == nil {
		return nil
	}
	if m.filterText == "" {
		return m.azureVMs
	}
	filter := strings.ToLower(m.filterText)
	var filtered []azure.VM
	for _, vm := range m.azureVMs {
		line := strings.ToLower(vm.Name + " " + vm.ResourceGroup + " " + vm.PowerState + " " + vm.VMSize + " " + vm.OSType + " " + vm.OSDisk + " " + vm.PrivateIP + " " + vm.Hostname)
		if strings.Contains(line, filter) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// sortAzureVMs sorts the VM list by the active sort column.
func (m *Model) sortAzureVMs() {
	sort.Slice(m.azureVMs, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1:
			less = strings.ToLower(m.azureVMs[i].Name) < strings.ToLower(m.azureVMs[j].Name)
		case 2:
			less = strings.ToLower(m.azureVMs[i].ResourceGroup) < strings.ToLower(m.azureVMs[j].ResourceGroup)
		case 3:
			less = m.azureVMs[i].PowerState < m.azureVMs[j].PowerState
		case 4:
			less = m.azureVMs[i].VMSize < m.azureVMs[j].VMSize
		case 5:
			less = m.azureVMs[i].OSDisk < m.azureVMs[j].OSDisk
		case 6:
			less = m.azureVMs[i].PrivateIP < m.azureVMs[j].PrivateIP
		case 7:
			less = strings.ToLower(m.azureVMs[i].Hostname) < strings.ToLower(m.azureVMs[j].Hostname)
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// filteredAzureAKS returns AKS clusters matching the current filter.
func (m Model) filteredAzureAKS() []azure.AKSDetail {
	if m.azureAKSClusters == nil {
		return nil
	}
	if m.filterText == "" {
		return m.azureAKSClusters
	}
	filter := strings.ToLower(m.filterText)
	var filtered []azure.AKSDetail
	for _, c := range m.azureAKSClusters {
		line := strings.ToLower(c.Name + " " + c.ResourceGroup + " " + c.PowerState + " " + c.KubernetesVersion)
		if strings.Contains(line, filter) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// sortAzureAKS sorts the AKS cluster list by the active sort column.
func (m *Model) sortAzureAKS() {
	sort.Slice(m.azureAKSClusters, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1:
			less = strings.ToLower(m.azureAKSClusters[i].Name) < strings.ToLower(m.azureAKSClusters[j].Name)
		case 2:
			less = strings.ToLower(m.azureAKSClusters[i].ResourceGroup) < strings.ToLower(m.azureAKSClusters[j].ResourceGroup)
		case 3:
			less = m.azureAKSClusters[i].PowerState < m.azureAKSClusters[j].PowerState
		case 4:
			less = m.azureAKSClusters[i].KubernetesVersion < m.azureAKSClusters[j].KubernetesVersion
		case 5:
			less = m.azureAKSClusters[i].NodeCount < m.azureAKSClusters[j].NodeCount
		case 6:
			less = len(m.azureAKSClusters[i].Pools) < len(m.azureAKSClusters[j].Pools)
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// buildK8sClusterList creates the runtime cluster list from a Kubernetes fleet.
func buildK8sClusterList(f config.Fleet) []k8s.K8sClusterItem {
	var clusters []k8s.K8sClusterItem
	for _, g := range f.Groups {
		clusters = append(clusters, k8s.K8sClusterItem{
			Name:   g.Name,
			Status: k8s.ClusterChecking,
		})
	}
	return clusters
}

// startK8sProbe launches parallel connectivity checks for all clusters.
func (m Model) startK8sProbe() tea.Cmd {
	m.logger.Debug("startK8sProbe", "cluster_count", len(m.k8sClusters))
	var cmds []tea.Cmd
	for i, c := range m.k8sClusters {
		idx := i
		name := c.Name
		km := m.k8s
		logger := m.logger
		cmds = append(cmds, func() tea.Msg {
			ctxCount := k8s.CountContexts(km, name)
			if ctxCount == 0 {
				return k8sClusterProbeMsg{index: idx, err: fmt.Errorf("no kubectl context found")}
			}
			version, err := k8s.CheckCluster(km, name, logger)
			return k8sClusterProbeMsg{index: idx, contextCount: ctxCount, k8sVersion: version, err: err}
		})
	}
	return tea.Batch(cmds...)
}

// sortK8sClusters sorts the cluster list by the active sort column.
func (m *Model) sortK8sClusters() {
	sort.Slice(m.k8sClusters, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1:
			less = strings.ToLower(m.k8sClusters[i].Name) < strings.ToLower(m.k8sClusters[j].Name)
		case 2:
			less = m.k8sClusters[i].Status < m.k8sClusters[j].Status
		case 3:
			less = m.k8sClusters[i].ContextCount < m.k8sClusters[j].ContextCount
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// sortAzureSubs sorts the subscription list by the active sort column.
func (m *Model) sortAzureSubs() {
	sort.Slice(m.azureSubs, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1:
			less = strings.ToLower(m.azureSubs[i].Name) < strings.ToLower(m.azureSubs[j].Name)
		case 2:
			less = strings.ToLower(m.azureSubs[i].Tenant) < strings.ToLower(m.azureSubs[j].Tenant)
		case 3:
			less = strings.ToLower(m.azureSubs[i].User) < strings.ToLower(m.azureSubs[j].User)
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// filteredK8sNodes returns nodes matching the current filter.
func (m Model) filteredK8sNodes() []k8s.K8sNode {
	if m.k8sNodes == nil {
		return nil
	}
	if m.filterText == "" {
		return m.k8sNodes
	}
	filter := strings.ToLower(m.filterText)
	var filtered []k8s.K8sNode
	for _, n := range m.k8sNodes {
		line := strings.ToLower(n.Name + " " + n.Pool + " " + n.Status + " " + n.Version + " " + n.VMSize)
		if strings.Contains(line, filter) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// sortK8sNodes sorts the node list by the active sort column.
func (m *Model) sortK8sNodes() {
	sort.Slice(m.k8sNodes, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1:
			less = m.k8sNodes[i].Name < m.k8sNodes[j].Name
		case 2:
			less = m.k8sNodes[i].Status < m.k8sNodes[j].Status
		case 3:
			less = m.k8sNodes[i].Version < m.k8sNodes[j].Version
		case 4:
			less = m.k8sNodes[i].Taints < m.k8sNodes[j].Taints
		case 5:
			less = m.k8sNodes[i].CPUUsage < m.k8sNodes[j].CPUUsage
		case 6:
			less = m.k8sNodes[i].CPUPct < m.k8sNodes[j].CPUPct
		case 7:
			less = m.k8sNodes[i].MemUsage < m.k8sNodes[j].MemUsage
		case 8:
			less = m.k8sNodes[i].MemPct < m.k8sNodes[j].MemPct
		case 9:
			less = m.k8sNodes[i].CPUA < m.k8sNodes[j].CPUA
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// sortK8sNodePods sorts the pod list by the active sort column.
func (m *Model) sortK8sNodePods() {
	sort.Slice(m.k8sNodePods, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1:
			less = m.k8sNodePods[i].Namespace < m.k8sNodePods[j].Namespace
		case 2:
			less = m.k8sNodePods[i].Name < m.k8sNodePods[j].Name
		case 3:
			less = m.k8sNodePods[i].Status < m.k8sNodePods[j].Status
		case 4:
			less = m.k8sNodePods[i].Ready < m.k8sNodePods[j].Ready
		case 5:
			less = m.k8sNodePods[i].CPUReq < m.k8sNodePods[j].CPUReq
		case 6:
			less = m.k8sNodePods[i].CPULim < m.k8sNodePods[j].CPULim
		case 7:
			less = m.k8sNodePods[i].MemReq < m.k8sNodePods[j].MemReq
		case 8:
			less = m.k8sNodePods[i].MemLim < m.k8sNodePods[j].MemLim
		case 9:
			less = m.k8sNodePods[i].Age < m.k8sNodePods[j].Age
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// filteredK8sNodePods returns node pods matching the current filter.
func (m Model) filteredK8sNodePods() []k8s.K8sNodePod {
	if m.k8sNodePods == nil {
		return nil
	}
	if m.filterText == "" {
		return m.k8sNodePods
	}
	filter := strings.ToLower(m.filterText)
	var filtered []k8s.K8sNodePod
	for _, p := range m.k8sNodePods {
		line := strings.ToLower(p.Namespace + " " + p.Name + " " + p.Status)
		if strings.Contains(line, filter) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filteredK8sContexts returns contexts matching the current filter.
func (m Model) filteredK8sContexts() []k8s.K8sContext {
	if m.k8sContexts == nil {
		return nil
	}
	if m.filterText == "" {
		return m.k8sContexts
	}
	filter := strings.ToLower(m.filterText)
	var filtered []k8s.K8sContext
	for _, ctx := range m.k8sContexts {
		line := strings.ToLower(ctx.Name + " " + ctx.User + " " + ctx.Namespace)
		if strings.Contains(line, filter) {
			filtered = append(filtered, ctx)
		}
	}
	return filtered
}

// filteredK8sNamespaces returns namespaces matching the current filter.
func (m Model) filteredK8sNamespaces() []k8s.K8sNamespace {
	if m.k8sNamespaces == nil { return nil }
	if m.filterText == "" { return m.k8sNamespaces }
	filter := strings.ToLower(m.filterText)
	var filtered []k8s.K8sNamespace
	for _, ns := range m.k8sNamespaces {
		if strings.Contains(strings.ToLower(ns.Name+" "+ns.Status), filter) { filtered = append(filtered, ns) }
	}
	return filtered
}

func (m *Model) sortK8sNamespaces() {
	sort.Slice(m.k8sNamespaces, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1: less = m.k8sNamespaces[i].Name < m.k8sNamespaces[j].Name
		case 2: less = m.k8sNamespaces[i].Status < m.k8sNamespaces[j].Status
		case 3: less = m.k8sNamespaces[i].PodCount < m.k8sNamespaces[j].PodCount
		case 4: less = m.k8sNamespaces[i].DeployCount < m.k8sNamespaces[j].DeployCount
		case 5: less = m.k8sNamespaces[i].STSCount < m.k8sNamespaces[j].STSCount
		case 6: less = m.k8sNamespaces[i].DSCount < m.k8sNamespaces[j].DSCount
		case 7: less = m.k8sNamespaces[i].Age < m.k8sNamespaces[j].Age
		default: return false
		}
		if m.sortAsc { return less }
		return !less
	})
}

// filteredK8sWorkloads returns workloads matching the current filter.
func (m Model) filteredK8sWorkloads() []k8s.K8sWorkload {
	if m.k8sWorkloads == nil { return nil }
	if m.filterText == "" { return m.k8sWorkloads }
	filter := strings.ToLower(m.filterText)
	var filtered []k8s.K8sWorkload
	for _, w := range m.k8sWorkloads {
		if strings.Contains(strings.ToLower(w.Kind+" "+w.Name+" "+w.Ready), filter) { filtered = append(filtered, w) }
	}
	return filtered
}

func (m *Model) sortK8sWorkloads() {
	sort.Slice(m.k8sWorkloads, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1: less = m.k8sWorkloads[i].Name < m.k8sWorkloads[j].Name
		case 2: less = m.k8sWorkloads[i].Ready < m.k8sWorkloads[j].Ready
		case 3: less = m.k8sWorkloads[i].UpToDate < m.k8sWorkloads[j].UpToDate
		case 4: less = m.k8sWorkloads[i].Available < m.k8sWorkloads[j].Available
		case 5: less = m.k8sWorkloads[i].Age < m.k8sWorkloads[j].Age
		default: return false
		}
		if m.sortAsc { return less }
		return !less
	})
}

// filteredK8sPodList returns pods matching the current filter.
func (m Model) filteredK8sPodList() []k8s.K8sPod {
	if m.k8sPodList == nil { return nil }
	if m.filterText == "" { return m.k8sPodList }
	filter := strings.ToLower(m.filterText)
	var filtered []k8s.K8sPod
	for _, p := range m.k8sPodList {
		line := strings.ToLower(p.Name + " " + p.Status + " " + p.Node)
		if strings.Contains(line, filter) { filtered = append(filtered, p) }
	}
	return filtered
}

func (m *Model) sortK8sPodList() {
	sort.Slice(m.k8sPodList, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case 1: less = m.k8sPodList[i].Name < m.k8sPodList[j].Name
		case 2: less = m.k8sPodList[i].Status < m.k8sPodList[j].Status
		case 3: less = m.k8sPodList[i].Ready < m.k8sPodList[j].Ready
		case 4: less = m.k8sPodList[i].Restarts < m.k8sPodList[j].Restarts
		case 5: less = m.k8sPodList[i].Node < m.k8sPodList[j].Node
		case 6: less = m.k8sPodList[i].Age < m.k8sPodList[j].Age
		default: return false
		}
		if m.sortAsc { return less }
		return !less
	})
}
