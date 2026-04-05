package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type view int

const (
	viewFleetPicker view = iota
	viewHostList
	viewResourcePicker
	viewServiceList
	viewContainerList
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
			m.services = msg.services
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
			return m, tea.Batch(tea.EnterAltScreen, m.fetchServices())
		case viewContainerList:
			return m, tea.Batch(tea.EnterAltScreen, m.fetchContainers())
		}
		return m, tea.EnterAltScreen

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			return m, m.startProbe()
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
			// pre-fetch services and containers for accurate counts
			return m, tea.Batch(m.fetchServices(), m.fetchContainers())
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
		if m.resourceCursor < 1 {
			m.resourceCursor++
		}
	case "enter":
		if m.resourceCursor == 0 {
			m.serviceCursor = 0
			m.services = nil
			m.view = viewServiceList
			return m, m.fetchServices()
		} else {
			m.containerCursor = 0
			m.containers = nil
			m.view = viewContainerList
			return m, m.fetchContainers()
		}
	case "esc":
		m.view = viewHostList
	}
	return m, nil
}

func (m model) handleServiceListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.serviceCursor > 0 {
			m.serviceCursor--
		}
	case "down", "j":
		if m.serviceCursor < len(m.services)-1 {
			m.serviceCursor++
		}
	case "s":
		if len(m.services) > 0 {
			return m, m.svcAction("start")
		}
	case "o":
		if len(m.services) > 0 {
			return m, m.svcAction("stop")
		}
	case "t":
		if len(m.services) > 0 {
			return m, m.svcAction("restart")
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
		m.flash = "Refreshing..."
		return m, m.fetchServices()
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
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
	var hosts []host
	for _, g := range f.Groups {
		for _, e := range g.Hosts {
			hosts = append(hosts, host{
				Entry:  e,
				Group:  g.Name,
				Status: hostConnecting,
			})
		}
	}
	for _, e := range f.Hosts {
		hosts = append(hosts, host{
			Entry:  e,
			Status: hostConnecting,
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
	} else {
		s += m.renderHintBar([][]string{
			{"Enter", "Drill In"},
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
	s := m.renderHeader(breadcrumb, m.resourceCursor+1, 2) + "\n"
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

	rows := []resRow{
		{"Services", svcTotal, svcRunning, svcFailed},
		{"Containers", ctnTotal, ctnRunning, ctnFailed},
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

	breadcrumb := f.Name + " › " + h.Entry.Name + " › Services"
	s := m.renderHeader(breadcrumb, m.serviceCursor+1, len(m.services)) + "\n"
	s += borderStyle.Render("┌"+strings.Repeat("─", iw)+"┐") + "\n"

	if len(m.services) == 0 {
		s += borderedRow("  No services found.", iw, normalRowStyle) + "\n"
	} else {
		nameCol := len("SERVICE")
		enabledCol := len("ENABLED")
		for _, svc := range m.services {
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
		if end > len(m.services) {
			end = len(m.services)
		}

		for i := offset; i < end; i++ {
			svc := m.services[i]
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
	s += m.renderHintBar([][]string{
		{"s", "Start"},
		{"o", "Stop"},
		{"t", "Restart"},
		{"l", "Logs"},
		{"i", "Status"},
		{"Esc", "Back"},
	})
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
