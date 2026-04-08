package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/azure"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// fetchServicesMsg is sent when service list fetch completes.
type fetchServicesMsg struct {
	services []config.Service
	err      error
}

// fetchContainersMsg is sent when container list fetch completes.
type fetchContainersMsg struct {
	containers []config.Container
	err        error
}

// serviceActionMsg is sent when a service action (start/stop/restart) completes.
type serviceActionMsg struct {
	action string
	unit   string
	err    error
}

// fetchServices returns a tea.Cmd that fetches systemd services from a host.
func (m Model) fetchServices() func() tea.Msg {
	idx := m.selectedHost
	h := m.hosts[idx]
	sm := m.ssh
	logger := m.logger
	mode := h.Entry.SystemdMode
	filters := h.Entry.ServiceFilter

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "services", "host_idx", idx)
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
			logger.Error("fetch failed", "view", "services", "host_idx", idx, "err", err)
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
		var services []config.Service
		for _, line := range strings.Split(strings.TrimSpace(unitsOut), "\n") {
			if line == "" {
				continue
			}
			svc := ssh.ParseServiceLine(line)
			if svc.Name != "" {
				if en, ok := enabledMap[svc.Name]; ok {
					svc.Enabled = en
				}
				if ssh.MatchesFilter(svc.Name, filters) {
					services = append(services, svc)
				}
			}
		}
		sort.Slice(services, func(i, j int) bool {
			oi, oj := ssh.ServiceStateOrder(services[i].State), ssh.ServiceStateOrder(services[j].State)
			if oi != oj {
				return oi < oj
			}
			return services[i].Name < services[j].Name
		})
		logger.Debug("fetch complete", "view", "services", "host_idx", idx, "count", len(services), "elapsed", time.Since(start))
		return fetchServicesMsg{services: services}
	}
}

// fetchContainers returns a tea.Cmd that fetches Podman containers from a host.
func (m Model) fetchContainers() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "containers", "host_idx", idx)
		cmd := `podman ps -a --format "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.ID}}" 2>/dev/null`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "containers", "host_idx", idx, "err", err)
			return fetchContainersMsg{err: fmt.Errorf("list containers: %w", err)}
		}

		var containers []config.Container
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 4)
			if len(parts) < 3 {
				continue
			}
			c := config.Container{
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
			oi, oj := ssh.ContainerStateOrder(containers[i].Status), ssh.ContainerStateOrder(containers[j].Status)
			if oi != oj {
				return oi < oj
			}
			return containers[i].Name < containers[j].Name
		})
		logger.Debug("fetch complete", "view", "containers", "host_idx", idx, "count", len(containers), "elapsed", time.Since(start))
		return fetchContainersMsg{containers: containers}
	}
}

// confirmSvcAction returns a tea.Cmd that runs a systemctl action via terminal handover with sudo.
// After the action, it shows systemctl status so the user can see the result.
func (m Model) confirmSvcAction(action string) (tea.Model, tea.Cmd) {
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

// --- Service Detail ---

type fetchServiceDetailMsg struct {
	status   config.ServiceStatus
	logLines []string
	err      error
}

// confirmDetailSvcAction triggers a service action from the detail view using the stored unit name.
func (m Model) confirmDetailSvcAction(action string) (tea.Model, tea.Cmd) {
	h := m.hosts[m.selectedHost]
	unit := m.serviceDetailUnit
	q := shellQuote(unit)

	sysctl := "sudo systemctl"
	if h.Entry.SystemdMode == "user" {
		sysctl = "systemctl --user"
	}

	m.showConfirm = true
	m.confirmMessage = fmt.Sprintf("%s %s? [Y/n]", action, unit)
	m.confirmCmd = fmt.Sprintf(
		`%s %s '%s'; rc=$?; echo ''; if [ $rc -eq 0 ]; then echo '✓ %s %s succeeded'; else echo '✗ %s %s failed (exit '$rc')'; fi; echo ''; %s status '%s' --no-pager 2>&1; true`,
		sysctl, action, q,
		action, unit,
		action, unit,
		sysctl, q,
	)
	m.confirmBanner = fmt.Sprintf("%s %s on %s", action, unit, h.Entry.Name)
	return m, nil
}

func (m Model) fetchServiceDetail(unit string) func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger
	h := m.hosts[idx]
	mode := h.Entry.SystemdMode

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "service_detail", "host_idx", idx, "unit", unit)
		sysctl := "systemctl"
		journal := "sudo journalctl -u"
		if mode == "user" {
			sysctl = "systemctl --user"
			journal = "journalctl --user-unit"
		}

		cmd := fmt.Sprintf(
			"(%s show '%s' --property=Id,Description,LoadState,ActiveState,SubState,MainPID,MemoryCurrent,TasksCurrent,ActiveEnterTimestamp,UnitFileState --no-pager; echo '---LOGS---'; %s '%s' --no-pager -n 50 -q --reverse 2>/dev/null) | cat",
			sysctl, shellQuote(unit), journal, shellQuote(unit),
		)
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "service_detail", "host_idx", idx, "unit", unit, "err", err)
			return fetchServiceDetailMsg{err: fmt.Errorf("service detail: %w", err)}
		}

		parts := strings.SplitN(out, "---LOGS---", 2)
		status := ssh.ParseServiceStatus(parts[0])

		var logLines []string
		if len(parts) > 1 {
			for _, line := range strings.Split(strings.TrimSpace(parts[1]), "\n") {
				if line != "" {
					logLines = append(logLines, line)
				}
			}
		}

		logger.Debug("fetch complete", "view", "service_detail", "host_idx", idx, "unit", unit, "log_lines", len(logLines), "elapsed", time.Since(start))
		return fetchServiceDetailMsg{status: status, logLines: logLines}
	}
}

