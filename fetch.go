package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
				services = append(services, svc)
			}
		}
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
		return fetchContainersMsg{containers: containers}
	}
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
