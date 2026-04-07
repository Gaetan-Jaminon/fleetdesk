package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// Bridge functions to internal/ssh.
var (
	serviceStateOrder   = issh.ServiceStateOrder
	containerStateOrder = issh.ContainerStateOrder
	matchesFilter       = issh.MatchesFilter
	extractPkgName      = issh.ExtractPkgName
)

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
		out, err := sm.RunCommand(idx, cmd)
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

var parseServiceLine = issh.ParseServiceLine

// fetchContainers returns a tea.Cmd that fetches Podman containers from a host.
func (m model) fetchContainers() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh

	return func() tea.Msg {
		cmd := `podman ps -a --format "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.ID}}" 2>/dev/null`
		out, err := sm.RunCommand(idx, cmd)
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


// svcAction returns a tea.Cmd that runs a systemctl action via terminal handover with sudo.
// After the action, it shows systemctl status so the user can see the result.
func (m model) confirmSvcAction(action string) (tea.Model, tea.Cmd) {
	h := m.hosts[m.selectedHost]
	unit := m.services[m.serviceCursor].Name + ".service"

	sysctl := "sudo systemctl"
	statusctl := "sudo systemctl"
	if h.Entry.SystemdMode == "user" {
		sysctl = "systemctl --user"
		statusctl = "systemctl --user"
	}

	m.showConfirm = true
	m.confirmMessage = fmt.Sprintf("%s %s? [Y/n]", action, unit)
	m.confirmCmd = fmt.Sprintf(
		`%s %s %s; rc=$?; echo ''; if [ $rc -eq 0 ]; then echo '✓ %s %s succeeded'; else echo '✗ %s %s failed (exit '$rc')'; fi; echo ''; %s status %s --no-pager 2>&1; true`,
		sysctl, action, unit,
		action, unit,
		action, unit,
		statusctl, unit,
	)
	m.confirmBanner = fmt.Sprintf("%s %s on %s", action, unit, h.Entry.Name)
	return m, nil
}

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
		out, err := sm.RunCommand(idx, cmd)
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
		out, err := sm.RunCommand(idx, cmd)
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
		out, err := sm.RunCommand(idx, cmd)
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
		cmd := `dnf --setopt=skip_if_unavailable=1 check-update 2>&1; echo '===SECURITY==='; dnf --setopt=skip_if_unavailable=1 updateinfo list --security --quiet 2>/dev/null`
		out, err := sm.RunCommand(idx, cmd)
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
				// format: "RHSA-2026:6277 Important/Sec. ansible-core-1:2.16.17-1.el9ap.noarch"
				nvra := fields[len(fields)-1]
				// strip .arch
				if idx := strings.LastIndex(nvra, "."); idx > 0 {
					nvra = nvra[:idx]
				}
				// extract package name: everything before the version
				// NVR format: name-[epoch:]version-release
				// Find the last two dashes that precede a digit
				pkg := extractPkgName(nvra)
				if pkg != "" {
					secPkgs[pkg] = true
				}
			}
		}

		var updates []update

		// capture errors/warnings from dnf output
		for _, line := range strings.Split(strings.TrimSpace(updatesSection), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "Warning:") {
				updates = append(updates, update{
					Package: line,
					Version: "",
					Type:    "error",
				})
			}
		}

		for _, line := range strings.Split(strings.TrimSpace(updatesSection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Last metadata") || strings.HasPrefix(line, "Obsoleting") || strings.HasPrefix(line, "Is this ok") || strings.HasPrefix(line, "Not root") || strings.HasPrefix(line, "Microsoft") || strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "Warning:") || strings.HasPrefix(line, "Importing") || strings.HasPrefix(line, "Userid") || strings.HasPrefix(line, "Fingerprint") || strings.HasPrefix(line, "From") {
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

		// sort: errors first, then security, then bugfix, then alphabetically
		typeOrder := map[string]int{"error": 0, "security": 1, "bugfix": 2}
		sort.Slice(updates, func(i, j int) bool {
			oi := typeOrder[updates[i].Type]
			oj := typeOrder[updates[j].Type]
			if oi != oj {
				return oi < oj
			}
			return updates[i].Package < updates[j].Package
		})

		return fetchUpdatesMsg{updates: updates}
	}
}

// --- Subscription ---

type fetchSubscriptionMsg struct {
	subs []subscription
	err  error
}