// --- Cron Jobs ---

type fetchCronMsg struct {
	jobs []config.CronJob
	err  error
}

func (m Model) fetchCronJobs() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "cron_jobs", "host_idx", idx)
		cmd := `echo '===CRONTAB===' && crontab -l 2>/dev/null || true && echo '===CROND===' && for f in /etc/cron.d/*; do echo "FILE:$f"; cat "$f" 2>/dev/null; done`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "cron_jobs", "host_idx", idx, "err", err)
			return fetchCronMsg{err: fmt.Errorf("cron: %w", err)}
		}

		var jobs []config.CronJob
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
				jobs = append(jobs, config.CronJob{
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
				jobs = append(jobs, config.CronJob{
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

		logger.Debug("fetch complete", "view", "cron_jobs", "host_idx", idx, "count", len(jobs), "elapsed", time.Since(start))
		return fetchCronMsg{jobs: jobs}
	}
}

// --- Log Levels ---

type fetchLogLevelsMsg struct {
	levels []config.LogLevelEntry
	err    error
}

func (m Model) fetchLogLevels() func() tea.Msg {
	idx := m.selectedHost
	h := m.hosts[idx]
	sm := m.ssh
	logger := m.logger
	since := h.ErrorLogSince

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "log_levels", "host_idx", idx)
		// count entries per priority level
		cmd := fmt.Sprintf(
			`for p in 0 1 2 3 4 5 6; do echo $(sudo journalctl -p $p..$p --since '%s' --no-pager -q 2>/dev/null | wc -l); done`,
			since,
		)
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "log_levels", "host_idx", idx, "err", err)
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
		var levels []config.LogLevelEntry
		for i, n := range names {
			count := 0
			if i < len(lines) {
				fmt.Sscanf(strings.TrimSpace(lines[i]), "%d", &count)
			}
			levels = append(levels, config.LogLevelEntry{
				Level: n.level,
				Code:  n.code,
				Count: count,
			})
		}

		logger.Debug("fetch complete", "view", "log_levels", "host_idx", idx, "count", len(levels), "elapsed", time.Since(start))
		return fetchLogLevelsMsg{levels: levels}
	}
}

// --- Error Logs ---

type fetchErrorLogsMsg struct {
	logs []config.ErrorLog
	err  error
}

func (m Model) fetchErrorLogs() func() tea.Msg {
	idx := m.selectedHost
	h := m.hosts[idx]
	sm := m.ssh
	logger := m.logger
	since := h.ErrorLogSince
	level := m.selectedLogLevel

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "error_logs", "host_idx", idx, "level", level)
		cmd := fmt.Sprintf("(sudo journalctl -p %s --since '%s' --no-pager -q -o short --no-hostname --reverse 2>/dev/null | head -500) | cat", level, since)
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "error_logs", "host_idx", idx, "err", err)
			return fetchErrorLogsMsg{err: fmt.Errorf("error logs: %w", err)}
		}

		var logs []config.ErrorLog
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
			logs = append(logs, config.ErrorLog{
				Time:    timeStr,
				Unit:    unit,
				Message: msg,
			})
		}

		logger.Debug("fetch complete", "view", "error_logs", "host_idx", idx, "count", len(logs), "elapsed", time.Since(start))
		return fetchErrorLogsMsg{logs: logs}
	}
}

// --- Updates ---

type fetchUpdatesMsg struct {
	updates []config.Update
	err     error
}

