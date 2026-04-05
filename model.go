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
				m.hosts[msg.index].Status = hostUnreachable
				m.hosts[msg.index].Error = msg.err.Error()
			} else {
				m.hosts[msg.index].Status = hostOnline
				m.hosts[msg.index].FQDN = msg.info.FQDN
				m.hosts[msg.index].OS = msg.info.OS
				m.hosts[msg.index].UpSince = msg.info.UpSince
				m.hosts[msg.index].ServiceCount = msg.info.ServiceCount
				m.hosts[msg.index].ContainerCount = msg.info.ContainerCount
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.view = viewResourcePicker
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
			m.view = viewServiceList
			// TODO: fetch services
		} else {
			m.containerCursor = 0
			m.view = viewContainerList
			// TODO: fetch containers
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
	case "esc":
		m.view = viewResourcePicker
	}
	return m, nil
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

		// compute uptime column width from actual data
		upCol := len("UP SINCE")
		for _, h := range m.hosts {
			if len(h.UpSince) > upCol {
				upCol = len(h.UpSince)
			}
		}

		hdr := fmt.Sprintf("     %-*s  %-*s  %-*s  %5s  %5s", nameCol, "HOST", osCol, "OS", upCol, "UP SINCE", "SVC", "CTN")
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

		for i := offset; i < end; i++ {
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
				line = fmt.Sprintf("%s  %-*s  %-*s  %-*s  %5d  %5d",
					cur, nameCol, h.Entry.Name, osCol, h.OS, upCol, h.UpSince, h.ServiceCount, h.ContainerCount)
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
	s += m.renderHintBar([][]string{
		{"Enter", "Drill In"},
		{"r", "Refresh"},
		{"Esc", "Back"},
		{"q", "Quit"},
	})
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

	options := []string{
		fmt.Sprintf("Services              %d units", h.ServiceCount),
		fmt.Sprintf("Containers            %d running", h.ContainerCount),
	}
	for i, opt := range options {
		cur := "   "
		if i == m.resourceCursor {
			cur = " ▸ "
		}
		line := cur + "  " + opt

		var style lipgloss.Style
		if i == m.resourceCursor {
			style = selectedRowStyle
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
		for _, svc := range m.services {
			if len(svc.Name) > nameCol {
				nameCol = len(svc.Name)
			}
		}
		nameCol += 2

		hdr := fmt.Sprintf("     %-*s  %-10s  %s", nameCol, "SERVICE", "STATE", "ENABLED")
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
			line := fmt.Sprintf("%s  %s%-*s  %-10s  %s", cur, prefix, nameCol, svc.Name, svc.State, svc.Enabled)

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
			if c.Status != "running" && c.Status != "exited" {
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
