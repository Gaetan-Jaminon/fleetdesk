package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/app"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("fleetdesk %s (%s)\n", version, commit)
		os.Exit(0)
	}

	fleets, err := config.ScanFleets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning fleets: %v\n", err)
		os.Exit(1)
	}

	m := app.NewModel(fleets)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
