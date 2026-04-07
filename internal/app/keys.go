package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// log detail mode -- any key goes back
	if m.showLogDetail {
		m.showLogDetail = false
		return m, nil
	}

	// filter mode -- capture search input
	if m.filterActive {
		switch msg.Type {
		case tea.KeyEnter:
			m.filterActive = false
			m.serviceCursor = 0
			m.errorCursor = 0
			m.accountCursor = 0
		case tea.KeyEsc:
			m.filterActive = false
			m.filterText = ""
			m.serviceCursor = 0
			m.errorCursor = 0
			m.accountCursor = 0
		case tea.KeyBackspace:
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.serviceCursor = 0
				m.errorCursor = 0
				m.accountCursor = 0
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.filterText += string(msg.Runes)
				m.serviceCursor = 0
				m.errorCursor = 0
				m.accountCursor = 0
			}
		}
		return m, nil
	}

	// confirmation prompt mode
	if m.showConfirm {
		switch msg.String() {
		case "y", "Y":
			// fall through
		case "n", "N", "esc":
			m.showConfirm = false
			m.confirmCmd = ""
			m.confirmBanner = ""
			m.flash = "Cancelled"
			return m, nil
		default:
			if msg.Type == tea.KeyEnter {
				// Enter = default yes, fall through
			} else {
				return m, nil
			}
		}
		// confirmed -- execute
		m.showConfirm = false
		h := m.hosts[m.selectedHost]
		cmd := m.confirmCmd
		banner := m.confirmBanner
		m.confirmCmd = ""
		m.confirmBanner = ""
		m.flash = ""
		return m, sshHandover(h, []string{cmd}, banner)
	}

	// password prompt mode -- capture input
	if m.showPasswordPrompt {
		switch msg.Type {
		case tea.KeyEnter:
			m.showPasswordPrompt = false
			password := m.passwordInput
			m.passwordInput = "" // clear input immediately
			idx := m.passwordHostIdx
			m.hosts[idx].Status = config.HostConnecting
			m.flash = ""
			// cache password temporarily in ssh manager for batch retries
			m.ssh.SetCachedPassword(password)
			return m, retryWithPassword(m.ssh, idx, m.hosts[idx], password)
		case tea.KeyEsc:
			m.showPasswordPrompt = false
			m.hosts[m.passwordHostIdx].Status = config.HostUnreachable
			m.hosts[m.passwordHostIdx].Error = "password prompt cancelled"
			m.flash = ""
			return m, nil
		case tea.KeyBackspace:
			if len(m.passwordInput) > 0 {
				m.passwordInput = m.passwordInput[:len(m.passwordInput)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.passwordInput += string(msg.Runes)
			}
		}
		return m, nil
	}

	m.flash = ""
	m.flashError = false

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	switch m.view {
	case viewFleetPicker:
		return m.handleFleetPickerKeys(msg)
	case viewHostList:
		return m.handleHostListKeys(msg)
	case viewResourcePicker:
		return m.handleResourcePickerKeys(msg)
	case viewServiceList:
		return m.handleServiceListKeys(msg)
	case viewContainerList:
		return m.handleContainerListKeys(msg)
	case viewCronList:
		return m.handleCronListKeys(msg)
	case viewLogLevelPicker:
		return m.handleLogLevelPickerKeys(msg)
	case viewErrorLogList:
		return m.handleErrorLogListKeys(msg)
	case viewUpdateList:
		return m.handleUpdateListKeys(msg)
	case viewDiskList:
		return m.handleDiskListKeys(msg)
	case viewAccountList:
		return m.handleAccountListKeys(msg)
	case viewSubscription:
		return m.handleSubscriptionKeys(msg)
	}
	return m, nil
}

func (m Model) handleFleetPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.fleetCursor > 0 {
			m.fleetCursor--
		}
	case "down", "j":
		if m.fleetCursor < len(m.fleets)-1 {
			m.fleetCursor++
		}
	case "e":
		if len(m.fleets) > 0 {
			return m, m.editFleetFile()
		}
	case "r":
		fleets, err := config.ScanFleets()
		if err != nil {
			m.flash = fmt.Sprintf("Reload failed: %v", err)
			m.flashError = true
			return m, nil
		}
		m.fleets = fleets
		if m.fleetCursor >= len(m.fleets) {
			m.fleetCursor = max(0, len(m.fleets)-1)
		}
		m.flash = "Reloaded"
	case "enter":
		if len(m.fleets) > 0 {
			f := m.fleets[m.fleetCursor]
			if f.Type != "vm" {
				m.flash = "K8s support coming soon"
				m.flashError = false
				return m, nil
			}
			m.selectedFleet = m.fleetCursor
			m.ssh.Close()
			m.hosts = buildHostList(f)
			m.hostCursor = 0
			m.view = viewHostList
			return m, tea.Batch(m.startProbe(), m.tickCmd())
		}
	}
	return m, nil
}