func (m Model) fetchUpdates() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "updates", "host_idx", idx)
		// get pending updates and security updates in one command
		cmd := `dnf --setopt=skip_if_unavailable=1 check-update 2>&1; echo '===SECURITY==='; dnf --setopt=skip_if_unavailable=1 updateinfo list --security --quiet 2>/dev/null`
		out, err := sm.RunCommand(idx, cmd)
		// dnf check-update returns exit 100 when updates are available
		if err != nil && !strings.Contains(out, "===SECURITY===") {
			logger.Error("fetch failed", "view", "updates", "host_idx", idx, "err", err)
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
				pkg := ssh.ExtractPkgName(nvra)
				if pkg != "" {
					secPkgs[pkg] = true
				}
			}
		}

		var updates []config.Update

		// capture errors/warnings from dnf output
		for _, line := range strings.Split(strings.TrimSpace(updatesSection), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "Warning:") {
				updates = append(updates, config.Update{
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
				updates = append(updates, config.Update{
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

		logger.Debug("fetch complete", "view", "updates", "host_idx", idx, "count", len(updates), "elapsed", time.Since(start))
		return fetchUpdatesMsg{updates: updates}
	}
}

// --- Subscription ---

type fetchSubscriptionMsg struct {
	subs []config.Subscription
	err  error
}

func (m Model) fetchSubscription() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "subscription", "host_idx", idx)
		cmd := `echo '===IDENTITY===' && sudo subscription-manager identity 2>&1 && echo '===STATUS===' && sudo subscription-manager status 2>&1 && echo '===SERVER===' && sudo subscription-manager config --list 2>&1 | grep 'hostname' | head -1 && echo '===REPOS===' && dnf repolist --enabled 2>&1 && echo '===REPOCHECK===' && for repo in $(dnf repolist --enabled -q 2>/dev/null | tail -n+2 | awk '{print $1}'); do (echo "REPO:$repo:$(dnf repoinfo --disablerepo='*' --enablerepo=$repo 2>&1 | grep -c 'Error:')") & done; wait`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && !strings.Contains(out, "===IDENTITY===") {
			logger.Error("fetch failed", "view", "subscription", "host_idx", idx, "err", err)
			return fetchSubscriptionMsg{err: fmt.Errorf("subscription: %w", err)}
		}

		var subs []config.Subscription

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
		subs = append(subs, config.Subscription{Field: "Registration", Value: regType})
		if serverHost != "" {
			subs = append(subs, config.Subscription{Field: "Server", Value: serverHost})
		}

		// parse identity
		for _, line := range strings.Split(strings.TrimSpace(identitySection), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if idx := strings.Index(line, ": "); idx > 0 {
				subs = append(subs, config.Subscription{
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
				subs = append(subs, config.Subscription{
					Field: line[:idx],
					Value: line[idx+2:],
				})
			} else if strings.Contains(line, "Simple Content Access") {
				subs = append(subs, config.Subscription{
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
				subs = append(subs, config.Subscription{
					Field: "Repo: " + repoID,
					Value: status,
				})
			}
		}

		logger.Debug("fetch complete", "view", "subscription", "host_idx", idx, "count", len(subs), "elapsed", time.Since(start))
		return fetchSubscriptionMsg{subs: subs}
	}
}

// --- Accounts ---

type fetchAccountsMsg struct {
	accounts []config.Account
	err      error
}

func (m Model) fetchAccounts() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "accounts", "host_idx", idx)
		// Combine getent passwd (local users) + /home/* dirs (IPA/IDM users) for a complete list.
		// Only fetch basic info here — details (lastlog, passwd status, chage) are fetched on demand.
		cmd := `(getent passwd | awk -F: '$3 >= 1000 && $3 != 65534 {print $1}'; for d in /home/*/; do u=$(basename "$d"); getent passwd "$u" >/dev/null 2>&1 && echo "$u"; done) | sort -u | while IFS= read -r user; do
  entry=$(getent passwd "$user")
  uid=$(printf '%s' "$entry" | cut -d: -f3)
  shell=$(printf '%s' "$entry" | cut -d: -f7)
  groups=$(groups "$user" 2>/dev/null | cut -d: -f2 | xargs)
  echo "$user|$uid|$groups|$shell"
done`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "accounts", "host_idx", idx, "err", err)
			return fetchAccountsMsg{err: fmt.Errorf("accounts: %w", err)}
		}

		var accounts []config.Account
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			a := ssh.ParseAccountLine(line)
			if a.User != "" {
				accounts = append(accounts, a)
			}
		}
		sort.Slice(accounts, func(i, j int) bool {
			oi, oj := ssh.AccountStateOrder(accounts[i]), ssh.AccountStateOrder(accounts[j])
			if oi != oj {
				return oi < oj
			}
			return accounts[i].User < accounts[j].User
		})
		logger.Debug("fetch complete", "view", "accounts", "host_idx", idx, "count", len(accounts), "elapsed", time.Since(start))
		return fetchAccountsMsg{accounts: accounts}
	}
}

