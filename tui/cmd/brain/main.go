package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// ── Flags ─────────────────────────────────────────────────────────────────
	useCurrentDir := flag.Bool("current-dir", false,
		"Use the project root as the raw source directory instead of raw/")
	uninstall := flag.Bool("uninstall", false,
		"Remove the brain binary and global config, then exit")
	flag.Parse()

	if *uninstall {
		runUninstall()
		return
	}

	// Project path = wherever brain was launched from.
	projectPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current directory:", err)
		os.Exit(1)
	}

	// Load API keys from global config dir (~/.config/secondbrain/.env)
	godotenv.Load(config.EnvPath()) //nolint:errcheck

	if config.IsFirstRun() {
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
		godotenv.Load(config.EnvPath()) //nolint:errcheck
	}

	cfg, err := config.Load(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n\nFix: delete %s and re-run 'brain'.\n",
			err, config.ConfigDir())
		os.Exit(1)
	}

	// --current-dir: override the raw source path to the project root.
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

// runUninstall removes the brain binary and global config directory,
// and cleans up the PATH line the installer added to shell rc files.
func runUninstall() {
	fmt.Println()
	fmt.Println("  ◆  TheSecondBrain — uninstall")
	fmt.Println()

	// ── Confirm ───────────────────────────────────────────────────────────────
	fmt.Print("  This will remove the brain binary and ~/.config/secondbrain/.\n")
	fmt.Print("  Your vault data (raw/, wiki/, knowledge-base/) will NOT be touched.\n\n")
	fmt.Print("  Continue? [y/N]: ")

	var reply string
	fmt.Scanln(&reply)
	if strings.ToLower(strings.TrimSpace(reply)) != "y" {
		fmt.Println("  Cancelled.")
		return
	}
	fmt.Println()

	ok := true

	// ── Remove config directory ───────────────────────────────────────────────
	cfgDir := config.ConfigDir()
	if err := os.RemoveAll(cfgDir); err != nil {
		fmt.Fprintf(os.Stderr, "  ✗  Failed to remove config dir %s: %v\n", cfgDir, err)
		ok = false
	} else {
		fmt.Printf("  ✓  Removed config: %s\n", cfgDir)
	}

	// ── Remove binary ─────────────────────────────────────────────────────────
	// os.Executable() returns the resolved path of the running binary.
	binPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗  Could not locate brain binary: %v\n", err)
		ok = false
	} else {
		binPath, _ = filepath.EvalSymlinks(binPath)
		if err := os.Remove(binPath); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗  Failed to remove binary %s: %v\n", binPath, err)
			fmt.Fprintf(os.Stderr, "     Try: sudo rm %s\n", binPath)
			ok = false
		} else {
			fmt.Printf("  ✓  Removed binary: %s\n", binPath)
		}
	}

	// ── Clean PATH line from shell rc files ───────────────────────────────────
	home, _ := os.UserHomeDir()
	rcFiles := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".bashrc"),
	}

	for _, rc := range rcFiles {
		if cleaned, changed := removePathLine(rc); changed {
			if err := os.WriteFile(rc, []byte(cleaned), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗  Could not clean PATH from %s: %v\n", rc, err)
			} else {
				fmt.Printf("  ✓  Cleaned PATH from: %s\n", rc)
			}
		}
	}

	// ── Done ──────────────────────────────────────────────────────────────────
	fmt.Println()
	if ok {
		fmt.Println("  ✓  Uninstall complete.")
	} else {
		fmt.Println("  ⚠  Uninstall finished with some errors (see above).")
	}
	fmt.Println()
}

// removePathLine strips the PATH export line and its comment that the
// installer wrote into a shell rc file. Returns the cleaned content and
// whether anything changed.
func removePathLine(rcFile string) (string, bool) {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return "", false // file doesn't exist — nothing to do
	}

	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	changed := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// Skip the comment line the installer wrote
		if strings.TrimSpace(line) == "# Added by TheSecondBrain installer" {
			changed = true
			continue
		}
		// Skip the PATH export line for either install location
		if strings.Contains(line, ".local/bin") && strings.Contains(line, "PATH") &&
			strings.Contains(line, "TheSecondBrain") ||
			(strings.Contains(line, "export PATH") &&
				(strings.Contains(line, ".local/bin:$PATH") ||
					strings.Contains(line, "go/bin:$PATH"))) {
			// Only skip if it looks like ours (contains our install dirs)
			if strings.Contains(line, ".local/bin") || strings.Contains(line, "go/bin") {
				changed = true
				continue
			}
		}
		out = append(out, line)
	}

	// Collapse any run of 3+ blank lines left by removal into 2
	cleaned := collapseBlankLines(strings.Join(out, "\n"))
	return cleaned, changed
}

// collapseBlankLines reduces consecutive blank lines to at most two.
func collapseBlankLines(s string) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var b strings.Builder
	blanks := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks <= 2 {
				b.WriteString(line + "\n")
			}
		} else {
			blanks = 0
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
