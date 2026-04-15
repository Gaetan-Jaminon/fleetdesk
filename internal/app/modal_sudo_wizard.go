package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/ssh"
)

// sudoWizardResultMsg is sent after the sudo wizard tests the password.
type sudoWizardResultMsg struct {
	hostIdx int
	success bool
}

// NewSudoWizard creates a modal that asks for the sudo password and tests it.
// On success it sends sudoWizardResultMsg{success: true}.
// On cancel it sends sudoWizardResultMsg{success: false}.
func NewSudoWizard(user, host string, hostIdx int, sshMgr *ssh.Manager) *ModalOverlay {
	prompt := fmt.Sprintf("Sudo password for %s@%s:", user, host)
	m := NewModalOverlay("Sudo Authentication", []ModalStep{
		{Title: prompt, Content: NewMaskedTextInputContent(prompt)},
	}, func(results []any) tea.Cmd {
		pw := results[0].(string)
		idx := hostIdx
		sm := sshMgr
		return func() tea.Msg {
			// Test the password
			sm.SetSudoPassword(idx, pw)
			out, err := sm.RunSudoCommand(idx, "sudo true")
			sm.SetSudoPassword(idx, "") // clear — Update handler sets on success
			success := err == nil && !ssh.IsSudoOutput(out)
			if success {
				return sudoWizardResultMsg{hostIdx: idx, success: true}
			}
			// Wrong password — send failure
			return sudoWizardResultMsg{hostIdx: idx, success: false}
		}
	}, func() tea.Cmd {
		idx := hostIdx
		return func() tea.Msg {
			return sudoWizardResultMsg{hostIdx: idx, success: false}
		}
	})
	return m
}