type accountDetailSection struct {
	title string
	items [][2]string // key-value pairs
}

type fetchAccountDetailMsg struct {
	sections []accountDetailSection
	err      error
}

func (m Model) fetchAccountDetail(user string) func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "account_detail", "host_idx", idx, "user", user)
		u := shellQuote(user)
		// Detect if IPA user (not in /etc/passwd) and adapt commands
		cmd := fmt.Sprintf(
			`echo '===IDENTITY===' && id '%s' 2>&1 && echo '===PASSWORD===' && (sudo chage -l '%s' 2>/dev/null || echo 'Managed by IPA') && echo '===LOGIN===' && (lastlog -u '%s' 2>/dev/null | tail -1 || echo 'N/A') && echo '===SUDO===' && sudo -l -U '%s' 2>/dev/null`,
			u, u, u, u,
		)
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "account_detail", "host_idx", idx, "user", user, "err", err)
			return fetchAccountDetailMsg{err: fmt.Errorf("account detail: %w", err)}
		}

		var sections []accountDetailSection

		// Parse identity section
		if part := extractSection(out, "===IDENTITY===", "===PASSWORD==="); part != "" {
			sec := accountDetailSection{title: "Identity"}
			for _, line := range strings.Split(strings.TrimSpace(part), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				sec.items = append(sec.items, [2]string{"", line})
			}
			sections = append(sections, sec)
		}

		// Parse password/chage section
		if part := extractSection(out, "===PASSWORD===", "===LOGIN==="); part != "" {
			sec := accountDetailSection{title: "Password Policy"}
			for _, line := range strings.Split(strings.TrimSpace(part), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.Contains(line, "does not exist") {
					continue
				}
				if line == "Managed by IPA" {
					sec.items = append(sec.items, [2]string{"Source", "Managed by IPA/IDM"})
					continue
				}
				if idx := strings.Index(line, ":"); idx > 0 {
					key := strings.TrimSpace(line[:idx])
					val := strings.TrimSpace(line[idx+1:])
					sec.items = append(sec.items, [2]string{key, val})
				}
			}
			if len(sec.items) > 0 {
				sections = append(sections, sec)
			}
		}

		// Parse login section
		if part := extractSection(out, "===LOGIN===", "===SUDO==="); part != "" {
			sec := accountDetailSection{title: "Last Login"}
			for _, line := range strings.Split(strings.TrimSpace(part), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "Username") {
					continue
				}
				sec.items = append(sec.items, [2]string{"", line})
			}
			if len(sec.items) > 0 {
				sections = append(sections, sec)
			}
		}

		// Parse sudo section
		if sudoIdx := strings.Index(out, "===SUDO==="); sudoIdx >= 0 {
			part := out[sudoIdx+len("===SUDO==="):]
			sec := accountDetailSection{title: "Sudo Privileges"}
			for _, line := range strings.Split(strings.TrimSpace(part), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "Matching Defaults") || strings.HasPrefix(line, "!visiblepw") {
					continue
				}
				if strings.Contains(line, "env_keep") {
					continue
				}
				sec.items = append(sec.items, [2]string{"", line})
			}
			if len(sec.items) > 0 {
				sections = append(sections, sec)
			}
		}

		logger.Debug("fetch complete", "view", "account_detail", "host_idx", idx, "user", user, "sections", len(sections), "elapsed", time.Since(start))
		return fetchAccountDetailMsg{sections: sections}
	}
}

func extractSection(out, startMarker, endMarker string) string {
	startIdx := strings.Index(out, startMarker)
	if startIdx < 0 {
		return ""
	}
	content := out[startIdx+len(startMarker):]
	endIdx := strings.Index(content, endMarker)
	if endIdx >= 0 {
		content = content[:endIdx]
	}
	return content
}

// --- Network Info ---

type fetchNetworkInfoMsg struct {
	routeCount    int
	firewallType  string
	firewallCount int
	err           error
}

