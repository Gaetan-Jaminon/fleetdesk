package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpForView_AllViewsCovered(t *testing.T) {
	// Every view constant must return non-empty help text.
	views := []view{
		viewFleetPicker, viewHostList, viewMetrics, viewResourcePicker,
		viewServiceList, viewContainerList, viewCronList, viewLogLevelPicker,
		viewErrorLogList, viewUpdateList, viewDiskList, viewSubscription,
		viewAccountList, viewNetworkPicker, viewNetworkInterfaces,
		viewNetworkPorts, viewNetworkRoutes, viewNetworkFirewall,
		viewSecurityFailedLogins, viewSecuritySudo, viewSecuritySELinux,
		viewSecurityAudit, viewAzureSubList, viewAzureResourcePicker,
		viewAzureVMList, viewAzureVMDetail, viewAzureAKSList,
		viewAzureAKSDetail, viewK8sClusterList, viewK8sContextList,
		viewK8sResourcePicker, viewK8sNodeList, viewK8sNodeDetail,
		viewK8sNamespaceList, viewK8sWorkloadList, viewK8sPodList,
		viewK8sPodDetail, viewK8sPodLogs, viewConfig,
	}
	for _, v := range views {
		text := helpForView(v)
		if text == "" {
			t.Errorf("helpForView(%d) returned empty string", v)
		}
	}
}

func TestHelpForView_ContainsGlobalSection(t *testing.T) {
	views := []view{
		viewFleetPicker, viewHostList, viewServiceList,
		viewAzureVMList, viewK8sPodList, viewConfig,
	}
	for _, v := range views {
		text := helpForView(v)
		if !strings.Contains(text, "Global") {
			t.Errorf("helpForView(%d) missing Global section", v)
		}
		if !strings.Contains(text, "Quit") {
			t.Errorf("helpForView(%d) missing q/Quit in Global section", v)
		}
		if !strings.Contains(text, "Help") {
			t.Errorf("helpForView(%d) missing ?/Help in Global section", v)
		}
	}
}

func TestHelpForView_ContainsNavigationSection(t *testing.T) {
	// Views with explicit Navigation group
	views := []view{
		viewFleetPicker, viewHostList, viewMetrics,
		viewResourcePicker, viewSubscription,
	}
	for _, v := range views {
		text := helpForView(v)
		if !strings.Contains(text, "Navigation") {
			t.Errorf("helpForView(%d) missing Navigation section", v)
		}
	}
}

func TestHelpForView_MultiModeViewsShowBothModes(t *testing.T) {
	// Services has both list and detail modes
	text := helpForView(viewServiceList)
	if !strings.Contains(text, "List Mode") {
		t.Error("services help missing List Mode section")
	}
	if !strings.Contains(text, "Detail Mode") {
		t.Error("services help missing Detail Mode section")
	}
}

func TestHelpForView_FilterViewsShowFilter(t *testing.T) {
	// Views that support / filter should show it in Global section
	text := helpForView(viewServiceList)
	if !strings.Contains(text, "Filter") {
		t.Error("services help missing / Filter in Global section")
	}
}

func TestHintWithHelp(t *testing.T) {
	hints := [][]string{
		{"Enter", "Select"},
		{"Esc", "Back"},
	}
	result := hintWithHelp(hints)
	if len(result) != 3 {
		t.Fatalf("expected 3 hints, got %d", len(result))
	}
	last := result[len(result)-1]
	if last[0] != "?" || last[1] != "Help" {
		t.Errorf("last hint = %v, want [? Help]", last)
	}
	// Original slice should not be modified
	if len(hints) != 2 {
		t.Error("hintWithHelp should not modify original slice")
	}
}

func TestStaticContent_QuestionMarkDismisses(t *testing.T) {
	sc := NewStaticContent("some help text")
	_, _, done := sc.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !done {
		t.Error("? should dismiss StaticContent")
	}
}

func TestHelpOverlay_QuestionMarkOpensModal(t *testing.T) {
	// Simulate pressing ? in handleKey by constructing the modal directly
	// (same logic as handleKey)
	text := helpForView(viewHostList)
	modal := NewModalOverlay("Keybindings", []ModalStep{
		{Title: "", Content: NewStaticContent(text)},
	}, func(_ []any) tea.Cmd { return nil },
		func() tea.Cmd { return nil })

	if modal.Done() {
		t.Error("modal should not be done immediately")
	}

	// View should contain help text
	view := modal.View("", 100, 40)
	if !strings.Contains(view, "Keybindings") {
		t.Error("modal view should contain title")
	}
}

func TestHelpOverlay_QuestionMarkClosesModal(t *testing.T) {
	text := helpForView(viewHostList)
	modal := NewModalOverlay("Keybindings", []ModalStep{
		{Title: "", Content: NewStaticContent(text)},
	}, func(_ []any) tea.Cmd { return nil },
		func() tea.Cmd { return nil })

	// Press ? to close (StaticContent handles it)
	modal.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !modal.Done() {
		t.Error("? should dismiss the help overlay")
	}
}

func TestHelpOverlay_EscClosesModal(t *testing.T) {
	text := helpForView(viewHostList)
	modal := NewModalOverlay("Keybindings", []ModalStep{
		{Title: "", Content: NewStaticContent(text)},
	}, func(_ []any) tea.Cmd { return nil },
		func() tea.Cmd { return nil })

	modal.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !modal.Done() {
		t.Error("Esc should dismiss the help overlay")
	}
}
