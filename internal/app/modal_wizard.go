package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

// wizardCompleteMsg is sent when the first-run wizard finishes.
type wizardCompleteMsg struct {
	appCfg config.AppConfig
	fleets []config.Fleet
}

// wizardCancelMsg is sent when the user cancels the wizard.
type wizardCancelMsg struct{}

// wizardNeedCustomEditorMsg triggers a follow-up modal for custom editor input.
type wizardNeedCustomEditorMsg struct {
	fleetDir string
}

// NewFirstRunWizard creates the 2-step first-run setup modal.
// If user selects "custom" editor, a follow-up modal is triggered via wizardNeedCustomEditorMsg.
func NewFirstRunWizard() *ModalOverlay {
	return newWizardSteps()
}

func newWizardSteps() *ModalOverlay {
	editorOptions := []string{"vim", "neovim", "nano", "custom"}

	steps := []ModalStep{
		{
			Title: "Enter the path to your fleet files directory",
			Content: NewTextInputContent("Fleet directory:", func(s string) error {
				return config.ValidateFleetDir(s)
			}),
		},
		{
			Title:   "Select your preferred editor",
			Content: NewSelectContent("Editor:", editorOptions),
		},
	}

	return NewModalOverlay("FleetDesk Setup", steps,
		func(results []any) tea.Cmd {
			fleetDir := results[0].(string)
			editorChoice := results[1].(string)

			if editorChoice == "custom" {
				return func() tea.Msg {
					return wizardNeedCustomEditorMsg{fleetDir: fleetDir}
				}
			}

			editor := editorChoice
			if editorChoice == "neovim" {
				editor = "nvim"
			}

			return finalizeWizard(fleetDir, editor)
		},
		cancelWizard,
	)
}

// newCustomEditorWizard creates a 1-step modal for custom editor input.
func newCustomEditorWizard(fleetDir string) *ModalOverlay {
	steps := []ModalStep{
		{
			Title:   "Enter your editor command",
			Content: NewTextInputContent("Editor command:", nil),
		},
	}

	return NewModalOverlay("FleetDesk Setup", steps,
		func(results []any) tea.Cmd {
			editor := results[0].(string)
			return finalizeWizard(fleetDir, editor)
		},
		cancelWizard,
	)
}

func finalizeWizard(fleetDir, editor string) tea.Cmd {
	return func() tea.Msg {
		configDir := config.ConfigPath()
		if err := config.WriteDefaultAppConfig(configDir, fleetDir, editor); err != nil {
			return wizardCancelMsg{}
		}

		appCfg, err := config.LoadAppConfig(configDir)
		if err != nil {
			return wizardCancelMsg{}
		}

		fleets, err := config.ScanFleets(appCfg.FleetDir)
		if err != nil {
			return wizardCancelMsg{}
		}

		return wizardCompleteMsg{appCfg: appCfg, fleets: fleets}
	}
}

func cancelWizard() tea.Cmd {
	return func() tea.Msg {
		return wizardCancelMsg{}
	}
}
