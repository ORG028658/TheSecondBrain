package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── phases ────────────────────────────────────────────────────────────────────

type setupPhase int

const (
	phaseWelcome  setupPhase = iota
	phaseAPIKey              // enter Rakuten API key
	phaseCreating            // creating config + vault structure
	phaseDone                // animated completion
)

// ── messages ──────────────────────────────────────────────────────────────────

type setupDoneMsg struct{ err error }
type setupTickMsg struct{}

// ── model ────────────────────────────────────────────────────────────────────

type SetupModel struct {
	phase       setupPhase
	keyInput    textinput.Model
	spinner     spinner.Model
	progress    []string
	completed   bool
	err         error
	width       int
	height      int
	tickCount   int
	projectPath string // CWD — vault structure created here
}

func NewSetupModel(projectPath string) SetupModel {
	ki := textinput.New()
	ki.Placeholder = "sk-... or your provider's API key"
	ki.EchoMode = textinput.EchoPassword
	ki.EchoCharacter = '•'
	ki.Width = 52

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return SetupModel{
		phase:       phaseWelcome,
		keyInput:    ki,
		spinner:     sp,
		projectPath: projectPath,
	}
}

func (m SetupModel) Completed() bool { return m.completed }

func (m SetupModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			switch m.phase {
			case phaseWelcome:
				m.phase = phaseAPIKey
				m.keyInput.Focus()

			case phaseAPIKey:
				key := strings.TrimSpace(m.keyInput.Value())
				if key == "" {
					break
				}
				m.phase = phaseCreating
				m.keyInput.Blur()
				return m, m.runSetup(key)

			case phaseDone:
				m.completed = true
				return m, tea.Quit
			}

		case tea.KeyEsc:
			if m.phase == phaseAPIKey {
				m.phase = phaseWelcome
				m.keyInput.Blur()
			}
		}

	case setupDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = phaseAPIKey
		} else {
			m.phase = phaseDone
			cmds = append(cmds, tickCmd())
		}

	case setupTickMsg:
		m.tickCount++
		if m.tickCount < 50 {
			cmds = append(cmds, tickCmd())
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m SetupModel) View() string {
	if m.width == 0 {
		return ""
	}
	switch m.phase {
	case phaseWelcome:
		return m.viewWelcome()
	case phaseAPIKey:
		return m.viewAPIKey()
	case phaseCreating:
		return m.viewCreating()
	case phaseDone:
		return m.viewDone()
	}
	return ""
}

// ── views ─────────────────────────────────────────────────────────────────────

func (m SetupModel) viewWelcome() string {
	logo := renderLogo(m.width)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(1, 4).
		Width(min(m.width-4, 58)).
		Align(lipgloss.Center)

	body := lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa")).Render(
		"A personal knowledge vault that grows smarter\n"+
			"every time you add something.\n\n"+
			"brain operates in the current directory:\n") +
		lipgloss.NewStyle().Foreground(colorGreen).Render(m.projectPath) +
		"\n\n" +
		helpStyle.Render("Press Enter to continue  ·  Ctrl+C to quit")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, logo, "\n",
			box.Render(lipgloss.NewStyle().Foreground(colorWhite).Bold(true).Render("Welcome to TheSecondBrain")+"\n\n"+body)))
}

func (m SetupModel) viewAPIKey() string {
	label := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("LLM API key")
	hint := lipgloss.NewStyle().Foreground(colorGray).Render(
		"Stored in:  " + config.EnvPath() + "\n" +
			"Never committed to any project.")

	var errMsg string
	if m.err != nil {
		errMsg = "\n" + errorMsgStyle.Render("⚠ "+m.err.Error())
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(1, 3).
		Width(min(m.width-4, 62))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		box.Render(label+"\n\n"+hint+"\n\n"+m.keyInput.View()+errMsg+"\n\n"+
			helpStyle.Render("Enter to confirm  ·  Esc to go back")))
}

func (m SetupModel) viewCreating() string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("Setting up..."))
	lines = append(lines, "")
	for _, p := range m.progress {
		lines = append(lines, systemMsgStyle.Render("  "+p))
	}
	lines = append(lines, "")
	lines = append(lines, m.spinner.View()+" "+systemMsgStyle.Render("Working..."))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(1, 3).
		Width(min(m.width-4, 62))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		box.Render(strings.Join(lines, "\n")))
}

func (m SetupModel) viewDone() string {
	full := "✓  Setup complete!"
	visible := full
	if m.tickCount < len([]rune(full)) {
		visible = string([]rune(full)[:m.tickCount])
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(1, 4).
		Width(min(m.width-4, 62)).
		Align(lipgloss.Center)

	body := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render(visible) + "\n\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa")).Render(
			"Config : "+config.ConfigDir()+"\n"+
				"Project: "+m.projectPath+"\n\n"+
				"Drop files into  raw/  and run /pull\nto start building your knowledge base.",
		) + "\n\n" +
		helpStyle.Render("Press Enter to launch TheSecondBrain")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, renderLogo(m.width), "\n", box.Render(body)))
}

// ── setup execution ───────────────────────────────────────────────────────────

func (m *SetupModel) runSetup(apiKey string) tea.Cmd {
	projectPath := m.projectPath
	return func() tea.Msg {
		// 1. Save global config
		if err := config.SaveNew(); err != nil {
			return setupDoneMsg{err: fmt.Errorf("saving config: %w", err)}
		}

		// 2. Save API key
		if err := config.UpdateAPIKey(apiKey); err != nil {
			return setupDoneMsg{err: fmt.Errorf("saving API key: %w", err)}
		}

		// 3. Create vault structure in project dir
		dirs := []string{
			"raw",
			"wiki/sources", "wiki/entities", "wiki/concepts", "wiki/synthesis",
			"knowledge-base/embeddings", "knowledge-base/metadata", "knowledge-base/output",
		}
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(projectPath, d), 0755); err != nil {
				return setupDoneMsg{err: fmt.Errorf("creating %s: %w", d, err)}
			}
		}

		writeIfMissing(filepath.Join(projectPath, "wiki", "index.md"),
			"# Wiki Index\nLast updated: — | Pages: 0\n\n## Sources\n## Entities\n## Concepts\n## Synthesis\n")
		writeIfMissing(filepath.Join(projectPath, "wiki", "log.md"),
			"# Wiki Log\n\nAppend-only record of all wiki operations.\n\n---\n\n")
		writeIfMissing(filepath.Join(projectPath, "knowledge-base", "metadata", "sources.json"), "{}\n")

		return setupDoneMsg{}
	}
}

func writeIfMissing(path, content string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.WriteFile(path, []byte(content), 0644) //nolint:errcheck
	}
}

func tickCmd() tea.Cmd {
	return func() tea.Msg { return setupTickMsg{} }
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