func (m Model) fetchNetworkInfo() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "network_info", "host_idx", idx)
		cmd := `ip route 2>/dev/null | wc -l && if systemctl is-active firewalld >/dev/null 2>&1; then echo "firewalld"; (firewall-cmd --list-all-zones 2>/dev/null | grep -cE 'services:|ports:' || echo 0); elif command -v nft >/dev/null 2>&1 && nft list ruleset 2>/dev/null | grep -q 'chain'; then echo "nftables"; (nft list ruleset 2>/dev/null | grep -c 'rule' || echo 0); else echo "iptables"; (sudo iptables -L -n 2>/dev/null | tail -n+3 | grep -cv '^$' || echo 0); fi`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "network_info", "host_idx", idx, "err", err)
			return fetchNetworkInfoMsg{err: fmt.Errorf("network info: %w", err)}
		}

		lines := strings.Split(strings.TrimSpace(out), "\n")
		msg := fetchNetworkInfoMsg{}
		if len(lines) > 0 {
			fmt.Sscanf(lines[0], "%d", &msg.routeCount)
		}
		if len(lines) > 1 {
			msg.firewallType = strings.TrimSpace(lines[1])
		}
		if len(lines) > 2 {
			fmt.Sscanf(strings.TrimSpace(lines[2]), "%d", &msg.firewallCount)
		}
		logger.Debug("fetch complete", "view", "network_info", "host_idx", idx, "elapsed", time.Since(start))
		return msg
	}
}

// --- Network Interfaces ---

type fetchInterfacesMsg struct {
	interfaces []config.NetInterface
	err        error
}

func (m Model) fetchInterfaces() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "interfaces", "host_idx", idx)
		cmd := `ip -br addr && echo '---MTU---' && ip -o link | awk '{print $2, $5}'`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "interfaces", "host_idx", idx, "err", err)
			return fetchInterfacesMsg{err: fmt.Errorf("interfaces: %w", err)}
		}

		parts := strings.SplitN(out, "---MTU---", 2)
		addrSection := parts[0]
		mtuSection := ""
		if len(parts) > 1 {
			mtuSection = parts[1]
		}

		// parse MTU map
		mtuMap := make(map[string]string)
		for _, line := range strings.Split(strings.TrimSpace(mtuSection), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				name := strings.TrimSuffix(fields[0], ":")
				mtuMap[name] = fields[1]
			}
		}

		var interfaces []config.NetInterface
		for _, line := range strings.Split(strings.TrimSpace(addrSection), "\n") {
			if line == "" {
				continue
			}
			iface := ssh.ParseInterfaceLine(line)
			if iface.Name != "" {
				if mtu, ok := mtuMap[iface.Name]; ok {
					iface.MTU = mtu
				}
				interfaces = append(interfaces, iface)
			}
		}
		sort.Slice(interfaces, func(i, j int) bool {
			oi, oj := ssh.InterfaceStateOrder(interfaces[i].State), ssh.InterfaceStateOrder(interfaces[j].State)
			if oi != oj {
				return oi < oj
			}
			return interfaces[i].Name < interfaces[j].Name
		})
		logger.Debug("fetch complete", "view", "interfaces", "host_idx", idx, "count", len(interfaces), "elapsed", time.Since(start))
		return fetchInterfacesMsg{interfaces: interfaces}
	}
}

// --- Ports ---

type fetchPortsMsg struct {
	ports []config.ListeningPort
	err   error
}

func (m Model) fetchPorts() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "ports", "host_idx", idx)
		cmd := `ss -tlnp | tail -n +2`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "ports", "host_idx", idx, "err", err)
			return fetchPortsMsg{err: fmt.Errorf("ports: %w", err)}
		}

		var ports []config.ListeningPort
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			p := ssh.ParsePortLine(line)
			if p.Port > 0 {
				ports = append(ports, p)
			}
		}
		sort.Slice(ports, func(i, j int) bool {
			return ports[i].Port < ports[j].Port
		})
		logger.Debug("fetch complete", "view", "ports", "host_idx", idx, "count", len(ports), "elapsed", time.Since(start))
		return fetchPortsMsg{ports: ports}
	}
}

// --- Routes & DNS ---

type fetchRoutesMsg struct {
	routes      []config.Route
	nameservers []string
	search      string
	err         error
}

