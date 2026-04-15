package main

import (
	"fmt"
	"os"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// Project path = wherever brain was launched from.
	// This is the working directory for all raw/, wiki/, knowledge-base/ operations.
	projectPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current directory:", err)
		os.Exit(1)
	}

	// Load API keys from global config dir (~/.config/secondbrain/.env)
	godotenv.Load(config.EnvPath()) //nolint:errcheck

	if config.IsFirstRun() {
		// ── First run: setup wizard (API key only — no project path needed) ───
		p := tea.NewProgram(ui.NewSetupModel(projectPath), tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Setup failed:", err)
			os.Exit(1)
		}
		sm, ok := result.(ui.SetupModel)
		if !ok || !sm.Completed() {
			fmt.Println("Setup cancelled.")
			return
		}
		// Reload env after setup wrote .env
		godotenv.Load(config.EnvPath()) //nolint:errcheck
	}

	// ── Normal run: open brain in the current directory ──────────────────────
	cfg, err := config.Load(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n\nFix: delete %s and re-run 'brain'.\n",
			err, config.ConfigDir())
		os.Exit(1)
	}

	model := ui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
