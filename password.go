package main

import (
	tea "github.com/charmbracelet/bubbletea"

	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// passwordRetryResult is sent after retrying connection with a password.
type passwordRetryResult struct {
	index int
	info  probeInfo
	err   error
}

// isAuthError delegates to internal/ssh.IsAuthError.
var isAuthError = issh.IsAuthError

// retryWithPassword attempts to connect a specific host using password auth.
func (sm *sshManager) retryWithPassword(idx int, h host, password string) tea.Cmd {
	return func() tea.Msg {
		return sm.connectWithPassword(idx, h, password)
	}
}
