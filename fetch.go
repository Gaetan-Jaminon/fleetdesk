package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// serviceStateOrder returns a sort priority for service states.
// Lower = shown first.
func serviceStateOrder(state string) int {
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

// containerStateOrder returns a sort priority for container states.
func containerStateOrder(status string) int {
	if strings.HasPrefix(status, "Up") {
		return 0
	}
	if strings.HasPrefix(status, "Exited") {
		return 1
	}
	return 2
}

// fetchServicesMsg is sent when service list fetch completes.
type fetchServicesMsg struct {
	services []service
	err      error
}

// fetchContainersMsg is sent when container list fetch completes.
type fetchContainersMsg struct {
	containers []container
	err        error
}

// serviceActionMsg is sent when a service action (start/stop/restart) completes.
type serviceActionMsg struct {
	action string
	unit   string
	err    error
}

// fetchServices returns a tea.Cmd that fetches systemd services from a host.
func (m model) fetchServices() func() tea.Msg {
	idx := m.selectedHost
	h := m.hosts[idx]
	sm := m.ssh
	mode := h.Entry.SystemdMode
	filters := h.Entry.ServiceFilter

	return func() tea.Msg {
		sysctl := "systemctl"
		if mode == "user" {
			sysctl = "systemctl --user"
		}

		// fetch units and unit-files in one SSH command
		cmd := fmt.Sprintf(
			"%s list-units --type=service --all --no-pager --plain --no-legend && echo '---SEPARATOR---' && %s list-unit-files --type=service --no-pager --plain --no-legend",
			sysctl, sysctl,
		)
		out, err := sm.runCommand(idx, cmd)
		if err != nil {
			return fetchServicesMsg{err: fmt.Errorf("list services: %w", err)}
		}

		// split output into units and unit-files sections
		parts := strings.SplitN(out, "---SEPARATOR---", 2)
		unitsOut := parts[0]
		var unitFilesOut string
		if len(parts) > 1 {
			unitFilesOut = parts[1]
		}

		// parse enabled status from unit-files
		enabledMap := make(map[string]string)
		for _, line := range strings.Split(strings.TrimSpace(unitFilesOut), "\n") {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				name := strings.TrimSuffix(fields[0], ".service")
				enabledMap[name] = fields[1]
			}
		}

		// parse services from list-units
		var services []service
		for _, line := range strings.Split(strings.TrimSpace(unitsOut), "\n") {
			if line == "" {
				continue
			}
			svc := parseServiceLine(line)
			if svc.Name != "" {
				if en, ok := enabledMap[svc.Name]; ok {
					svc.Enabled = en
				}
				if matchesFilter(svc.Name, filters) {
					services = append(services, svc)
				}
			}
		}
		sort.Slice(services, func(i, j int) bool {
			oi, oj := serviceStateOrder(services[i].State), serviceStateOrder(services[j].State)
			if oi != oj {
				return oi < oj
			}
			return services[i].Name < services[j].Name
		})
		return fetchServicesMsg{services: services}
	}
}

// parseServiceLine parses a single line from systemctl list-units output.
// Format: UNIT LOAD ACTIVE SUB DESCRIPTION...
func parseServiceLine(line string) service {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return service{}
	}

	name := fields[0]
	// strip .service suffix for cleaner display
	name = strings.TrimSuffix(name, ".service")

	state := fields[2] // active, inactive, failed
	sub := fields[3]   // running, dead, exited, waiting, etc.

	// use sub-state for more detail when relevant
	display := state
	if state == "active" && sub != "" {
		display = sub
	}

	// description is everything after the 4th field
	desc := ""
	if len(fields) > 4 {
		desc = strings.Join(fields[4:], " ")
	}

	return service{
		Name:        name,
		State:       display,
		Enabled:     "—",
		Description: desc,
	}
}

// fetchContainers returns a tea.Cmd that fetches Podman containers from a host.
func (m model) fetchContainers() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh

	return func() tea.Msg {
		cmd := `podman ps -a --format "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.ID}}" 2>/dev/null`
		out, err := sm.runCommand(idx, cmd)
		if err != nil {
			return fetchContainersMsg{err: fmt.Errorf("list containers: %w", err)}
		}

		var containers []container
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 4)
			if len(parts) < 3 {
				continue
			}
			c := container{
				Name:   parts[0],
				Image:  parts[1],
				Status: parts[2],
			}
			if len(parts) >= 4 {
				c.ID = parts[3]
			}
			containers = append(containers, c)
		}
		sort.Slice(containers, func(i, j int) bool {
			oi, oj := containerStateOrder(containers[i].Status), containerStateOrder(containers[j].Status)
			if oi != oj {
				return oi < oj
			}
			return containers[i].Name < containers[j].Name
		})
		return fetchContainersMsg{containers: containers}
	}
}

