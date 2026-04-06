package main

// Bridge file — delegates to internal/ssh.
// Will be removed when app moves to internal/.

import (
	tea "github.com/charmbracelet/bubbletea"

	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

var isAuthError = issh.IsAuthError

// retryWithPassword attempts to connect a specific host using password auth.
func retryWithPassword(sm *sshManager, idx int, h host, password string) tea.Cmd {
	return func() tea.Msg {
		return sm.ConnectWithPassword(idx, h, password)
	}
}