func (m Model) fetchRoutes() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "routes", "host_idx", idx)
		cmd := `ip route 2>/dev/null && echo '---DNS---' && cat /etc/resolv.conf 2>/dev/null | grep -E '^(nameserver|search|domain)'`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "routes", "host_idx", idx, "err", err)
			return fetchRoutesMsg{err: fmt.Errorf("routes: %w", err)}
		}

		parts := strings.SplitN(out, "---DNS---", 2)
		routeSection := parts[0]
		dnsSection := ""
		if len(parts) > 1 {
			dnsSection = parts[1]
		}

		var routes []config.Route
		for _, line := range strings.Split(strings.TrimSpace(routeSection), "\n") {
			if line == "" {
				continue
			}
			r := ssh.ParseRouteLine(line)
			if r.Destination != "" {
				routes = append(routes, r)
			}
		}
		// sort: default first, then alphabetical by destination
		sort.Slice(routes, func(i, j int) bool {
			if routes[i].IsDefault != routes[j].IsDefault {
				return routes[i].IsDefault
			}
			return routes[i].Destination < routes[j].Destination
		})

		// parse DNS
		var nameservers []string
		var search string
		for _, line := range strings.Split(strings.TrimSpace(dnsSection), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "nameserver") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					nameservers = append(nameservers, fields[1])
				}
			} else if strings.HasPrefix(line, "search") || strings.HasPrefix(line, "domain") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					search = strings.Join(fields[1:], " ")
				}
			}
		}

		logger.Debug("fetch complete", "view", "routes", "host_idx", idx, "count", len(routes), "elapsed", time.Since(start))
		return fetchRoutesMsg{
			routes:      routes,
			nameservers: nameservers,
			search:      search,
		}
	}
}

// --- Firewall ---

type fetchFirewallMsg struct {
	rules   []config.FirewallRule
	backend string
	err     error
}

func (m Model) fetchFirewall() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "firewall", "host_idx", idx)
		cmd := `if systemctl is-active firewalld >/dev/null 2>&1; then echo '---FIREWALLD---'; sudo firewall-cmd --list-all-zones 2>/dev/null; elif command -v nft >/dev/null 2>&1; then _nft=$(sudo nft list ruleset 2>/dev/null); if echo "$_nft" | grep -q 'chain'; then echo '---NFTABLES---'; echo "$_nft"; else echo '---IPTABLES---'; sudo iptables -L -n --line-numbers 2>/dev/null; fi; else echo '---IPTABLES---'; sudo iptables -L -n --line-numbers 2>/dev/null; fi`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "firewall", "host_idx", idx, "err", err)
			return fetchFirewallMsg{err: fmt.Errorf("firewall: %w", err)}
		}

		backend := ssh.DetectFirewallBackend(out)
		var rules []config.FirewallRule
		switch backend {
		case "firewalld":
			content := strings.Replace(out, "---FIREWALLD---", "", 1)
			rules = ssh.ParseFirewalldOutput(content)
		case "nftables":
			content := strings.Replace(out, "---NFTABLES---", "", 1)
			rules = ssh.ParseNftablesOutput(content)
		case "iptables":
			content := strings.Replace(out, "---IPTABLES---", "", 1)
			rules = ssh.ParseIptablesOutput(content)
		}

		logger.Debug("fetch complete", "view", "firewall", "host_idx", idx, "backend", backend, "count", len(rules), "elapsed", time.Since(start))
		return fetchFirewallMsg{rules: rules, backend: backend}
	}
}

// --- Disk ---

type fetchDiskMsg struct {
	disks []config.Disk
	err   error
}

func (m Model) fetchDisk() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "disk", "host_idx", idx)
		cmd := `df -h --output=source,size,used,avail,pcent,target -x tmpfs -x devtmpfs 2>/dev/null | tail -n+2`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "disk", "host_idx", idx, "err", err)
			return fetchDiskMsg{err: fmt.Errorf("disk: %w", err)}
		}

		var disks []config.Disk
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				disks = append(disks, config.Disk{
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

		logger.Debug("fetch complete", "view", "disk", "host_idx", idx, "count", len(disks), "elapsed", time.Since(start))
		return fetchDiskMsg{disks: disks}
	}
}

// --- Failed Logins ---

type fetchFailedLoginsMsg struct {
	logins []config.FailedLogin
	err    error
}

func (m Model) fetchFailedLogins() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "failed_logins", "host_idx", idx)
		cmd := `sudo journalctl -u sshd --no-pager -q --no-hostname -o short -n 500 2>/dev/null | grep -iE 'Failed|Invalid user' | tac; true`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "failed_logins", "host_idx", idx, "err", err)
			return fetchFailedLoginsMsg{err: fmt.Errorf("failed logins: %w", err)}
		}

		var logins []config.FailedLogin
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			fl := ssh.ParseFailedLoginLine(line)
			if fl.Source != "" || fl.User != "" {
				logins = append(logins, fl)
			}
		}
		logger.Debug("fetch complete", "view", "failed_logins", "host_idx", idx, "count", len(logins), "elapsed", time.Since(start))
		return fetchFailedLoginsMsg{logins: logins}
	}
}

// --- Sudo Activity ---

type fetchSudoActivityMsg struct {
	entries []config.SudoEntry
	err     error
}

