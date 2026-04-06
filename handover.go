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

// sshHandoverFinishedMsg is sent when a terminal handover SSH command completes.
type sshHandoverFinishedMsg struct {
	Err error
}

// editFinishedMsg is sent when the editor returns.
type editFinishedMsg struct {
	Err error
}

// sshExec wraps an SSH command with terminal handover.
type sshExec struct {
	host   string
	user   string
	port   int
	args   []string
	banner string
	err    error
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

	return nil
}

func (s *sshExec) SetStdin(_ io.Reader)  {}
func (s *sshExec) SetStdout(_ io.Writer) {}
func (s *sshExec) SetStderr(_ io.Writer) {}

// editorExec wraps an editor command for terminal handover.
type editorExec struct {
	path string
	err  error
}

func (e *editorExec) Run() error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, e.path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	e.err = c.Run()
	return nil
}

func (e *editorExec) SetStdin(_ io.Reader)  {}
func (e *editorExec) SetStdout(_ io.Writer) {}
func (e *editorExec) SetStderr(_ io.Writer) {}

// editFleetFile opens the selected fleet file in the user's editor.
func (m model) editFleetFile() tea.Cmd {
	f := m.fleets[m.fleetCursor]
	e := &editorExec{path: f.Path}
	return tea.Exec(e, func(err error) tea.Msg {
		return editFinishedMsg{Err: e.err}
	})
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
		return sshHandoverFinishedMsg{Err: e.err}
	})
}
