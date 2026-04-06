package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// tickMsg triggers a periodic host probe refresh.
type tickMsg time.Time

func (m model) tickCmd() tea.Cmd {
	interval, err := time.ParseDuration(m.fleets[m.selectedFleet].Defaults.RefreshInterval)
	if err != nil {
		return nil
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type view int

const (
	viewFleetPicker view = iota
	viewHostList
	viewResourcePicker
	viewServiceList
	viewContainerList
	viewCronList
	viewLogLevelPicker
	viewErrorLogList
	viewUpdateList
	viewDiskList
	viewSubscription
)

// hostProbeResult is sent when an SSH probe completes for a host.
type hostProbeResult struct {
	index int
	info  probeInfo
	err   error
}

type model struct {
	view view

	// fleet picker
	fleets      []fleet
	fleetCursor int

	// host list
	selectedFleet int
	hosts         []host
	hostCursor    int

	// resource picker
	selectedHost   int
	resourceCursor int

	// service list
	services      []service
	serviceCursor int

	// container list
	containers      []container
	containerCursor int

	// cron jobs
	cronJobs    []cronJob
	cronCursor  int

	// log level picker
	logLevels      []logLevelEntry
	logLevelCursor int

	// error logs
	errorLogs      []errorLog
	errorCursor    int
	selectedLogLevel string

	// updates
	updates      []update
	updateCursor int

	// disk
	disks      []disk
	diskCursor int

	// subscription
	subscriptions      []subscription
	subscriptionCursor int

	// filter / search
	filterActive bool
	filterText   string

	// log detail
	showLogDetail bool

	// confirmation prompt
	showConfirm    bool
	confirmMessage string
	confirmCmd     string
	confirmBanner  string

	// SSH
	ssh *sshManager

	// password prompt
	passwordInput    string
	passwordHostIdx  int
	showPasswordPrompt bool

	// flash message
	flash      string
	flashError bool

	// terminal size
	width  int
	height int
}

func newModel(fleets []fleet) model {
	return model{
		view:   viewFleetPicker,
		fleets: fleets,
		ssh:    newSSHManager(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case hostProbeResult:
		if msg.index < len(m.hosts) {
			if msg.err != nil {
				if isAuthError(msg.err) {
					// mark as needing password
					m.hosts[msg.index].Status = hostUnreachable
					m.hosts[msg.index].Error = "password required"
					m.hosts[msg.index].NeedsPassword = true
					// show prompt if not already showing one
					if !m.showPasswordPrompt {
						m.passwordHostIdx = msg.index
						m.passwordInput = ""
						m.showPasswordPrompt = true
					}
					return m, nil
				}
				m.hosts[msg.index].Status = hostUnreachable
				m.hosts[msg.index].Error = msg.err.Error()
			} else {
				m.applyProbeInfo(msg.index, msg.info)
			}
		}
		return m, nil

	case passwordRetryResult:
		if msg.index < len(m.hosts) {
			if msg.err != nil {
				m.hosts[msg.index].Status = hostUnreachable
				m.hosts[msg.index].Error = msg.err.Error()
			} else {
				m.hosts[msg.index].NeedsPassword = false
				m.applyProbeInfo(msg.index, msg.info)
			}
		}
		// check if more hosts need the same password
		return m, m.retryRemainingPasswordHosts()


	case fetchServicesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.services == nil {
				m.services = []service{}
			} else {
				m.services = msg.services
			}
		}
		return m, nil

	case fetchContainersMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.containers = msg.containers
		}
		return m, nil

	case fetchCronMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.cronJobs = msg.jobs
		}
		return m, nil

	case fetchLogLevelsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.logLevels = msg.levels
		}
		return m, nil

	case fetchErrorLogsMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.errorLogs = msg.logs
		}
		return m, nil

	case fetchUpdatesMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			if msg.updates == nil {
				m.updates = []update{}
			} else {
				m.updates = msg.updates
			}
		}
		return m, nil

	case fetchSubscriptionMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.subscriptions = msg.subs
		}
		return m, nil

	case fetchDiskMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed: %v", msg.err)
			m.flashError = true
		} else {
			m.disks = msg.disks
		}
		return m, nil

	case serviceActionMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("%s %s failed: %v", msg.action, msg.unit, msg.err)
			m.flashError = true
		} else {
			m.flash = fmt.Sprintf("%s %s: ok", msg.action, msg.unit)
		}
		// refresh service list after action
		return m, m.fetchServices()

	case editFinishedMsg:
		// reload fleets after editor returns
		fleets, err := scanFleets()
		if err != nil {
			m.flash = fmt.Sprintf("Reload failed: %v", err)
			m.flashError = true
		} else {
			m.fleets = fleets
			if m.fleetCursor >= len(m.fleets) {
				m.fleetCursor = max(0, len(m.fleets)-1)
			}
			m.flash = "Reloaded"
		}
		return m, tea.EnterAltScreen

	case sshHandoverFinishedMsg:
		// refresh list after terminal handover returns
		switch m.view {
		case viewServiceList:
			m.serviceCursor = 0
			m.services = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchServices())
		case viewContainerList:
			return m, tea.Batch(tea.EnterAltScreen, m.fetchContainers())
		case viewUpdateList:
			m.updates = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchUpdates())
		case viewSubscription:
			m.subscriptions = nil
			return m, tea.Batch(tea.EnterAltScreen, m.fetchSubscription())
		}
		return m, tea.EnterAltScreen

	case tickMsg:
		if m.view == viewHostList {
			return m, tea.Batch(m.startProbe(), m.tickCmd())
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// log detail mode — any key goes back
	if m.showLogDetail {
		m.showLogDetail = false
		return m, nil
	}

	// filter mode — capture search input
	if m.filterActive {
		switch msg.Type {
		case tea.KeyEnter:
			m.filterActive = false
			m.serviceCursor = 0
			m.errorCursor = 0
		case tea.KeyEsc:
			m.filterActive = false
			m.filterText = ""
			m.serviceCursor = 0
			m.errorCursor = 0
		case tea.KeyBackspace:
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.serviceCursor = 0
				m.errorCursor = 0
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.filterText += string(msg.Runes)
				m.serviceCursor = 0
				m.errorCursor = 0
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
		// confirmed — execute
		m.showConfirm = false
		h := m.hosts[m.selectedHost]
		cmd := m.confirmCmd
		banner := m.confirmBanner
		m.confirmCmd = ""
		m.confirmBanner = ""
		m.flash = ""
		return m, sshHandover(h, []string{cmd}, banner)
	}

	// password prompt mode — capture input
	if m.showPasswordPrompt {
		switch msg.Type {
		case tea.KeyEnter:
			m.showPasswordPrompt = false
			password := m.passwordInput
			m.passwordInput = "" // clear input immediately
			idx := m.passwordHostIdx
			m.hosts[idx].Status = hostConnecting
			m.flash = ""
			// cache password temporarily in ssh manager for batch retries
			m.ssh.setCachedPassword(password)
			return m, m.ssh.retryWithPassword(idx, m.hosts[idx], password)
		case tea.KeyEsc:
			m.showPasswordPrompt = false
			m.hosts[m.passwordHostIdx].Status = hostUnreachable
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
	case viewSubscription:
		return m.handleSubscriptionKeys(msg)
	}
	return m, nil
}

func (m model) handleFleetPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		fleets, err := scanFleets()
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
			m.ssh.close()
			m.hosts = buildHostList(f)
			m.hostCursor = 0
			m.view = viewHostList
			return m, tea.Batch(m.startProbe(), m.tickCmd())
		}
	}
	return m, nil
}