func (m model) fetchSubscription() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh

	return func() tea.Msg {
		cmd := `echo '===IDENTITY===' && sudo subscription-manager identity 2>&1 && echo '===STATUS===' && sudo subscription-manager status 2>&1 && echo '===SERVER===' && sudo subscription-manager config --list 2>&1 | grep 'hostname' | head -1 && echo '===REPOS===' && dnf repolist --enabled 2>&1 && echo '===REPOCHECK===' && for repo in $(dnf repolist --enabled -q 2>/dev/null | tail -n+2 | awk '{print $1}'); do (echo "REPO:$repo:$(dnf repoinfo --disablerepo='*' --enablerepo=$repo 2>&1 | grep -c 'Error:')") & done; wait`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && !strings.Contains(out, "===IDENTITY===") {
			return fetchSubscriptionMsg{err: fmt.Errorf("subscription: %w", err)}
		}

		var subs []subscription

		parts := strings.SplitN(out, "===STATUS===", 2)
		identitySection := strings.Replace(parts[0], "===IDENTITY===", "", 1)
		statusSection := ""
		serverSection := ""
		repoSection := ""
		if len(parts) > 1 {
			parts2 := strings.SplitN(parts[1], "===SERVER===", 2)
			statusSection = parts2[0]
			if len(parts2) > 1 {
				parts3 := strings.SplitN(parts2[1], "===REPOS===", 2)
				serverSection = parts3[0]
				if len(parts3) > 1 {
					repoSection = parts3[1]
				}
			}
		}

		// detect registration type from server hostname
		regType := "Unknown"
		serverHost := ""
		for _, line := range strings.Split(serverSection, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "hostname") {
				if idx := strings.Index(line, "["); idx > 0 {
					serverHost = strings.Trim(line[idx:], "[]")
				}
			}
		}
		if strings.Contains(serverHost, "rhsm.redhat.com") {
			regType = "Red Hat CDN"
		} else if strings.Contains(serverHost, "satellite") || strings.Contains(serverHost, "katello") {
			regType = "Satellite"
		} else if serverHost != "" {
			regType = "Custom (" + serverHost + ")"
		}
		subs = append(subs, subscription{Field: "Registration", Value: regType})
		if serverHost != "" {
			subs = append(subs, subscription{Field: "Server", Value: serverHost})
		}

		// parse identity
		for _, line := range strings.Split(strings.TrimSpace(identitySection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if idx := strings.Index(line, ": "); idx > 0 {
				subs = append(subs, subscription{
					Field: line[:idx],
					Value: line[idx+2:],
				})
			}
		}

		// parse status
		for _, line := range strings.Split(strings.TrimSpace(statusSection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "+--") || strings.HasPrefix(line, "System Status") {
				continue
			}
			if idx := strings.Index(line, ": "); idx > 0 {
				subs = append(subs, subscription{
					Field: line[:idx],
					Value: line[idx+2:],
				})
			} else if strings.Contains(line, "Simple Content Access") {
				subs = append(subs, subscription{
					Field: "Content Access",
					Value: "Simple Content Access",
				})
			}
		}

		// parse repos and status
		parts3 := strings.SplitN(repoSection, "===REPOCHECK===", 2)
		repoListSection := parts3[0]
		repoCheckSection := ""
		if len(parts3) > 1 {
			repoCheckSection = parts3[1]
		}

		// build repo error map
		repoErrors := make(map[string]bool)
		for _, line := range strings.Split(strings.TrimSpace(repoCheckSection), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "REPO:") {
				parts := strings.SplitN(line, ":", 3)
				if len(parts) == 3 && parts[2] != "0" {
					repoErrors[parts[1]] = true
				}
			}
		}

		// list enabled repos with status
		for _, line := range strings.Split(strings.TrimSpace(repoListSection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "repo id") || strings.HasPrefix(line, "Not root") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				repoID := fields[0]
				status := "OK"
				if repoErrors[repoID] {
					status = "ERROR"
				}
				subs = append(subs, subscription{
					Field: "Repo: " + repoID,
					Value: status,
				})
			}
		}

		return fetchSubscriptionMsg{subs: subs}
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
		out, err := sm.RunCommand(idx, cmd)
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
			var ii, jj int
			fmt.Sscanf(pi, "%d", &ii)
			fmt.Sscanf(pj, "%d", &jj)
			return ii > jj
		})

		return fetchDiskMsg{disks: disks}
	}
}