func (m Model) fetchSudoActivity() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "sudo_activity", "host_idx", idx)
		cmd := `sudo journalctl _COMM=sudo --no-pager -q --no-hostname -o short -n 500 2>/dev/null | tac; true`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "sudo_activity", "host_idx", idx, "err", err)
			return fetchSudoActivityMsg{err: fmt.Errorf("sudo activity: %w", err)}
		}

		var entries []config.SudoEntry
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			se := ssh.ParseSudoLine(line)
			if se.User != "" {
				entries = append(entries, se)
			}
		}
		logger.Debug("fetch complete", "view", "sudo_activity", "host_idx", idx, "count", len(entries), "elapsed", time.Since(start))
		return fetchSudoActivityMsg{entries: entries}
	}
}

// --- SELinux Denials ---

type fetchSELinuxMsg struct {
	denials []config.SELinuxDenial
	err     error
}

func (m Model) fetchSELinuxDenials() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "selinux_denials", "host_idx", idx)
		cmd := `sudo journalctl _TRANSPORT=audit --no-pager -q --no-hostname -o short -n 500 2>/dev/null | grep 'avc:' | tac; true`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "selinux_denials", "host_idx", idx, "err", err)
			return fetchSELinuxMsg{err: fmt.Errorf("selinux: %w", err)}
		}

		var denials []config.SELinuxDenial
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			d := ssh.ParseSELinuxDenialLine(line)
			if d.Action != "" {
				denials = append(denials, d)
			}
		}
		logger.Debug("fetch complete", "view", "selinux_denials", "host_idx", idx, "count", len(denials), "elapsed", time.Since(start))
		return fetchSELinuxMsg{denials: denials}
	}
}

// --- Audit Summary ---

type fetchAuditSummaryMsg struct {
	events []config.AuditEvent
	err    error
}

func (m Model) fetchAuditSummary() func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "audit_summary", "host_idx", idx)
		cmd := `sudo aureport --auth -i 2>/dev/null | grep -E '^\s*[0-9]+\.' | tac; true`
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "audit_summary", "host_idx", idx, "err", err)
			return fetchAuditSummaryMsg{err: fmt.Errorf("audit: %w", err)}
		}

		var events []config.AuditEvent
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			ae := ssh.ParseAuditEventLine(strings.TrimSpace(line))
			if ae.User != "" {
				events = append(events, ae)
			}
		}
		logger.Debug("fetch complete", "view", "audit_summary", "host_idx", idx, "count", len(events), "elapsed", time.Since(start))
		return fetchAuditSummaryMsg{events: events}
	}
}

// --- Update Detail ---

type fetchUpdateDetailMsg struct {
	lines []string
	err   error
}

func (m Model) fetchUpdateDetail(pkg string) func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "update_detail", "host_idx", idx, "pkg", pkg)
		cmd := fmt.Sprintf("dnf info '%s' 2>/dev/null", shellQuote(pkg))
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "update_detail", "host_idx", idx, "pkg", pkg, "err", err)
			return fetchUpdateDetailMsg{err: fmt.Errorf("info %s: %w", pkg, err)}
		}
		var lines []string
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			lines = append(lines, line)
		}
		logger.Debug("fetch complete", "view", "update_detail", "host_idx", idx, "pkg", pkg, "lines", len(lines), "elapsed", time.Since(start))
		return fetchUpdateDetailMsg{lines: lines}
	}
}

// --- Disk Detail ---

type fetchDiskDetailMsg struct {
	lines []string
	err   error
}

func (m Model) fetchDiskDetail(mount string) func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "disk_detail", "host_idx", idx, "mount", mount)
		cmd := fmt.Sprintf("df -h '%s' 2>/dev/null && echo '---' && (sudo tune2fs -l $(findmnt -n -o SOURCE '%s') 2>/dev/null || lsblk -f $(findmnt -n -o SOURCE '%s') 2>/dev/null || echo 'No additional details available')", shellQuote(mount), shellQuote(mount), shellQuote(mount))
		out, err := sm.RunCommand(idx, cmd)
		if err != nil && out == "" {
			logger.Error("fetch failed", "view", "disk_detail", "host_idx", idx, "mount", mount, "err", err)
			return fetchDiskDetailMsg{err: fmt.Errorf("disk detail %s: %w", mount, err)}
		}
		var lines []string
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			lines = append(lines, line)
		}
		logger.Debug("fetch complete", "view", "disk_detail", "host_idx", idx, "mount", mount, "lines", len(lines), "elapsed", time.Since(start))
		return fetchDiskDetailMsg{lines: lines}
	}
}

// --- Container Detail ---

type fetchContainerDetailMsg struct {
	detail config.ContainerDetail
	err    error
}