func (m Model) handleHostListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.hostCursor > 0 {
			m.hostCursor--
		}
	case "down", "j":
		if m.hostCursor < len(m.hosts)-1 {
			m.hostCursor++
		}
	case "x":
		if len(m.hosts) > 0 {
			h := m.hosts[m.hostCursor]
			if h.Status != config.HostOnline {
				m.flash = "Host is not reachable"
				m.flashError = true
				return m, nil
			}
			return m, sshHandover(h, []string{}, fmt.Sprintf("shell %s@%s", h.Entry.User, h.Entry.Name))
		}
	case "R":
		if len(m.hosts) > 0 {
			h := m.hosts[m.hostCursor]
			if h.Status != config.HostOnline {
				m.flash = "Host is not reachable"
				m.flashError = true
				return m, nil
			}
			m.selectedHost = m.hostCursor
			m.showConfirm = true
			m.confirmMessage = fmt.Sprintf("REBOOT %s? [Y/n]", h.Entry.Name)
			m.confirmCmd = `sudo reboot; echo 'Reboot initiated'`
			m.confirmBanner = fmt.Sprintf("reboot %s", h.Entry.Name)
		}
	case "r":
		f := m.fleets[m.selectedFleet]
		m.ssh.Close()
		m.hosts = buildHostList(f)
		m.flash = "Refreshing..."
		return m, m.startProbe()
	case "enter":
		if len(m.hosts) > 0 {
			h := m.hosts[m.hostCursor]
			if h.Status != config.HostOnline {
				m.flash = "Host is not reachable"
				m.flashError = true
				return m, nil
			}
			m.selectedHost = m.hostCursor
			m.resourceCursor = 0
			m.services = nil
			m.containers = nil
			m.view = viewResourcePicker
			// pre-fetch for accurate counts
			return m, tea.Batch(m.fetchServices(), m.fetchContainers(), m.fetchUpdates())
		}
	case "esc":
		m.ssh.Close()
		m.view = viewFleetPicker
	}
	return m, nil
}

func (m Model) handleResourcePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.resourceCursor > 0 {
			m.resourceCursor--
		}
	case "down", "j":
		if m.resourceCursor < resourceCount-1 {
			m.resourceCursor++
		}
	case "enter":
		switch m.resourceCursor {
		case 0: // Services
			m.serviceCursor = 0
			m.services = nil
			m.view = viewServiceList
			return m, m.fetchServices()
		case 1: // Containers
			m.containerCursor = 0
			m.containers = nil
			m.view = viewContainerList
			return m, m.fetchContainers()
		case 2: // Cron Jobs
			m.cronCursor = 0
			m.cronJobs = nil
			m.view = viewCronList
			return m, m.fetchCronJobs()
		case 3: // Error Logs -> Log Level Picker
			m.logLevelCursor = 0
			m.logLevels = nil
			m.view = viewLogLevelPicker
			return m, m.fetchLogLevels()
		case 4: // Updates
			m.updateCursor = 0
			m.updates = nil
			m.view = viewUpdateList
			return m, m.fetchUpdates()
		case 5: // Disk
			m.diskCursor = 0
			m.disks = nil
			m.view = viewDiskList
			return m, m.fetchDisk()
		case 6: // Subscription
			m.subscriptionCursor = 0
			m.subscriptions = nil
			m.view = viewSubscription
			return m, m.fetchSubscription()
		case 7: // Accounts
			m.accountCursor = 0
			m.accounts = nil
			m.view = viewAccountList
			return m, m.fetchAccounts()
		case 8: // Network
			m.flash = "Network view coming in v0.5.0"
		}
	case "esc":
		m.view = viewHostList
	}
	return m, nil
}