// matchesFilter returns true if the service name matches any of the filter patterns.
// If no filters are defined, all services match.
func matchesFilter(name string, filters []string) bool {
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

// svcAction returns a tea.Cmd that runs a systemctl action via terminal handover with sudo.
// After the action, it shows systemctl status so the user can see the result.
func (m model) svcAction(action string) tea.Cmd {
	h := m.hosts[m.selectedHost]
	unit := m.services[m.serviceCursor].Name + ".service"

	sysctl := "sudo systemctl"
	statusctl := "sudo systemctl"
	if h.Entry.SystemdMode == "user" {
		sysctl = "systemctl --user"
		statusctl = "systemctl --user"
	}

	cmd := fmt.Sprintf(
		`%s %s %s; rc=$?; echo ''; if [ $rc -eq 0 ]; then echo '✓ %s %s succeeded'; else echo '✗ %s %s failed (exit '$rc')'; fi; echo ''; %s status %s --no-pager 2>&1; true`,
		sysctl, action, unit,
		action, unit,
		action, unit,
		statusctl, unit,
	)
	banner := fmt.Sprintf("%s %s on %s", action, unit, h.Entry.Name)
	return sshHandover(h, []string{cmd}, banner)
}

// --- Cron Jobs ---

type fetchCronMsg struct {
	jobs []cronJob
	err  error
}

func (m model) fetchCronJobs() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh

	return func() tea.Msg {
		cmd := `echo '===CRONTAB===' && crontab -l 2>/dev/null || true && echo '===CROND===' && for f in /etc/cron.d/*; do echo "FILE:$f"; cat "$f" 2>/dev/null; done`
		out, err := sm.runCommand(idx, cmd)
		if err != nil {
			return fetchCronMsg{err: fmt.Errorf("cron: %w", err)}
		}

		var jobs []cronJob
		parts := strings.SplitN(out, "===CROND===", 2)
		crontabSection := ""
		crondSection := ""
		if len(parts) >= 1 {
			crontabSection = strings.Replace(parts[0], "===CRONTAB===", "", 1)
		}
		if len(parts) >= 2 {
			crondSection = parts[1]
		}

		// parse user crontab
		for _, line := range strings.Split(strings.TrimSpace(crontabSection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "no crontab") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				jobs = append(jobs, cronJob{
					Schedule: strings.Join(fields[:5], " "),
					Command:  strings.Join(fields[5:], " "),
					Source:   "crontab",
				})
			}
		}

		// parse /etc/cron.d/ files
		currentFile := ""
		for _, line := range strings.Split(strings.TrimSpace(crondSection), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "FILE:") {
				currentFile = filepath.Base(strings.TrimPrefix(line, "FILE:"))
				continue
			}
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "SHELL") || strings.HasPrefix(line, "PATH") || strings.HasPrefix(line, "MAILTO") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 7 { // 5 schedule + user + command
				jobs = append(jobs, cronJob{
					Schedule: strings.Join(fields[:5], " "),
					Command:  strings.Join(fields[6:], " "),
					Source:   currentFile,
				})
			}
		}

		// sort: user crontab first, then system
		sort.Slice(jobs, func(i, j int) bool {
			iUser := jobs[i].Source == "crontab"
			jUser := jobs[j].Source == "crontab"
			if iUser != jUser {
				return iUser
			}
			return jobs[i].Source < jobs[j].Source
		})

		return fetchCronMsg{jobs: jobs}
	}
}

// --- Log Levels ---

type fetchLogLevelsMsg struct {
	levels []logLevelEntry
	err    error
}

func (m model) fetchLogLevels() func() tea.Msg {
	idx := m.selectedHost
	h := m.hosts[idx]
	sm := m.ssh
	since := h.ErrorLogSince

	return func() tea.Msg {
		// count entries per priority level
		cmd := fmt.Sprintf(
			`for p in 0 1 2 3 4 5 6; do echo $(sudo journalctl -p $p..$p --since '%s' --no-pager -q 2>/dev/null | wc -l); done`,
			since,
		)
		out, err := sm.runCommand(idx, cmd)
		if err != nil {
			return fetchLogLevelsMsg{err: fmt.Errorf("log levels: %w", err)}
		}

		names := []struct {
			level string
			code  string
		}{
			{"Emergency", "0"},
			{"Alert", "1"},
			{"Critical", "2"},
			{"Error", "3"},
			{"Warning", "4"},
			{"Notice", "5"},
			{"Info", "6"},
		}

		lines := strings.Split(strings.TrimSpace(out), "\n")
		var levels []logLevelEntry
		for i, n := range names {
			count := 0
			if i < len(lines) {
				fmt.Sscanf(strings.TrimSpace(lines[i]), "%d", &count)
			}
			levels = append(levels, logLevelEntry{
				Level: n.level,
				Code:  n.code,
				Count: count,
			})
		}

		return fetchLogLevelsMsg{levels: levels}
	}
}

// --- Error Logs ---

type fetchErrorLogsMsg struct {
	logs []errorLog
	err  error
}

