package main

// Bridge file — delegates to internal/ssh.
// Will be removed when app moves to internal/.

import (
	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

type probeInfo = issh.ProbeInfo

var formatDateEU = issh.FormatDateEU
