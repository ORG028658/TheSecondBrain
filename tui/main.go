package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// ── Flags ─────────────────────────────────────────────────────────────────
	// --current-dir: treat the project root as the raw source directory for this
	// session. Equivalent to running /pull --current-dir on every pull. Files are
	// ingested from their actual locations; the sources: frontmatter in wiki pages
	// will reflect real paths (e.g. src/main.go) rather than raw/src/main.go.
	//
	// Useful when pointing the brain at an existing codebase or project directory
	// without copying files into raw/ first.
	//
	// The /pull --current-dir TUI command applies the same override for a single
	// pull operation without restarting the session.
	useCurrentDir := flag.Bool("current-dir", false,
		"Use the project root as the raw source directory instead of raw/")
	flag.Parse()

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

	// --current-dir: override the raw source path to the project root.
	// The ingest pipeline will walk projectPath instead of projectPath/raw/.
	// wiki/ and knowledge-base/ remain in their default locations.
	if *useCurrentDir {
		cfg.Paths.Raw = projectPath
	}

	model := ui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
