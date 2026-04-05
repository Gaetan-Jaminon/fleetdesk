package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// sshExec wraps an SSH command with terminal handover.
type sshExec struct {
	host    string
	user    string
	port    int
	args    []string
	banner  string
	err     error
}

func (s *sshExec) Run() error {
	sep := strings.Repeat("━", 50)
	fmt.Printf("\n%s\n▶ %s\n%s\n\n", sep, s.banner, sep)

	sshArgs := []string{
		"-t",
		"-o", "StrictHostKeyChecking=no",
		"-p", fmt.Sprintf("%d", s.port),
		fmt.Sprintf("%s@%s", s.user, s.host),
	}
	sshArgs = append(sshArgs, s.args...)

	c := exec.Command("ssh", sshArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	s.err = c.Run()

	status := "✓ done"
	if s.err != nil {
		status = fmt.Sprintf("✗ %v", s.err)
	}
	fmt.Printf("\n%s\n%s\nPress Enter to return to fleetdesk...", sep, status)
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	return nil // always return nil so TUI resumes
}

func (s *sshExec) SetStdin(_ io.Reader)  {}
func (s *sshExec) SetStdout(_ io.Writer) {}
func (s *sshExec) SetStderr(_ io.Writer) {}

// sshHandoverFinishedMsg is sent when a terminal handover SSH command completes.
type sshHandoverFinishedMsg struct {
	err error
}

// sshHandover creates a tea.Cmd for terminal handover to an SSH command.
func sshHandover(h host, args []string, banner string) tea.Cmd {
	e := &sshExec{
		host:   h.Entry.Hostname,
		user:   h.Entry.User,
		port:   h.Entry.Port,
		args:   args,
		banner: banner,
	}
	return tea.Exec(e, func(err error) tea.Msg {
		return sshHandoverFinishedMsg{err: e.err}
	})
}
