package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// passwordRetryResult is sent after retrying connection with a password.
type passwordRetryResult struct {
	index int
	info  probeInfo
	err   error
}

// isAuthError checks if an error is an SSH authentication failure.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unable to authenticate") ||
		strings.Contains(s, "no supported methods remain") ||
		strings.Contains(s, "handshake failed")
}

// retryWithPassword attempts to connect a specific host using password auth.
func (sm *sshManager) retryWithPassword(idx int, h host, password string) tea.Cmd {
	return func() tea.Msg {
		return sm.connectWithPassword(idx, h, password)
	}
}
