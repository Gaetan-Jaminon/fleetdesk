package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

func makeTestCommands() []config.CommandEntry {
	return []config.CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -h"},
		{Name: "clear-cache", Group: "OS", Run: "sudo dnf clean all"},
		{Name: "migrate", Group: "AAP", Run: "sudo -u awx awx-manage migrate"},
		{Name: "restart-all", Group: "AAP", Run: "sudo supervisorctl restart all"},
	}
}

func makeSingleGroupCommands() []config.CommandEntry {
	return []config.CommandEntry{
		{Name: "check-disk", Group: "OS", Run: "df -h"},
		{Name: "clear-cache", Group: "OS", Run: "sudo dnf clean all"},
	}
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestCommandWizard_3Step_Confirm(t *testing.T) {
	cmds := makeTestCommands()
	var streamedCmd config.CommandEntry
	var streamCalled bool

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd {
			streamCalled = true
			streamedCmd = cmd
			return nil
		},
		func() tea.Cmd { return nil },
	)

	// Step 1: group picker — should have AAP and OS (alphabetical)
	if len(wizard.steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(wizard.steps))
	}

	// Groups are alphabetical: AAP (0), OS (1)
	// Select AAP (cursor=0), press Enter
	wizard.HandleKey(keyMsg("enter"))

	// Step 2: command picker — AAP commands: migrate, restart-all (alphabetical)
	// Select migrate (cursor=0), press Enter
	wizard.HandleKey(keyMsg("enter"))

	// Step 3: confirm — press Y
	wizard.HandleKey(keyMsg("y"))

	if !wizard.Done() {
		t.Error("wizard should be done")
	}
	if !streamCalled {
		t.Error("onStream should have been called")
	}
	if streamedCmd.Name != "migrate" {
		t.Errorf("expected 'migrate', got %q", streamedCmd.Name)
	}
	if streamedCmd.Run != "sudo -u awx awx-manage migrate" {
		t.Errorf("unexpected run: %q", streamedCmd.Run)
	}
}

func TestCommandWizard_3Step_SelectSecondGroup(t *testing.T) {
	cmds := makeTestCommands()
	var streamedCmd config.CommandEntry

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd {
			streamedCmd = cmd
			return nil
		},
		func() tea.Cmd { return nil },
	)

	// Move down to OS (index 1), Enter
	wizard.HandleKey(keyMsg("down"))
	wizard.HandleKey(keyMsg("enter"))

	// OS commands: check-disk, clear-cache (alphabetical)
	// Select clear-cache (down once), Enter
	wizard.HandleKey(keyMsg("down"))
	wizard.HandleKey(keyMsg("enter"))

	// Confirm
	wizard.HandleKey(keyMsg("y"))

	if streamedCmd.Name != "clear-cache" {
		t.Errorf("expected 'clear-cache', got %q", streamedCmd.Name)
	}
}

func TestCommandWizard_3Step_BackNavigation(t *testing.T) {
	cmds := makeTestCommands()
	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Step 1: select AAP, Enter
	wizard.HandleKey(keyMsg("enter"))

	// Step 2: Esc → back to step 1
	wizard.HandleKey(keyMsg("esc"))

	if wizard.current != 0 {
		t.Errorf("expected back to step 0, got %d", wizard.current)
	}
	if wizard.Done() {
		t.Error("wizard should not be done after back")
	}
}

func TestCommandWizard_3Step_EscFromConfirm(t *testing.T) {
	cmds := makeTestCommands()
	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Step 1: Enter (AAP)
	wizard.HandleKey(keyMsg("enter"))
	// Step 2: Enter (migrate)
	wizard.HandleKey(keyMsg("enter"))
	// Step 3: Esc → back to step 2
	wizard.HandleKey(keyMsg("esc"))

	if wizard.current != 1 {
		t.Errorf("expected back to step 1, got %d", wizard.current)
	}
}

func TestCommandWizard_3Step_CancelFromGroupPicker(t *testing.T) {
	cmds := makeTestCommands()
	var cancelCalled bool

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd {
			cancelCalled = true
			return nil
		},
	)

	// Esc at step 0 → cancel
	wizard.HandleKey(keyMsg("esc"))

	if !wizard.Done() {
		t.Error("wizard should be done after cancel")
	}
	if !cancelCalled {
		t.Error("onCancel should have been called")
	}
}

