package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func (m Model) renderConfig() string {
	w := m.width
	if w < 20 {
		w = 80
	}
	iw := w - 2

	s := m.renderHeader("Config", 0, 0) + "\n"
	s += borderStyle.Render("\u250c"+strings.Repeat("\u2500", iw)+"\u2510") + "\n"

	settingCol := 22
	s += borderedRow("", iw, normalRowStyle) + "\n"
	s += borderedRow(fmt.Sprintf("  %-*s  %s", settingCol, "SETTING", "VALUE"), iw, colHeaderStyle) + "\n"
	s += borderStyle.Render("\u251c"+strings.Repeat("\u2500", iw)+"\u2524") + "\n"
	s += borderedRow(fmt.Sprintf("    %-*s  %s", settingCol, "Fleet directory", m.appCfg.FleetDir), iw, normalRowStyle) + "\n"
	s += borderedRow(fmt.Sprintf("    %-*s  %s", settingCol, "Editor", m.appCfg.Editor()), iw, normalRowStyle) + "\n"
	s += borderedRow("", iw, normalRowStyle) + "\n"

	s = m.padToBottom(s, iw)
	s += borderStyle.Render("\u2514"+strings.Repeat("\u2500", iw)+"\u2518") + "\n"
	s += m.renderHintBar(hintWithHelp([][]string{
		{"e", "Edit config"},
		{"r", "Reload"},
		{"Esc", "Back"},
	}))
	return s
}

func (m Model) handleConfigKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "e":
		return m, m.editConfigFile()
	case "r":
		newCfg, err := config.LoadAppConfig(config.ConfigPath())
		if err != nil {
			m.flash = fmt.Sprintf("Reload failed: %v", err)
			m.flashError = true
			return m, nil
		}
		newFleets, err := config.ScanFleets(newCfg.FleetDir)
		if err != nil {
			m.flash = fmt.Sprintf("Fleet scan failed: %v", err)
			m.flashError = true
			return m, nil
		}
		m.appCfg = newCfg
		m.fleets = newFleets
		if m.fleetCursor >= len(m.fleets) {
			m.fleetCursor = max(0, len(m.fleets)-1)
		}
		m.flash = "Config reloaded"
		m.view = viewFleetPicker
	case "esc":
		m.view = viewFleetPicker
	}
	return m, nil
}