func (m model) handleHostListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			if h.Status != hostOnline {
				m.flash = "Host is not reachable"
				m.flashError = true
				return m, nil
			}
			return m, sshHandover(h, []string{}, fmt.Sprintf("shell %s@%s", h.Entry.User, h.Entry.Name))
		}
	case "R":
		if len(m.hosts) > 0 {
			h := m.hosts[m.hostCursor]
			if h.Status != hostOnline {
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
		m.ssh.close()
		m.hosts = buildHostList(f)
		m.flash = "Refreshing..."
		return m, m.startProbe()
	case "enter":
		if len(m.hosts) > 0 {
			h := m.hosts[m.hostCursor]
			if h.Status != hostOnline {
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
		m.ssh.close()
		m.view = viewFleetPicker
	}
	return m, nil
}

func (m model) handleResourcePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.resourceCursor > 0 {
			m.resourceCursor--
		}
	case "down", "j":
		if m.resourceCursor < 6 {
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
		}
	case "esc":
		m.view = viewHostList
	}
	return m, nil
}

func (m model) filteredServices() []service {
	if m.filterText == "" {
		return m.services
	}
	filter := strings.ToLower(m.filterText)
	var filtered []service
	for _, s := range m.services {
		line := strings.ToLower(s.Name + " " + s.State + " " + s.Enabled + " " + s.Description)
		if strings.Contains(line, filter) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (m model) handleServiceListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m model) findServiceIndex(name string) int {
	for i, s := range m.services {
		if s.Name == name {
			return i
		}
	}
	return 0
}

func (m model) handleContainerListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			cmd := fmt.Sprintf("podman logs -f %s", ctr)
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("logs %s on %s (Ctrl+C to stop)", ctr, h.Entry.Name))
		}
	case "i":
		if len(m.containers) > 0 {
			h := m.hosts[m.selectedHost]
			ctr := m.containers[m.containerCursor].Name
			cmd := fmt.Sprintf("podman inspect %s | less", ctr)
			return m, sshHandover(h, []string{cmd}, fmt.Sprintf("inspect %s on %s", ctr, h.Entry.Name))
		}
	case "e":
		if len(m.containers) > 0 {
			h := m.hosts[m.selectedHost]
			ctr := m.containers[m.containerCursor].Name
			cmd := fmt.Sprintf("podman exec -it %s /bin/bash || podman exec -it %s /bin/sh", ctr, ctr)
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

func (m model) handleCronListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleLogLevelPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			// not a key=value pattern — treat the rest as plain text
			break
		}

		remaining = remaining[eqIdx+1:]

		var value string
		if len(remaining) > 0 && remaining[0] == '"' {
			// quoted value — find closing quote
			endQuote := strings.Index(remaining[1:], "\"")
			if endQuote >= 0 {
				value = remaining[1 : endQuote+1]
				remaining = remaining[endQuote+2:]
			} else {
				value = remaining[1:]
				remaining = ""
			}
		} else {
			// unquoted value — until next space
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

// filteredErrorLogs returns the error logs matching the current filter.
func (m model) filteredErrorLogs() []errorLog {
	if m.filterText == "" {
		return m.errorLogs
	}
	filter := strings.ToLower(m.filterText)
	var filtered []errorLog
	for _, e := range m.errorLogs {
		line := strings.ToLower(e.Time + " " + e.Unit + " " + e.Message)
		if strings.Contains(line, filter) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (m model) handleErrorLogListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleUpdateListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleDiskListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// applyProbeInfo updates a host with successful probe results.
func (m *model) applyProbeInfo(idx int, info probeInfo) {
	m.hosts[idx].Status = hostOnline
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
	m.hosts[idx].LastUpdate = info.LastUpdate
	m.hosts[idx].LastSecurity = info.LastSecurity
}

// retryRemainingPasswordHosts retries connection for hosts that still need a password.
// Uses the password from the last successful retry (stored temporarily in sshManager).
func (m model) retryRemainingPasswordHosts() tea.Cmd {
	var cmds []tea.Cmd
	for i, h := range m.hosts {
		if h.NeedsPassword && h.Status == hostUnreachable {
			idx := i
			hh := h
			sm := m.ssh
			cmds = append(cmds, func() tea.Msg {
				return sm.retryWithCachedPassword(idx, hh)
			})
		}
	}
	if len(cmds) == 0 {
		// all done — clear the cached password
		m.ssh.clearPassword()
		return nil
	}
	return tea.Batch(cmds...)
}

// startProbe launches parallel SSH connections and probes for all hosts.
// Returns a batch of commands, one per host, that will send hostProbeResult messages.
func (m model) startProbe() tea.Cmd {
	var cmds []tea.Cmd
	for i, h := range m.hosts {
		idx := i
		hh := h
		sm := m.ssh
		cmds = append(cmds, func() tea.Msg {
			return sm.connectAndProbe(idx, hh)
		})
	}
	return tea.Batch(cmds...)
}

// buildHostList creates the runtime host list from a fleet definition.
func buildHostList(f fleet) []host {
	errorLogSince := f.Defaults.ErrorLogSince
	var hosts []host
	for _, g := range f.Groups {
		for _, e := range g.Hosts {
			hosts = append(hosts, host{
				Entry:         e,
				Group:         g.Name,
				Status:        hostConnecting,
				ErrorLogSince: errorLogSince,
			})
		}
	}
	for _, e := range f.Hosts {
		hosts = append(hosts, host{
			Entry:         e,
			Status:        hostConnecting,
			ErrorLogSince: errorLogSince,
		})
	}
	return hosts
}

// --- View rendering ---

func (m model) View() string {
	switch m.view {
	case viewFleetPicker:
		return m.renderFleetPicker()
	case viewHostList:
		return m.renderHostList()
	case viewResourcePicker:
		return m.renderResourcePicker()
	case viewServiceList:
		return m.renderServiceList()
	case viewContainerList:
		return m.renderContainerList()
	case viewCronList:
		return m.renderCronList()
	case viewLogLevelPicker:
		return m.renderLogLevelPicker()
	case viewErrorLogList:
		return m.renderErrorLogList()
	case viewUpdateList:
		return m.renderUpdateList()
	case viewDiskList:
		return m.renderDiskList()
	case viewSubscription:
		return m.renderSubscription()
	}
	return ""
}

func (m model) renderHeader(breadcrumb string, current, total int) string {
	title := "fleetdesk"
	if breadcrumb != "" {
		title += " › " + breadcrumb
	}
	left := headerStyle.Render(title)
	count := headerCountStyle.Render(fmt.Sprintf("%d/%d", current, total))
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(count)
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + count
}

func (m model) renderHintBar(hints [][]string) string {
	var parts []string
	for _, h := range hints {
		parts = append(parts, hintKeyStyle.Render("<"+h[0]+">")+" "+hintActionStyle.Render(h[1]))
	}
	bar := strings.Join(parts, "  ")
	if m.flash != "" {
		style := flashStyle
		if m.flashError {
			style = flashErrorStyle
		}
		bar += "  " + style.Render(m.flash)
	}
	return hintBarStyle.Width(m.width).Render(bar)
}

// borderedRow wraps content with │ on each side, clamped to exactly w display columns.
func borderedRow(content string, w int, style lipgloss.Style) string {
	dw := runewidth.StringWidth(content)
	if dw > w {
		truncated := ""
		col := 0
		for _, r := range content {
			rw := runewidth.RuneWidth(r)
			if col+rw >= w {
				break
			}
			truncated += string(r)
			col += rw
		}
		content = truncated + "…"
		dw = runewidth.StringWidth(content)
	}
	if dw < w {
		content += strings.Repeat(" ", w-dw)
	}
	b := borderStyle.Render("│")
	return b + style.Render(content) + b
}

func (m model) padToBottom(s string, iw int) string {
	lines := strings.Count(s, "\n")
	for i := lines; i < m.height-3; i++ {
		s += borderedRow("", iw, normalRowStyle) + "\n"
	}
	return s
}

func (m model) renderFleetPicker() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader("", m.fleetCursor+1, len(m.fleets)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.fleets) == 0 {
		s += borderedRow("  No fleet files found in ~/.config/fleetdesk/", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("FLEET")
		for _, f := range m.fleets {
			if len(f.Name) > nameCol {
				nameCol = len(f.Name)
			}
		}
		nameCol += 2

		hdr := fmt.Sprintf("     %-*s  %-6s  %s", nameCol, "FLEET", "TYPE", "HOSTS")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.fleetCursor >= offset+maxVisible {
			offset = m.fleetCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.fleets) {
			end = len(m.fleets)
		}

		for i := offset; i < end; i++ {
			f := m.fleets[i]
			cur := "   "
			if i == m.fleetCursor {
				cur = " ▸ "
			}

			ftype := f.Type
			if ftype == "" {
				ftype = "vm"
			}
			hostCount := m.fleetHostCount(f)
			line := fmt.Sprintf("%s  %-*s  %-6s  %d", cur, nameCol, f.Name, ftype, hostCount)

			var style lipgloss.Style
			if i == m.fleetCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar([][]string{
		{"Enter", "Select"},
		{"e", "Edit"},
		{"r", "Reload"},
		{"q", "Quit"},
	})
	return s
}

func (m model) fleetHostCount(f fleet) int {
	count := len(f.Hosts)
	for _, g := range f.Groups {
		count += len(g.Hosts)
	}
	return count
}

func (m model) renderHostList() string {
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader(f.Name, m.hostCursor+1, len(m.hosts)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.hosts) == 0 {
		s += borderedRow("  No hosts in fleet.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("HOST")
		osCol := len("OS")
		for _, h := range m.hosts {
			if len(h.Entry.Name) > nameCol {
				nameCol = len(h.Entry.Name)
			}
			if len(h.OS) > osCol {
				osCol = len(h.OS)
			}
		}
		nameCol += 2
		osCol += 2

		// compute dynamic column widths from actual data
		upCol := len("UP SINCE")
		updCol := len("LAST UPDATE")
		secCol := len("LAST SECURITY")
		for _, h := range m.hosts {
			if len(h.UpSince) > upCol {
				upCol = len(h.UpSince)
			}
			if len(h.LastUpdate) > updCol {
				updCol = len(h.LastUpdate)
			}
			if len(h.LastSecurity) > secCol {
				secCol = len(h.LastSecurity)
			}
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %5s  %5s  %-*s  %-*s", nameCol, "HOST", osCol, "OS", upCol, "UP SINCE", "SVC", "CTN", updCol, "LAST UPDATE", secCol, "LAST SECURITY")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.hostCursor >= offset+maxVisible {
			offset = m.hostCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.hosts) {
			end = len(m.hosts)
		}

		// build group start index map
		groupStarts := make(map[int]string)
		for i, h := range m.hosts {
			if h.Group != "" {
				if i == 0 || m.hosts[i-1].Group != h.Group {
					groupStarts[i] = h.Group
				}
			}
		}

		for i := offset; i < end; i++ {
			// render group header if this host starts a new group
			if groupName, ok := groupStarts[i]; ok {
				groupLine := fmt.Sprintf("  ── %s ──", groupName)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
			}

			h := m.hosts[i]
			cur := "   "
			if i == m.hostCursor {
				cur = " ▸ "
			}

			var line string
			switch h.Status {
			case hostConnecting:
				line = fmt.Sprintf("%s  %-*s  connecting...", cur, nameCol, h.Entry.Name)
			case hostUnreachable:
				reason := h.Error
				if reason == "" {
					reason = "unknown"
				}
				line = fmt.Sprintf("%s  %-*s  unreachable (%s)", cur, nameCol, h.Entry.Name, reason)
			default:
				line = fmt.Sprintf("%s  %-*s  %-*s  %-*s  %5d  %5d  %-*s  %-*s",
					cur, nameCol, h.Entry.Name, osCol, h.OS, upCol, h.UpSince, h.ServiceCount, h.ContainerCount, updCol, h.LastUpdate, secCol, h.LastSecurity)
			}

			var style lipgloss.Style
			if i == m.hostCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.showPasswordPrompt {
		user := m.hosts[m.passwordHostIdx].Entry.User
		masked := strings.Repeat("*", len(m.passwordInput))
		prompt := fmt.Sprintf("  Password for %s: %s█", user, masked)
		s += hintBarStyle.Width(m.width).Render(prompt)
	} else if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else {
		s += m.renderHintBar([][]string{
			{"Enter", "Drill In"},
			{"x", "Shell"},
			{"R", "Reboot"},
			{"r", "Refresh"},
			{"Esc", "Back"},
			{"q", "Quit"},
		})
	}
	return s
}

func (m model) renderResourcePicker() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name
	resourceCount := 7
	s := m.renderHeader(breadcrumb, m.resourceCursor+1, resourceCount) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	nameCol := len("RESOURCE") + 4

	hdr := fmt.Sprintf("     %-*s  %7s  %7s  %7s", nameCol, "RESOURCE", "TOTAL", "RUNNING", "FAILED")
	s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

	type resRow struct {
		name    string
		total   int
		running int
		failed  int
	}

	// use fetched (filtered) data if available, otherwise probe counts
	svcTotal, svcRunning, svcFailed := h.ServiceCount, h.ServiceRunning, h.ServiceFailed
	ctnTotal, ctnRunning := h.ContainerCount, h.ContainerRunning
	if len(m.services) > 0 {
		svcTotal = len(m.services)
		svcRunning = 0
		svcFailed = 0
		for _, s := range m.services {
			switch s.State {
			case "running":
				svcRunning++
			case "failed":
				svcFailed++
			}
		}
	}
	ctnFailed := 0
	if len(m.containers) > 0 {
		ctnTotal = len(m.containers)
		ctnRunning = 0
		for _, c := range m.containers {
			if strings.HasPrefix(c.Status, "Up") {
				ctnRunning++
			} else if !strings.HasPrefix(c.Status, "Exited (0)") && c.Status != "Created" {
				ctnFailed++
			}
		}
	}

	updTotal, updFailed := h.UpdateCount, 0
	if len(m.updates) > 0 {
		updTotal = 0
		for _, u := range m.updates {
			if u.Type == "error" {
				updFailed++
			} else {
				updTotal++
			}
		}
	}

	rows := []resRow{
		{"Services", svcTotal, svcRunning, svcFailed},
		{"Containers", ctnTotal, ctnRunning, ctnFailed},
		{"Cron Jobs", h.CronCount, 0, 0},
		{"System Logs", h.ErrorCount, 0, 0},
		{"Updates", updTotal, 0, updFailed},
		{"Disk", h.DiskCount, 0, h.DiskHighCount},
		{"Subscription", 0, 0, 0},
	}
	for i, r := range rows {
		cur := "   "
		if i == m.resourceCursor {
			cur = " ▸ "
		}
		failedStr := fmt.Sprintf("%d", r.failed)
		line := fmt.Sprintf("%s  %-*s  %7d  %7d  %7s", cur, nameCol, r.name, r.total, r.running, failedStr)

		var style lipgloss.Style
		if i == m.resourceCursor {
			style = selectedRowStyle
		} else if i%2 == 0 {
			style = altRowStyle
		} else {
			style = normalRowStyle
		}
		s += borderedRow(line, iw, style) + "\n"
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar([][]string{
		{"Enter", "Select"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})
	return s
}

func (m model) renderServiceList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredServices()
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	breadcrumb := f.Name + " › " + h.Entry.Name + " › Services"
	s := m.renderHeader(breadcrumb+filterInfo, m.serviceCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if m.services == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No services found.", iw, normalRowStyle) + "\n"
		}
	} else {
		nameCol := len("SERVICE")
		enabledCol := len("ENABLED")
		for _, svc := range filtered {
			if len(svc.Name) > nameCol {
				nameCol = len(svc.Name)
			}
			if len(svc.Enabled) > enabledCol {
				enabledCol = len(svc.Enabled)
			}
		}
		nameCol += 2
		enabledCol += 2

		hdr := fmt.Sprintf("     %-*s  %-10s  %-*s  %s", nameCol, "SERVICE", "STATE", enabledCol, "ENABLED", "DESCRIPTION")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.serviceCursor >= offset+maxVisible {
			offset = m.serviceCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			svc := filtered[i]
			cur := "   "
			if i == m.serviceCursor {
				cur = " ▸ "
			}
			prefix := ""
			if svc.State == "failed" {
				prefix = "✗ "
			}
			desc := svc.Description
			if desc == "" {
				desc = "—"
			}
			line := fmt.Sprintf("%s  %s%-*s  %-10s  %-*s  %s", cur, prefix, nameCol, svc.Name, svc.State, enabledCol, svc.Enabled, desc)

			var style lipgloss.Style
			if i == m.serviceCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"/", "Search"},
			{"s", "Start"},
			{"o", "Stop"},
			{"t", "Restart"},
			{"l", "Logs"},
			{"i", "Status"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}

func (m model) renderContainerList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Containers"
	s := m.renderHeader(breadcrumb, m.containerCursor+1, len(m.containers)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.containers) == 0 {
		s += borderedRow("  No containers found.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("CONTAINER")
		imgCol := len("IMAGE")
		for _, c := range m.containers {
			if len(c.Name) > nameCol {
				nameCol = len(c.Name)
			}
			if len(c.Image) > imgCol {
				imgCol = len(c.Image)
			}
		}
		nameCol += 2
		imgCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", nameCol, "CONTAINER", imgCol, "IMAGE", "STATUS")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.containerCursor >= offset+maxVisible {
			offset = m.containerCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.containers) {
			end = len(m.containers)
		}

		for i := offset; i < end; i++ {
			c := m.containers[i]
			cur := "   "
			if i == m.containerCursor {
				cur = " ▸ "
			}
			prefix := ""
			if !strings.HasPrefix(c.Status, "Up") && !strings.HasPrefix(c.Status, "Exited (0)") && c.Status != "Created" {
				prefix = "✗ "
			}
			line := fmt.Sprintf("%s  %s%-*s  %-*s  %s", cur, prefix, nameCol, c.Name, imgCol, c.Image, c.Status)

			var style lipgloss.Style
			if i == m.containerCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar([][]string{
		{"l", "Logs"},
		{"i", "Inspect"},
		{"e", "Exec"},
		{"Esc", "Back"},
	})
	return s
}

func (m model) renderLogLevelPicker() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Logs"
	s := m.renderHeader(breadcrumb, m.logLevelCursor+1, len(m.logLevels)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.logLevels) == 0 {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("LEVEL") + 6

		hdr := fmt.Sprintf("     %-*s  %8s", nameCol, "LEVEL", "COUNT")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		for i, l := range m.logLevels {
			cur := "   "
			if i == m.logLevelCursor {
				cur = " ▸ "
			}
			line := fmt.Sprintf("%s  %-*s  %8d", cur, nameCol, l.Level, l.Count)

			var style lipgloss.Style
			if i == m.logLevelCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar([][]string{
		{"Enter", "View Logs"},
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}

func (m model) renderCronList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Cron Jobs"
	s := m.renderHeader(breadcrumb, m.cronCursor+1, len(m.cronJobs)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.cronJobs) == 0 {
		s += borderedRow("  No cron jobs found.", iw, normalRowStyle) + "\n"
	} else {
		schedCol := len("SCHEDULE")
		srcCol := len("SOURCE")
		for _, j := range m.cronJobs {
			if len(j.Schedule) > schedCol {
				schedCol = len(j.Schedule)
			}
			if len(j.Source) > srcCol {
				srcCol = len(j.Source)
			}
		}
		schedCol += 2
		srcCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", schedCol, "SCHEDULE", srcCol, "SOURCE", "COMMAND")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.cronCursor >= offset+maxVisible {
			offset = m.cronCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.cronJobs) {
			end = len(m.cronJobs)
		}

		lastGroup := ""
		for i := offset; i < end; i++ {
			j := m.cronJobs[i]

			// group header: crontab = User, anything else = System
			group := "System"
			if j.Source == "crontab" {
				group = "User"
			}
			if group != lastGroup {
				groupLine := fmt.Sprintf("  ── %s ──", group)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
				lastGroup = group
			}

			cur := "   "
			if i == m.cronCursor {
				cur = " ▸ "
			}
			line := fmt.Sprintf("%s  %-*s  %-*s  %s", cur, schedCol, j.Schedule, srcCol, j.Source, j.Command)

			var style lipgloss.Style
			if i == m.cronCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
	s += m.renderHintBar([][]string{
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}

func (m model) renderErrorLogList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	filtered := m.filteredErrorLogs()

	// log detail view
	if m.showLogDetail && m.errorCursor < len(filtered) {
		e := filtered[m.errorCursor]
		breadcrumb := f.Name + " › " + h.Entry.Name + " › Log Detail"
		s := m.renderHeader(breadcrumb, 0, 0) + "\n"
		s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

		s += borderedRow(fmt.Sprintf("  Time: %s", e.Time), iw, colHeaderStyle) + "\n"
		s += borderedRow(fmt.Sprintf("  Unit: %s", e.Unit), iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		// parse structured key=value fields from the message
		kvPairs := parseLogFields(e.Message)
		if len(kvPairs) > 0 {
			fieldCol := 0
			for _, kv := range kvPairs {
				if len(kv[0]) > fieldCol {
					fieldCol = len(kv[0])
				}
			}
			fieldCol += 2

			for i, kv := range kvPairs {
				line := fmt.Sprintf("  %-*s  %s", fieldCol, kv[0], kv[1])
				var style lipgloss.Style
				if kv[0] == "level" && (kv[1] == "error" || kv[1] == "crit") {
					style = lipgloss.NewStyle().Foreground(colorRed)
				} else if kv[0] == "err" || kv[0] == "error" {
					style = lipgloss.NewStyle().Foreground(colorRed)
				} else if i%2 == 0 {
					style = altRowStyle
				} else {
					style = normalRowStyle
				}
				s += borderedRow(line, iw, style) + "\n"
			}
		} else {
			// plain message — word-wrap
			msg := e.Message
			lineWidth := iw - 4
			for len(msg) > 0 {
				end := lineWidth
				if end > len(msg) {
					end = len(msg)
				}
				s += borderedRow("  "+msg[:end], iw, normalRowStyle) + "\n"
				msg = msg[end:]
			}
		}

		s = m.padToBottom(s, iw)
		s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"
		s += hintBarStyle.Width(m.width).Render("  Press any key to return")
		return s
	}

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Logs"
	filterInfo := ""
	if m.filterText != "" {
		filterInfo = fmt.Sprintf(" [filter: %s]", m.filterText)
	}
	s := m.renderHeader(breadcrumb+filterInfo, m.errorCursor+1, len(filtered)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if m.errorLogs == nil {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else if len(filtered) == 0 {
		if m.filterText != "" {
			s += borderedRow(fmt.Sprintf("  No matches for '%s'", m.filterText), iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No errors found.", iw, normalRowStyle) + "\n"
		}
	} else {
		timeCol := len("TIME")
		unitCol := len("UNIT")
		for _, e := range filtered {
			if len(e.Time) > timeCol {
				timeCol = len(e.Time)
			}
			if len(e.Unit) > unitCol {
				unitCol = len(e.Unit)
			}
		}
		timeCol += 2
		if unitCol > 40 {
			unitCol = 40
		}
		unitCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", timeCol, "TIME", unitCol, "UNIT", "MESSAGE")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.errorCursor >= offset+maxVisible {
			offset = m.errorCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := offset; i < end; i++ {
			e := filtered[i]
			cur := "   "
			if i == m.errorCursor {
				cur = " ▸ "
			}
			unit := e.Unit
			if len(unit) > unitCol-2 {
				unit = unit[:unitCol-3] + "…"
			}
			line := fmt.Sprintf("%s  %-*s  %-*s  %s", cur, timeCol, e.Time, unitCol, unit, e.Message)

			var style lipgloss.Style
			if i == m.errorCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.filterActive {
		s += hintBarStyle.Width(m.width).Render(fmt.Sprintf("  /%s█", m.filterText))
	} else {
		s += m.renderHintBar([][]string{
			{"Enter", "Detail"},
			{"/", "Search"},
			{"l", "Full Log"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}

func (m model) renderUpdateList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Updates"
	s := m.renderHeader(breadcrumb, m.updateCursor+1, len(m.updates)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.updates) == 0 {
		if m.updates == nil {
			s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
		} else {
			s += borderedRow("  No pending updates.", iw, normalRowStyle) + "\n"
		}
	} else {
		pkgCol := len("PACKAGE")
		verCol := len("VERSION")
		for _, u := range m.updates {
			if len(u.Package) > pkgCol {
				pkgCol = len(u.Package)
			}
			if len(u.Version) > verCol {
				verCol = len(u.Version)
			}
		}
		pkgCol += 2
		verCol += 2

		hdr := fmt.Sprintf("     %-*s  %-*s  %s", pkgCol, "PACKAGE", verCol, "VERSION", "TYPE")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.updateCursor >= offset+maxVisible {
			offset = m.updateCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.updates) {
			end = len(m.updates)
		}

		lastType := ""
		for i := offset; i < end; i++ {
			u := m.updates[i]

			// group header when type changes
			if u.Type != lastType {
				label := strings.ToUpper(u.Type[:1]) + u.Type[1:]
				groupLine := fmt.Sprintf("  ── %s ──", label)
				s += borderedRow(groupLine, iw, groupHeaderStyle) + "\n"
				lastType = u.Type
			}

			cur := "   "
			if i == m.updateCursor {
				cur = " ▸ "
			}
			line := fmt.Sprintf("%s  %-*s  %-*s  %s", cur, pkgCol, u.Package, verCol, u.Version, u.Type)

			var style lipgloss.Style
			if i == m.updateCursor {
				style = selectedRowStyle
			} else if (u.Type == "security" || u.Type == "error") && i != m.updateCursor {
				style = lipgloss.NewStyle().Foreground(colorRed)
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else {
		s += m.renderHintBar([][]string{
			{"u", "Update All"},
			{"p", "Security Only"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}

func (m model) renderDiskList() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Disk"
	s := m.renderHeader(breadcrumb, m.diskCursor+1, len(m.disks)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.disks) == 0 {
		s += borderedRow("  No partitions found.", iw, normalRowStyle) + "\n"
	} else {
		fsCol := len("FILESYSTEM")
		mountCol := len("MOUNT")
		for _, d := range m.disks {
			if len(d.Filesystem) > fsCol {
				fsCol = len(d.Filesystem)
			}
			if len(d.Mount) > mountCol {
				mountCol = len(d.Mount)
			}
		}
		fsCol += 2
		mountCol += 2

		hdr := fmt.Sprintf("     %-*s  %6s  %6s  %6s  %5s  %-*s", fsCol, "FILESYSTEM", "SIZE", "USED", "AVAIL", "USE%", mountCol, "MOUNT")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.diskCursor >= offset+maxVisible {
			offset = m.diskCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.disks) {
			end = len(m.disks)
		}

		for i := offset; i < end; i++ {
			d := m.disks[i]
			cur := "   "
			if i == m.diskCursor {
				cur = " ▸ "
			}
			line := fmt.Sprintf("%s  %-*s  %6s  %6s  %6s  %5s  %-*s", cur, fsCol, d.Filesystem, d.Size, d.Used, d.Avail, d.UsePercent, mountCol, d.Mount)

			var style lipgloss.Style
			if i == m.diskCursor {
				style = selectedRowStyle
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}

			// highlight high disk usage
			pct := strings.TrimSuffix(d.UsePercent, "%")
			var pctVal int
			fmt.Sscanf(pct, "%d", &pctVal)
			if pctVal >= 90 && i != m.diskCursor {
				style = flashErrorStyle
			} else if pctVal >= 80 && i != m.diskCursor {
				style = lipgloss.NewStyle().Foreground(colorYellow)
			}

			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	s += m.renderHintBar([][]string{
		{"r", "Refresh"},
		{"Esc", "Back"},
	})
	return s
}

func (m model) handleSubscriptionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
				m.confirmCmd = fmt.Sprintf("sudo dnf config-manager --set-disabled %s && echo '' && echo '✓ Repo %s disabled'", repoID, repoID)
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

func (m model) renderSubscription() string {
	h := m.hosts[m.selectedHost]
	f := m.fleets[m.selectedFleet]
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Subscription"
	s := m.renderHeader(breadcrumb, m.subscriptionCursor+1, len(m.subscriptions)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.subscriptions) == 0 {
		s += borderedRow("  Loading...", iw, normalRowStyle) + "\n"
	} else {
		fieldCol := len("FIELD")
		for _, sub := range m.subscriptions {
			if len(sub.Field) > fieldCol {
				fieldCol = len(sub.Field)
			}
		}
		fieldCol += 2

		hdr := fmt.Sprintf("     %-*s  %s", fieldCol, "FIELD", "VALUE")
		s += borderedRow(hdr, iw, colHeaderStyle) + "\n"
		s += borderStyle.Render("├"+strings.Repeat("─", iw)+"┤") + "\n"

		maxVisible := m.height - 8
		if maxVisible < 1 {
			maxVisible = 1
		}
		offset := 0
		if m.subscriptionCursor >= offset+maxVisible {
			offset = m.subscriptionCursor - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(m.subscriptions) {
			end = len(m.subscriptions)
		}

		for i := offset; i < end; i++ {
			sub := m.subscriptions[i]
			cur := "   "
			if i == m.subscriptionCursor {
				cur = " ▸ "
			}
			line := fmt.Sprintf("%s  %-*s  %s", cur, fieldCol, sub.Field, sub.Value)

			var style lipgloss.Style
			if i == m.subscriptionCursor {
				style = selectedRowStyle
			} else if sub.Value == "ERROR" {
				style = lipgloss.NewStyle().Foreground(colorRed)
			} else if i%2 == 0 {
				style = altRowStyle
			} else {
				style = normalRowStyle
			}
			s += borderedRow(line, iw, style) + "\n"
		}
	}

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("└"+strings.Repeat("─", iw)+"┘") + "\n"

	if m.showConfirm {
		s += hintBarStyle.Width(m.width).Render("  " + flashErrorStyle.Render(m.confirmMessage))
	} else {
		s += m.renderHintBar([][]string{
			{"d", "Disable Repo"},
			{"r", "Refresh"},
			{"Esc", "Back"},
		})
	}
	return s
}
