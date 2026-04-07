package main

// Bridge file — delegates to internal/ssh.
// Will be removed when app moves to internal/.

import (
	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

type sshManager = issh.Manager
type hostProbeResult = issh.HostProbeResult
type passwordRetryResult = issh.PasswordRetryResult

var newSSHManager = issh.NewManager