func (m model) fetchErrorLogs() func() tea.Msg {
	idx := m.selectedHost
	h := m.hosts[idx]
	sm := m.ssh
	since := h.ErrorLogSince
	level := m.selectedLogLevel

	return func() tea.Msg {
		cmd := fmt.Sprintf("sudo journalctl -p %s --since '%s' --no-pager -q -o short --no-hostname 2>/dev/null | tail -500", level, since)
		out, err := sm.runCommand(idx, cmd)
		if err != nil {
			return fetchErrorLogsMsg{err: fmt.Errorf("error logs: %w", err)}
		}

		var logs []errorLog
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			// format: "Apr 06 14:32:01 unit[pid]: message"
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			timeStr := strings.Join(fields[:3], " ")
			unit := strings.TrimSuffix(fields[3], ":")
			msg := strings.Join(fields[4:], " ")
			logs = append(logs, errorLog{
				Time:    timeStr,
				Unit:    unit,
				Message: msg,
			})
		}

		return fetchErrorLogsMsg{logs: logs}
	}
}

// --- Updates ---

type fetchUpdatesMsg struct {
	updates []update
	err     error
}

func (m model) fetchUpdates() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh

	return func() tea.Msg {
		// get pending updates and security updates in one command
		cmd := `dnf check-update --quiet 2>/dev/null; echo '===SECURITY==='; dnf updateinfo list --security --quiet 2>/dev/null`
		out, err := sm.runCommand(idx, cmd)
		// dnf check-update returns exit 100 when updates are available
		if err != nil && !strings.Contains(out, "===SECURITY===") {
			return fetchUpdatesMsg{err: fmt.Errorf("updates: %w", err)}
		}

		parts := strings.SplitN(out, "===SECURITY===", 2)
		updatesSection := parts[0]
		securitySection := ""
		if len(parts) > 1 {
			securitySection = parts[1]
		}

		// build security package set
		secPkgs := make(map[string]bool)
		for _, line := range strings.Split(strings.TrimSpace(securitySection), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// format: "RHSA-xxx:xx Important/Critical pkg.arch version"
				pkg := fields[len(fields)-2]
				// strip arch
				if idx := strings.LastIndex(pkg, "."); idx > 0 {
					pkg = pkg[:idx]
				}
				secPkgs[pkg] = true
			}
		}

		var updates []update
		for _, line := range strings.Split(strings.TrimSpace(updatesSection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Last metadata") || strings.HasPrefix(line, "Obsoleting") || strings.HasPrefix(line, "Is this ok") || strings.HasPrefix(line, "Not root") || strings.HasPrefix(line, "Microsoft") {
				continue
			}
			fields := strings.Fields(line)
			// package lines have format: name.arch  version  repo
			if len(fields) < 2 || !strings.Contains(fields[0], ".") {
				continue
			}
			if len(fields) >= 2 {
				pkg := fields[0]
				ver := fields[1]
				// strip arch from package name
				if idx := strings.LastIndex(pkg, "."); idx > 0 {
					pkg = pkg[:idx]
				}
				typ := "bugfix"
				if secPkgs[pkg] {
					typ = "security"
				}
				updates = append(updates, update{
					Package: pkg,
					Version: ver,
					Type:    typ,
				})
			}
		}

		// sort: security first, then alphabetically
		sort.Slice(updates, func(i, j int) bool {
			if updates[i].Type != updates[j].Type {
				if updates[i].Type == "security" {
					return true
				}
				if updates[j].Type == "security" {
					return false
				}
			}
			return updates[i].Package < updates[j].Package
		})

		return fetchUpdatesMsg{updates: updates}
	}
}

// --- Disk ---

type fetchDiskMsg struct {
	disks []disk
	err   error
}

func (m model) fetchDisk() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh

	return func() tea.Msg {
		cmd := `df -h --output=source,size,used,avail,pcent,target -x tmpfs -x devtmpfs 2>/dev/null | tail -n+2`
		out, err := sm.runCommand(idx, cmd)
		if err != nil {
			return fetchDiskMsg{err: fmt.Errorf("disk: %w", err)}
		}

		var disks []disk
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				disks = append(disks, disk{
					Filesystem: fields[0],
					Size:       fields[1],
					Used:       fields[2],
					Avail:      fields[3],
					UsePercent: fields[4],
					Mount:      fields[5],
				})
			}
		}

		// sort by use% descending
		sort.Slice(disks, func(i, j int) bool {
			pi := strings.TrimSuffix(disks[i].UsePercent, "%")
			pj := strings.TrimSuffix(disks[j].UsePercent, "%")
			vi, _ := fmt.Sscanf(pi, "%d", new(int))
			vj, _ := fmt.Sscanf(pj, "%d", new(int))
			_ = vi
			_ = vj
			var ii, jj int
			fmt.Sscanf(pi, "%d", &ii)
			fmt.Sscanf(pj, "%d", &jj)
			return ii > jj
		})

		return fetchDiskMsg{disks: disks}
	}
}
