package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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

	m := newModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
