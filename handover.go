package main

// Bridge file — delegates to internal/ssh.
// Will be removed when app moves to internal/.

import (
	issh "github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

type sshExec = issh.SSHExec
type editorExec = issh.EditorExec
type sshHandoverFinishedMsg = issh.SSHHandoverFinishedMsg
type editFinishedMsg = issh.EditFinishedMsg

var sshHandover = issh.Handover