func (m Model) handleServiceListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredServices()

	switch msg.String() {
	case "up", "k":
		if m.serviceCursor > 0 {
			m.serviceCursor--
		}
	case "down", "j":
		if m.serviceCursor < len(filtered)-1 {
			m.serviceCursor++
		}
	case "/":
		m.filterActive = true
		m.filterText = ""
		m.serviceCursor = 0
	case "s":
		if len(filtered) > 0 && m.serviceCursor < len(filtered) {
			// find the original index for the action
			m.serviceCursor = m.findServiceIndex(filtered[m.serviceCursor].Name)
			return m.confirmSvcAction("start")
		}
	case "o":
		if len(filtered) > 0 && m.serviceCursor < len(filtered) {
			m.serviceCursor = m.findServiceIndex(filtered[m.serviceCursor].Name)
			return m.confirmSvcAction("stop")
		}
	case "t":
		if len(filtered) > 0 && m.serviceCursor < len(filtered) {
			m.serviceCursor = m.findServiceIndex(filtered[m.serviceCursor].Name)
			return m.confirmSvcAction("restart")
		}
	case "l":
		if len(m.services) > 0 {
			h := m.hosts[m.selectedHost]
			unit := m.services[m.serviceCursor].Name + ".service"
			jctl := "sudo journalctl -u"
			if h.Entry.SystemdMode == "user" {
				jctl = "journalctl --user-unit"
			}
			cmd := fmt.Sprintf("%s %s -f", jctl, unit)
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("logs %s on %s (Ctrl+C to stop)", unit, h.Entry.Name))
		}
	case "i":
		if len(m.services) > 0 {
			h := m.hosts[m.selectedHost]
			unit := m.services[m.serviceCursor].Name + ".service"
			sysctl := "sudo systemctl"
			if h.Entry.SystemdMode == "user" {
				sysctl = "systemctl --user"
			}
			cmd := fmt.Sprintf("%s status %s --no-pager", sysctl, unit)
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("status %s on %s", unit, h.Entry.Name))
		}
	case "r":
		m.services = nil
		m.filterText = ""
		return m, m.fetchServices()
	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.serviceCursor = 0
		} else {
			m.view = viewResourcePicker
		}
	}
	return m, nil
}

func (m Model) handleContainerListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.containerCursor > 0 {
			m.containerCursor--
		}
	case "down", "j":
		if m.containerCursor < len(m.containers)-1 {
			m.containerCursor++
		}
	case "l":
		if len(m.containers) > 0 {
			h := m.hosts[m.selectedHost]
			ctr := m.containers[m.containerCursor].Name
			cmd := fmt.Sprintf("podman logs -f '%s'", shellQuote(ctr))
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("logs %s on %s (Ctrl+C to stop)", ctr, h.Entry.Name))
		}
	case "i":
		if len(m.containers) > 0 {
			h := m.hosts[m.selectedHost]
			ctr := m.containers[m.containerCursor].Name
			cmd := fmt.Sprintf("podman inspect '%s' | less", shellQuote(ctr))
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("inspect %s on %s", ctr, h.Entry.Name))
		}
	case "e":
		if len(m.containers) > 0 {
			h := m.hosts[m.selectedHost]
			ctr := m.containers[m.containerCursor].Name
			cmd := fmt.Sprintf("podman exec -it '%s' /bin/bash || podman exec -it '%s' /bin/sh", shellQuote(ctr), shellQuote(ctr))
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("exec %s on %s", ctr, h.Entry.Name))
		}
	case "r":
		m.containers = nil
		m.flash = "Refreshing..."
		return m, m.fetchContainers()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
}

func (m Model) handleCronListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cronCursor > 0 {
			m.cronCursor--
		}
	case "down", "j":
		if m.cronCursor < len(m.cronJobs)-1 {
			m.cronCursor++
		}
	case "r":
		m.cronJobs = nil
		m.flash = "Refreshing..."
		return m, m.fetchCronJobs()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
}

func (m Model) handleLogLevelPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.logLevelCursor > 0 {
			m.logLevelCursor--
		}
	case "down", "j":
		if m.logLevelCursor < len(m.logLevels)-1 {
			m.logLevelCursor++
		}
	case "enter":
		if len(m.logLevels) > 0 {
			selected := m.logLevels[m.logLevelCursor]
			m.selectedLogLevel = selected.Code
			m.errorCursor = 0
			m.errorLogs = nil
			m.view = viewErrorLogList
			return m, m.fetchErrorLogs()
		}
	case "r":
		m.logLevels = nil
		m.flash = "Refreshing..."
		return m, m.fetchLogLevels()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
}

