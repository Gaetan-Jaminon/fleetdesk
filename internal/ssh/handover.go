package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

// SSHHandoverFinishedMsg is sent when a terminal handover SSH command completes.
type SSHHandoverFinishedMsg struct {
	Err error
}

// EditFinishedMsg is sent when the editor returns.
type EditFinishedMsg struct {
	Err error
}

// SSHExec wraps an SSH command with terminal handover.
type SSHExec struct {
	host   string
	user   string
	port   int
	args   []string
	banner string
	err    error
}

// Run executes the SSH command with terminal handover.
func (s *SSHExec) Run() error {
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

// SetStdin implements tea.ExecCommand.
func (s *SSHExec) SetStdin(_ io.Reader) {}

// SetStdout implements tea.ExecCommand.
func (s *SSHExec) SetStdout(_ io.Writer) {}

// SetStderr implements tea.ExecCommand.
func (s *SSHExec) SetStderr(_ io.Writer) {}

// EditorExec wraps an editor command for terminal handover.
type EditorExec struct {
	path string
	err  error
}

// Run executes the editor.
func (e *EditorExec) Run() error {
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

// SetStdin implements tea.ExecCommand.
func (e *EditorExec) SetStdin(_ io.Reader) {}

// SetStdout implements tea.ExecCommand.
func (e *EditorExec) SetStdout(_ io.Writer) {}

// SetStderr implements tea.ExecCommand.
func (e *EditorExec) SetStderr(_ io.Writer) {}

// Handover creates a tea.Cmd for terminal handover to an SSH command.
func Handover(h config.Host, args []string, banner string) tea.Cmd {
	e := &SSHExec{
		host:   h.Entry.Hostname,
		user:   h.Entry.User,
		port:   h.Entry.Port,
		args:   args,
		banner: banner,
	}
	return tea.Exec(e, func(err error) tea.Msg {
		return SSHHandoverFinishedMsg{Err: e.err}
	})
}

// EditFile opens a file in the user's editor via terminal handover.
func EditFile(path string) tea.Cmd {
	e := &EditorExec{path: path}
	return tea.Exec(e, func(err error) tea.Msg {
		return EditFinishedMsg{Err: e.err}
	})
}
