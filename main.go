package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/app"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/config"
	"github.com/Gaetan-Jaminon/fleetdesk/internal/logging"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	debug := false
	showVersion := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--debug":
			debug = true
		case "--version":
			showVersion = true
		}
	}

	if showVersion {
		fmt.Printf("fleetdesk %s (%s)\n", version, commit)
		os.Exit(0)
	}

	fleets, err := config.ScanFleets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning fleets: %v\n", err)
		os.Exit(1)
	}

	logger := logging.InitLogger(debug, logging.LogDir())
	defer logging.CloseAll()
	logger.Info("fleetdesk starting", "version", version, "debug", debug, "fleets", len(fleets))

	m := app.NewModel(fleets, logger, version)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
