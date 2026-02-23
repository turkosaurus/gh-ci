package main

import (
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/ui"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if we have any repos to watch
	if len(cfg.Repos) == 0 {
		fmt.Fprintln(os.Stderr, "No repositories configured.")
		fmt.Fprintln(os.Stderr, "Run this command in a git repository with a GitHub remote,")
		fmt.Fprintln(os.Stderr, "or create a config file at ~/.config/gh-ci/config.yml:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "repos:")
		fmt.Fprintln(os.Stderr, "  - owner/repo")
		fmt.Fprintln(os.Stderr, "refresh_interval: 30")
		os.Exit(1)
	}

	logger, err := newFileLogger()
	if err != nil {
		fmt.Sprintf("fatal: cannot initialize logger: %v", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	model, err := ui.NewModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: cannot initialize model: %v\n", err)
		os.Exit(1)
	}
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: run: %v\n", err)
		os.Exit(1)
	}
}