func TestCommandWizard_2Step_Confirm(t *testing.T) {
	cmds := makeSingleGroupCommands()
	var streamedCmd config.CommandEntry

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd {
			streamedCmd = cmd
			return nil
		},
		func() tea.Cmd { return nil },
	)

	if len(wizard.steps) != 2 {
		t.Fatalf("expected 2 steps for single group, got %d", len(wizard.steps))
	}

	// Step 1: command picker — check-disk (0), clear-cache (1)
	// Select check-disk, Enter
	wizard.HandleKey(keyMsg("enter"))

	// Step 2: confirm — Y
	wizard.HandleKey(keyMsg("y"))

	if !wizard.Done() {
		t.Error("wizard should be done")
	}
	if streamedCmd.Name != "check-disk" {
		t.Errorf("expected 'check-disk', got %q", streamedCmd.Name)
	}
}

func TestCommandWizard_2Step_Cancel(t *testing.T) {
	cmds := makeSingleGroupCommands()
	var cancelCalled bool

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd {
			cancelCalled = true
			return nil
		},
	)

	// Esc at step 0 → cancel (no group step to go back to)
	wizard.HandleKey(keyMsg("esc"))

	if !wizard.Done() {
		t.Error("wizard should be done after cancel")
	}
	if !cancelCalled {
		t.Error("onCancel should have been called")
	}
}

func TestCommandWizard_2Step_EscFromConfirm(t *testing.T) {
	cmds := makeSingleGroupCommands()
	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Enter → confirm step
	wizard.HandleKey(keyMsg("enter"))
	// Esc → back to command picker
	wizard.HandleKey(keyMsg("esc"))

	if wizard.current != 0 {
		t.Errorf("expected back to step 0, got %d", wizard.current)
	}
}

func TestCommandWizard_ConfirmShowsRunString(t *testing.T) {
	cmds := makeSingleGroupCommands()
	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd { return nil },
	)

	// Select check-disk, Enter
	wizard.HandleKey(keyMsg("enter"))

	// Confirm step should show the run string
	confirmStep := wizard.steps[len(wizard.steps)-1]
	view := confirmStep.Content.View(80)
	if view == "" {
		t.Error("confirm view should not be empty")
	}
	// The ConfirmContent message should contain "df -h"
	cc := confirmStep.Content.(*ConfirmContent)
	if cc.message != "df -h" {
		t.Errorf("confirm message should be 'df -h', got %q", cc.message)
	}
}

func TestCommandWizard_ConfirmNRejectsAndCancels(t *testing.T) {
	cmds := makeSingleGroupCommands()
	var cancelCalled bool

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd { return nil },
		func() tea.Cmd {
			cancelCalled = true
			return nil
		},
	)

	// Enter → confirm
	wizard.HandleKey(keyMsg("enter"))
	// N → reject
	wizard.HandleKey(keyMsg("n"))

	if !wizard.Done() {
		t.Error("wizard should be done after N")
	}
	if !cancelCalled {
		t.Error("onCancel should have been called on N")
	}
}

func TestDeriveGroups_Alphabetical(t *testing.T) {
	cmds := []config.CommandEntry{
		{Name: "a", Group: "Zebra", Run: "echo z"},
		{Name: "b", Group: "Alpha", Run: "echo a"},
		{Name: "c", Group: "Zebra", Run: "echo z2"},
	}
	groups := deriveGroups(cmds)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0] != "Alpha" || groups[1] != "Zebra" {
		t.Errorf("expected [Alpha, Zebra], got %v", groups)
	}
}

func TestCommandWizard_3Step_GroupChangeResetsCommandList(t *testing.T) {
	cmds := makeTestCommands()
	var streamedCmd config.CommandEntry

	wizard := NewCommandWizard("test-host", cmds,
		func(cmd config.CommandEntry) tea.Cmd {
			streamedCmd = cmd
			return nil
		},
		func() tea.Cmd { return nil },
	)

	// Select AAP, Enter
	wizard.HandleKey(keyMsg("enter"))
	// Back to group picker
	wizard.HandleKey(keyMsg("esc"))
	// Select OS (down), Enter
	wizard.HandleKey(keyMsg("down"))
	wizard.HandleKey(keyMsg("enter"))
	// Select first OS command (check-disk), Enter
	wizard.HandleKey(keyMsg("enter"))
	// Confirm
	wizard.HandleKey(keyMsg("y"))

	if streamedCmd.Group != "OS" {
		t.Errorf("expected OS group after re-pick, got %q", streamedCmd.Group)
	}
}