func (m Model) fetchContainerDetail(name string) func() tea.Msg {
	idx := m.selectedHost
	sm := m.ssh
	logger := m.logger

	return func() tea.Msg {
		start := time.Now()
		logger.Debug("fetch start", "view", "container_detail", "host_idx", idx, "name", name)
		cmd := fmt.Sprintf("podman inspect '%s' 2>/dev/null", shellQuote(name))
		out, err := sm.RunCommand(idx, cmd)
		if err != nil {
			logger.Error("fetch failed", "view", "container_detail", "host_idx", idx, "name", name, "err", err)
			return fetchContainerDetailMsg{err: fmt.Errorf("inspect %s: %w", name, err)}
		}
		detail := ssh.ParseContainerInspect(out)
		logger.Debug("fetch complete", "view", "container_detail", "host_idx", idx, "name", name, "elapsed", time.Since(start))
		return fetchContainerDetailMsg{detail: detail}
	}
}

// --- Metrics ---

type fetchMetricsMsg struct {
	index   int
	metrics config.HostMetrics
	err     error
}

func (m Model) fetchAllMetrics() tea.Cmd {
	logger := m.logger
	var cmds []tea.Cmd
	for i, h := range m.hosts {
		if h.Status != config.HostOnline {
			continue
		}
		idx := i
		sm := m.ssh
		cmds = append(cmds, func() tea.Msg {
			start := time.Now()
			logger.Debug("fetch start", "view", "metrics", "host_idx", idx)
			cmd := `top -bn1 -d0 2>/dev/null | grep '^%Cpu' | awk '{printf "%.0f\n", 100-$8}' && free 2>/dev/null | awk '/Mem/{printf "%.0f\n", $3*100/$2}' && df -h / 2>/dev/null | tail -1 | awk '{print $5}' && awk '{print $1}' /proc/loadavg 2>/dev/null && (uptime -s 2>/dev/null || echo unknown)`
			out, err := sm.RunCommand(idx, cmd)
			if err != nil {
				logger.Error("fetch failed", "view", "metrics", "host_idx", idx, "err", err)
				return fetchMetricsMsg{index: idx, err: err}
			}
			logger.Debug("fetch complete", "view", "metrics", "host_idx", idx, "elapsed", time.Since(start))
			return fetchMetricsMsg{index: idx, metrics: ssh.ParseMetricsOutput(out)}
		})
	}
	return tea.Batch(cmds...)
}

// fetchAzureResourceCounts fetches VM, RG, and AKS counts concurrently.
func (m Model) fetchAzureResourceCounts() tea.Cmd {
	am := m.azure
	sub := m.azureSubs[m.selectedAzureSub]
	logger := m.logger
	return func() tea.Msg {
		counts, errs := azure.FetchResourceCounts(am, sub.Name, sub.ID, sub.TenantID, logger)
		return azureResourceCountsMsg{counts: counts, errs: errs}
	}
}

// fetchAzureVMs fetches the VM list for the selected Azure subscription.
func (m Model) fetchAzureVMs() tea.Cmd {
	am := m.azure
	sub := m.azureSubs[m.selectedAzureSub]
	logger := m.logger
	return func() tea.Msg {
		vms, err := azure.FetchVMs(am, sub.Name, sub.ID, sub.TenantID, logger)
		return fetchAzureVMsMsg{vms: vms, err: err}
	}
}

// fetchAzureVMDetail fetches extended properties for a specific VM.
func (m Model) fetchAzureVMDetail(vmName, rgName string) tea.Cmd {
	am := m.azure
	sub := m.azureSubs[m.selectedAzureSub]
	logger := m.logger
	return func() tea.Msg {
		detail, err := azure.FetchVMDetail(am, vmName, rgName, sub.Name, sub.TenantID, logger)
		return fetchAzureVMDetailMsg{detail: detail, err: err}
	}
}

// executeAzureVMAction executes a VM power action (start, deallocate, restart).
func (m Model) executeAzureVMAction(vmName, rgName, action string) tea.Cmd {
	am := m.azure
	sub := m.azureSubs[m.selectedAzureSub]
	logger := m.logger
	return func() tea.Msg {
		err := azure.VMAction(am, vmName, rgName, sub.Name, sub.TenantID, action, logger)
		return azureVMActionMsg{action: action, vm: vmName, err: err}
	}
}

// fetchAzureActivityLog fetches recent activity log for a resource group.
func (m Model) fetchAzureActivityLog(rgName string) tea.Cmd {
	am := m.azure
	sub := m.azureSubs[m.selectedAzureSub]
	logger := m.logger
	return func() tea.Msg {
		entries, err := azure.FetchActivityLog(am, rgName, sub.Name, sub.TenantID, logger)
		return fetchAzureActivityLogMsg{entries: entries, err: err}
	}
}