func (m Model) handleErrorLogListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredErrorLogs()

	switch msg.String() {
	case "up", "k":
		if m.errorCursor > 0 {
			m.errorCursor--
		}
	case "down", "j":
		if m.errorCursor < len(filtered)-1 {
			m.errorCursor++
		}
	case "/":
		m.filterActive = true
		m.filterText = ""
		m.errorCursor = 0
	case "enter":
		if len(filtered) > 0 && m.errorCursor < len(filtered) {
			m.showLogDetail = true
		}
	case "l":
		if len(filtered) > 0 {
			h := m.hosts[m.selectedHost]
			since := h.ErrorLogSince
			cmd := fmt.Sprintf("sudo journalctl -p %s --since '%s' --no-pager -q --no-hostname | less", m.selectedLogLevel, since)
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("%s logs on %s", m.logLevels[m.logLevelCursor].Level, h.Entry.Name))
		}
	case "r":
		m.errorLogs = nil
		m.filterText = ""
		m.flash = "Refreshing..."
		return m, m.fetchErrorLogs()
	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.errorCursor = 0
		} else {
			m.view = viewLogLevelPicker
		}
	}
	return m, nil
}

func (m Model) handleUpdateListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.updateCursor > 0 {
			m.updateCursor--
		}
	case "down", "j":
		if m.updateCursor < len(m.updates)-1 {
			m.updateCursor++
		}
	case "u":
		h := m.hosts[m.selectedHost]
		m.showConfirm = true
		m.confirmMessage = fmt.Sprintf("Apply ALL updates on %s? [Y/n]", h.Entry.Name)
		m.confirmCmd = `sudo dnf update -y --setopt=skip_if_unavailable=1; echo ''; echo 'Update complete. Press Enter to return...'`
		m.confirmBanner = fmt.Sprintf("full update on %s", h.Entry.Name)
	case "p":
		h := m.hosts[m.selectedHost]
		m.showConfirm = true
		m.confirmMessage = fmt.Sprintf("Apply SECURITY updates on %s? [Y/n]", h.Entry.Name)
		m.confirmCmd = `sudo dnf update --security -y --setopt=skip_if_unavailable=1; echo ''; echo 'Security update complete. Press Enter to return...'`
		m.confirmBanner = fmt.Sprintf("security update on %s", h.Entry.Name)
	case "r":
		m.updates = nil
		m.flash = "Refreshing..."
		return m, m.fetchUpdates()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
}

func (m Model) handleDiskListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.diskCursor > 0 {
			m.diskCursor--
		}
	case "down", "j":
		if m.diskCursor < len(m.disks)-1 {
			m.diskCursor++
		}
	case "r":
		m.disks = nil
		m.flash = "Refreshing..."
		return m, m.fetchDisk()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
}

func (m Model) handleSubscriptionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.subscriptionCursor > 0 {
			m.subscriptionCursor--
		}
	case "down", "j":
		if m.subscriptionCursor < len(m.subscriptions)-1 {
			m.subscriptionCursor++
		}
	case "d":
		if len(m.subscriptions) > 0 {
			sub := m.subscriptions[m.subscriptionCursor]
			if strings.HasPrefix(sub.Field, "Repo: ") {
				repoID := strings.TrimPrefix(sub.Field, "Repo: ")
				h := m.hosts[m.selectedHost]
				m.showConfirm = true
				m.confirmMessage = fmt.Sprintf("Disable repo %s? [Y/n]", repoID)
				m.confirmCmd = fmt.Sprintf("sudo dnf config-manager --set-disabled '%s' && echo '' && echo '\u2713 Repo %s disabled'", shellQuote(repoID), repoID)
				m.confirmBanner = fmt.Sprintf("disable %s on %s", repoID, h.Entry.Name)
			}
		}
	case "r":
		m.subscriptions = nil
		m.flash = "Refreshing..."
		return m, m.fetchSubscription()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
}

func (m Model) handleAccountListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showAccountDetail {
		m.showAccountDetail = false
		m.accountDetailSections = nil
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.accountCursor > 0 {
			m.accountCursor--
		}
	case "down", "j":
		accts := m.filteredAccounts()
		if m.accountCursor < len(accts)-1 {
			m.accountCursor++
		}
	case "enter":
		accts := m.filteredAccounts()
		if len(accts) > 0 {
			user := accts[m.accountCursor].User
			m.flash = fmt.Sprintf("Loading %s...", user)
			return m, m.fetchAccountDetail(user)
		}
	case "/":
		m.filterActive = true
		m.filterText = ""
		m.accountCursor = 0
	case "r":
		m.accounts = nil
		m.flash = "Refreshing..."
		return m, m.fetchAccounts()
	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.accountCursor = 0
		} else {
			m.view = viewResourcePicker
		}
	}
	return m, nil
}
