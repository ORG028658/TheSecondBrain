package ui

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ORG028658/TheSecondBrain/tui/internal/analyzer"
	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/embeddings"
	"github.com/ORG028658/TheSecondBrain/tui/internal/rag"
	"github.com/ORG028658/TheSecondBrain/tui/internal/store"
	"github.com/ORG028658/TheSecondBrain/tui/internal/wiki"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// ── message types ─────────────────────────────────────────────────────────────

type resultMsg struct {
	content string
	err     error
	refs    []string
}

type progressMsg struct {
	content string
	next    tea.Cmd
}

type progressDoneMsg struct {
	summary string
	err     error
}

// wiki correction confirmation
type correctionReadyMsg struct {
	wikiPath        string
	newContent      string
	originalExcerpt string
	proposedChange  string
	analysis        string
	isConsistent    bool
	summary         string
}

// streaming
type streamChunkMsg struct {
	content string
	next    tea.Cmd
}
type streamDoneMsg struct{ refs []string }

// file watcher
type rawFileChangedMsg struct{}
type autoAnalyzeMsg struct{}

// ── app state ─────────────────────────────────────────────────────────────────

type appState int

const (
	stateIdle       appState = iota
	stateLoading             // spinner, input disabled
	stateStreaming           // live token output
	stateConfirming          // waiting for user to confirm a wiki update
)

type chatMsg struct {
	role    string
	content string
	at      time.Time
}

// ── model ─────────────────────────────────────────────────────────────────────

type Model struct {
	cfg      *config.Config
	store    *store.Store
	embedder *embeddings.Client
	wiki     *wiki.Wiki
	analyzer *analyzer.Analyzer
	rag      *rag.RAG

	viewport  viewport.Model
	textInput textinput.Model
	spinner   spinner.Model

	msgs      []chatMsg
	state     appState
	width     int
	height    int
	ready     bool
	loadingOp string

	streamingContent string // plain string — safe to copy (Builder panics when copied by value)
	pendingRefs      []string
	lastAnswer       string
	lastRefs         []string

	history   []string
	histIdx   int
	histDraft string

	watcher         *fsnotify.Watcher
	pendingAnalysis bool
	atBottom        bool

	cancelOp context.CancelFunc // cancels the current pull/analyze/query; nil when idle

	// conversation context (last N exchanges for follow-up awareness)
	convHistory []rag.ConvMsg

	// wiki correction flow
	pendingUpdate *wikiUpdateState
}

// wikiUpdateState holds a pending correction waiting for user confirmation.
type wikiUpdateState struct {
	wikiPath        string
	newContent      string
	originalExcerpt string // excerpt of the wiki at time of change
	proposedChange  string // user's correction text
	analysis        string // LLM consistency analysis
	isConsistent    bool   // false = contradicts current content
	summary         string // shown to user
}

// amendmentStatus for the audit record
const (
	amendApplied      = "applied"
	amendForceApplied = "force-applied"
)

func NewModel(cfg *config.Config) *Model {
	// Ensure all vault directories exist (auto-create on startup)
	ensureVaultStructure(cfg)

	storePath := filepath.Join(cfg.Paths.KnowledgeBase, "embeddings", "store.json")
	s, storeErr := store.New(storePath)
	if storeErr != nil {
		s = store.NewFresh(storePath)
	}

	embedder := embeddings.New(cfg.Embeddings.Model, cfg.Embeddings.BaseURL)
	wikiSvc := wiki.New(cfg.Paths.Wiki)
	anlzr := analyzer.New(cfg, wikiSvc)
	ragSvc := rag.New(s, embedder, cfg)

	ti := textinput.New()
	ti.Placeholder = "Ask anything, or type /help"
	ti.Focus()
	ti.CharLimit = 0

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	// Start file watcher on raw/
	watcher := startWatcher(cfg.Paths.Raw)

	m := &Model{
		cfg:       cfg,
		store:     s,
		embedder:  embedder,
		wiki:      wikiSvc,
		analyzer:  anlzr,
		rag:       ragSvc,
		spinner:   sp,
		textInput: ti,
		histIdx:   -1,
		watcher:   watcher,
		atBottom:  true, // auto-follow new messages by default
	}

	// Welcome: show tips if wiki is empty, stats otherwise
	pages := wikiSvc.PageCount()
	_, chunks := s.Stats()
	if pages == 0 {
		m.msgs = append(m.msgs, chatMsg{role: "system", content: tipsMessage(cfg.Paths.Raw), at: time.Now()})
	} else {
		m.msgs = append(m.msgs, chatMsg{
			role:    "system",
			content: fmt.Sprintf("wiki: %d pages  ·  kb: %d chunks  ·  %s", pages, chunks, randomGreeting()),
			at:      time.Now(),
		})
	}
	if storeErr != nil {
		m.msgs = append(m.msgs, chatMsg{
			role:    "error",
			content: fmt.Sprintf("⚠ Knowledge base was corrupted and has been reset — run /pull to rebuild.\n  (%v)", storeErr),
			at:      time.Now(),
		})
	}
	if nestedRoot := detectNestedWikiRoot(cfg.Paths.Wiki); nestedRoot != "" {
		m.msgs = append(m.msgs, chatMsg{
			role: "system",
			content: fmt.Sprintf(
				"Warning: detected nested wiki content at %s.\nThis usually came from the old wiki/wiki path bug. New writes use canonical wiki/... paths, but you should inspect or migrate the nested files before trusting mixed results.",
				nestedRoot,
			),
			at: time.Now(),
		})
	}
	return m
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.spinner.Tick}
	if m.watcher != nil {
		cmds = append(cmds, watchRaw(m.watcher))
	}
	return tea.Batch(cmds...)
}

// ── update ────────────────────────────────────────────────────────────────────

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	manualViewportScroll := false

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpH := m.height - 3 // header(1) + divider(1) + footer(1)
		if vpH < 4 {
			vpH = 4
		}
		if !m.ready {
			m.viewport = viewport.New(m.width, vpH)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpH
		}

	// ── file watcher ─────────────────────────────────────────────────────────
	case rawFileChangedMsg:
		if m.watcher != nil {
			cmds = append(cmds, watchRaw(m.watcher))
		}
		if !m.pendingAnalysis && m.state == stateIdle {
			m.pendingAnalysis = true
			m.addMsg("system", "New files detected in raw/ — auto-analyzing in 3 seconds…")
			cmds = append(cmds, deferred(3*time.Second, autoAnalyzeMsg{}))
		}

	case autoAnalyzeMsg:
		m.pendingAnalysis = false
		if m.state == stateIdle {
			m.state = stateLoading
			m.loadingOp = randomAnalyzingPhrase()
			return m, tea.Batch(append(cmds, m.cmdPullFrom(m.cfg.Paths.Raw), m.spinner.Tick)...)
		}

	// ── keyboard ─────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

		case msg.Type == tea.KeyEsc:
			if (m.state == stateLoading || m.state == stateStreaming) && m.cancelOp != nil {
				m.cancelOp()
				m.cancelOp = nil
				m.state = stateIdle
				m.addMsg("system", "Cancelled.")
			}

		case msg.Type == tea.KeyCtrlY:
			if m.lastAnswer != "" {
				clipboard.WriteAll(m.lastAnswer) //nolint:errcheck
				m.addMsg("system", "Copied to clipboard")
			}

		case msg.Type == tea.KeyUp && m.state == stateIdle:
			if len(m.history) > 0 {
				if m.histIdx == -1 {
					m.histDraft = m.textInput.Value()
					m.histIdx = len(m.history) - 1
				} else if m.histIdx > 0 {
					m.histIdx--
				}
				m.textInput.SetValue(m.history[m.histIdx])
				m.textInput.CursorEnd()
			}

		case msg.Type == tea.KeyDown && m.state == stateIdle:
			if m.histIdx >= 0 {
				if m.histIdx < len(m.history)-1 {
					m.histIdx++
					m.textInput.SetValue(m.history[m.histIdx])
				} else {
					m.histIdx = -1
					m.textInput.SetValue(m.histDraft)
				}
				m.textInput.CursorEnd()
			}

		case msg.Type == tea.KeyPgUp:
			manualViewportScroll = true
			m.viewport.HalfViewUp()
		case msg.Type == tea.KeyPgDown:
			manualViewportScroll = true
			m.viewport.HalfViewDown()

		case msg.Type == tea.KeyEnter:
			if m.state != stateIdle && m.state != stateConfirming {
				break
			}
			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				break
			}
			m.textInput.Reset()
			m.histIdx = -1
			if len(m.history) == 0 || m.history[len(m.history)-1] != input {
				m.history = append(m.history, input)
			}
			m.addMsg("user", input)

			// ── confirmation flow ──────────────────────────────────────────
			if m.state == stateConfirming {
				switch strings.ToLower(strings.TrimSpace(input)) {
				case "confirm", "yes", "y":
					m.textInput.Reset()
					return m, m.applyWikiUpdate(amendApplied)
				case "force":
					m.textInput.Reset()
					return m, m.applyWikiUpdate(amendForceApplied)
				default:
					m.state = stateIdle
					m.pendingUpdate = nil
					m.textInput.Reset()
					m.addMsg("system", "Update cancelled. Amendment not recorded.")
				}
				break
			}

			switch {
			case strings.HasPrefix(input, "!"):
				return m, m.handleShell(strings.TrimPrefix(input, "!"))
			case strings.HasPrefix(input, "/"):
				return m, tea.Batch(m.handleCommand(input), m.spinner.Tick)
			default:
				// Track user turn in conversation history
				m.convHistory = append(m.convHistory, rag.ConvMsg{Role: "user", Content: input})
				return m, tea.Batch(m.handleQuery(input), m.spinner.Tick)
			}
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
				manualViewportScroll = true
			}
		}

	// ── correction ready ──────────────────────────────────────────────────────
	case correctionReadyMsg:
		m.state = stateConfirming
		m.pendingUpdate = &wikiUpdateState{
			wikiPath:        msg.wikiPath,
			newContent:      msg.newContent,
			originalExcerpt: msg.originalExcerpt,
			proposedChange:  msg.proposedChange,
			analysis:        msg.analysis,
			isConsistent:    msg.isConsistent,
			summary:         msg.summary,
		}
		m.addMsg("brain", msg.summary)

	// ── async results ─────────────────────────────────────────────────────────
	case progressMsg:
		if msg.content != "" {
			m.addMsg("system", msg.content)
		}
		if msg.next != nil {
			cmds = append(cmds, msg.next)
		}

	case progressDoneMsg:
		m.state = stateIdle
		if msg.err != nil {
			m.addMsg("error", formatError(msg.err))
		} else {
			m.addMsg("brain", msg.summary)
		}

	case resultMsg:
		m.state = stateIdle
		if msg.err != nil {
			m.addMsg("error", formatError(msg.err))
		} else {
			m.addMsg("brain", msg.content)
			if len(msg.refs) > 0 {
				m.lastAnswer = msg.content
				m.lastRefs = msg.refs
			}
		}

	// ── streaming ─────────────────────────────────────────────────────────────
	case streamChunkMsg:
		m.streamingContent += msg.content
		if msg.next != nil {
			cmds = append(cmds, msg.next)
		}

	case streamDoneMsg:
		m.state = stateIdle
		answer := m.streamingContent
		m.streamingContent = ""
		if len(msg.refs) > 0 {
			refsStr := "\n\nReferences:\n"
			for _, r := range msg.refs {
				refsStr += "  → " + r + "\n"
			}
			answer += refsStr
			m.lastAnswer = answer
			m.lastRefs = msg.refs
		}
		m.addMsg("brain", answer+"\n"+systemMsgStyle.Render("  → /save <title>  to keep this"))
		// Store in conversation history for follow-up awareness
		m.convHistory = append(m.convHistory, rag.ConvMsg{Role: "assistant", Content: answer})

	case spinner.TickMsg:
		if m.state == stateLoading || m.state == stateStreaming {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport.SetContent(m.renderMessages())
	if manualViewportScroll {
		m.atBottom = m.viewport.AtBottom()
	}
	if m.atBottom {
		m.viewport.GotoBottom()
	}
	return m, tea.Batch(cmds...)
}

// ── command routing ────────────────────────────────────────────────────────────

func (m *Model) handleCommand(input string) tea.Cmd {
	parts := strings.Fields(input)

	switch parts[0] {
	case "/help":
		return func() tea.Msg { return resultMsg{content: helpText()} }
	case "/tips":
		return func() tea.Msg { return resultMsg{content: tipsMessage(m.cfg.Paths.Raw)} }
	case "/status":
		return m.cmdStatus()
	case "/pull":
		m.state = stateLoading
		m.loadingOp = randomAnalyzingPhrase()
		return m.cmdPullFrom(resolveSourceDir(parts[1:], m.cfg.ProjectPath, m.cfg.Paths.Raw))
	case "/save":
		title := strings.TrimSpace(strings.TrimPrefix(input, "/save"))
		if title == "" {
			return func() tea.Msg {
				return resultMsg{content: "Usage: /save <title>  saves last answer to wiki/synthesis/"}
			}
		}
		return m.cmdSave(title)
	case "/sync":
		m.state = stateLoading
		m.loadingOp = "Syncing knowledge base..."
		return m.cmdSync()
	case "/analyze":
		m.state = stateLoading
		m.loadingOp = randomAnalyzingPhrase()
		return m.cmdAnalyzeFrom(resolveSourceDir(parts[1:], m.cfg.ProjectPath, m.cfg.Paths.Raw))
	case "/amendments":
		return m.cmdAmendments()

	case "/gap":
		topic := strings.TrimSpace(strings.TrimPrefix(input, "/gap"))
		if topic == "" {
			return func() tea.Msg {
				return resultMsg{content: "Usage: /gap <topic>\n\nMarks a topic as missing in your wiki and creates a research stub.\nExample: /gap transformer attention mechanism"}
			}
		}
		return m.cmdGap(topic)

	case "/fixwiki":
		args := strings.TrimSpace(strings.TrimPrefix(input, "/fixwiki"))
		if args == "" {
			return func() tea.Msg {
				return resultMsg{content: "Usage: /fixwiki <wiki-path-or-name> <what to fix>\n\nExamples:\n  /fixwiki concepts/transformer the activation should be ReLU not sigmoid\n  /fixwiki transformer  (brain finds the page, describe the fix next)"}
			}
		}
		m.state = stateLoading
		m.loadingOp = "Finding wiki page..."
		return m.cmdFixWiki(args)

	case "/lint":
		m.state = stateLoading
		m.loadingOp = "Checking wiki health..."
		return m.cmdLint()

	case "/config":
		return m.cmdConfig(strings.TrimSpace(strings.TrimPrefix(input, "/config")))

	case "/logout":
		return func() tea.Msg {
			if err := config.Logout(); err != nil {
				return resultMsg{err: fmt.Errorf("logout: %w", err)}
			}
			return resultMsg{content: fmt.Sprintf(
				"Config directory removed: %s\n\nRestart brain to run setup again.", config.ConfigDir())}
		}

	default:
		return func() tea.Msg {
			return resultMsg{content: fmt.Sprintf("Unknown command: %s\nType /help for all commands.", parts[0])}
		}
	}
}

func (m *Model) handleQuery(question string) tea.Cmd {
	// Detect file path — copy to raw/ and analyze
	if filePath := extractFilePath(question); filePath != "" {
		return m.handleFileInChat(filePath, question)
	}

	// Detect correction intent — route to correction flow
	if isCorrectionIntent(question) {
		return m.handleCorrection(question)
	}

	// Guard: empty knowledge base
	_, chunks := m.store.Stats()
	if chunks == 0 {
		return func() tea.Msg {
			return resultMsg{content: fmt.Sprintf(
				"Your knowledge base is empty.\n\n"+
					"  Drop files into: %s\n"+
					"  They are auto-analyzed when added, or run /pull now.\n\n"+
					"  Type /tips for the quick-start guide.",
				m.cfg.Paths.Raw)}
		}
	}

	// Build history for follow-up context (last 6 turns = 3 exchanges)
	history := m.recentHistory(6)

	m.state = stateStreaming
	m.loadingOp = randomThinkingPhrase()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelOp = cancel
	return func() tea.Msg {
		defer cancel()
		ch := m.rag.QueryStream(ctx, question, history)
		return makeStreamListener(ch)()
	}
}

// handleCorrection starts the wiki correction flow.
func (m *Model) handleCorrection(userMsg string) tea.Cmd {
	m.state = stateLoading
	m.loadingOp = "Finding the relevant wiki page..."
	return func() tea.Msg {
		ctx := context.Background()

		return buildCorrectionMsg(ctx, m, "", userMsg)
	}
}

// handleFileInChat copies a file mentioned in chat to raw/ and triggers analysis.
func (m *Model) handleFileInChat(srcPath, question string) tea.Cmd {
	rawPath := m.cfg.Paths.Raw
	return func() tea.Msg {
		// Expand ~ if needed
		if strings.HasPrefix(srcPath, "~/") {
			home, _ := os.UserHomeDir()
			srcPath = filepath.Join(home, srcPath[2:])
		}
		info, err := os.Stat(srcPath)
		if err != nil {
			// Not a real file path — treat as normal question
			_, chunks := m.store.Stats()
			if chunks == 0 {
				return resultMsg{content: fmt.Sprintf("Knowledge base is empty. Drop files into %s first.", rawPath)}
			}
			ch := m.rag.QueryStream(context.Background(), question, nil)
			return makeStreamListener(ch)()
		}
		if info.IsDir() {
			return resultMsg{content: fmt.Sprintf(
				"That's a directory. Drop individual files into:\n  %s\nor copy the folder manually.", rawPath)}
		}
		dest := filepath.Join(rawPath, filepath.Base(srcPath))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return resultMsg{err: fmt.Errorf("reading %s: %w", srcPath, err)}
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return resultMsg{err: fmt.Errorf("copying to raw/: %w", err)}
		}
		return resultMsg{content: fmt.Sprintf(
			"Added to raw/: %s\n\nWhy: Files must live in raw/ so the brain can index them.\nRunning analysis now…",
			dest)}
	}
}

func (m *Model) handleShell(cmdStr string) tea.Cmd {
	projectPath := m.cfg.ProjectPath // always run from where brain was launched
	return func() tea.Msg {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			return resultMsg{content: "Empty shell command"}
		}
		// Run via bash -c so pipes, &&, cd, etc. all work
		c := exec.Command("bash", "-c", cmdStr)
		c.Dir = projectPath
		out, err := c.CombinedOutput()
		output := strings.TrimSpace(string(out))
		if output == "" {
			output = "(no output)"
		}
		header := fmt.Sprintf("$ %s  [in %s]", cmdStr, projectPath)
		if err != nil {
			return resultMsg{content: fmt.Sprintf("%s\n%s\n⚠ exit: %v", header, output, err)}
		}
		return resultMsg{content: fmt.Sprintf("%s\n%s", header, output)}
	}
}

// ── command implementations ────────────────────────────────────────────────────

func (m *Model) cmdStatus() tea.Cmd {
	return func() tea.Msg {
		pages := m.wiki.PageCount()
		_, chunks := m.store.Stats()
		rawCount := countFiles(m.cfg.Paths.Raw)
		watching := "off"
		if m.watcher != nil {
			watching = "on  (auto-analyzes new files)"
		}
		rawStatus := "✓"
		if _, err := os.Stat(m.cfg.Paths.Raw); err != nil {
			rawStatus = "✗ missing"
		}
		nestedWarning := ""
		if nestedRoot := detectNestedWikiRoot(m.cfg.Paths.Wiki); nestedRoot != "" {
			nestedWarning = "\n\nWarning    : nested wiki content detected at " + nestedRoot
		}
		return resultMsg{content: fmt.Sprintf(
			"Project dir : %s\n"+
				"raw/        : %s  [%s]  %d files\n"+
				"wiki/       : %s\n"+
				"Wiki pages  : %d\n"+
				"KB chunks   : %d\n"+
				"Auto-watch  : %s\n\n"+
				"Config dir  : %s\n"+
				"API key     : %s%s",
			m.cfg.ProjectPath,
			m.cfg.Paths.Raw, rawStatus, rawCount,
			m.cfg.Paths.Wiki,
			pages, chunks, watching,
			config.ConfigDir(),
			maskKey(os.Getenv("LLM_COMPATIBLE_API_KEY")),
			nestedWarning)}
	}
}

func (m *Model) cmdPullFrom(rawDir string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelOp = cancel
	ch := make(chan string, 40)
	go func() {
		defer close(ch)
		defer cancel()
		summary, err := m.analyzer.AnalyzeFrom(ctx, rawDir, func(s string) { ch <- s })
		_ = m.store.Save()
		if err != nil {
			ch <- fmt.Sprintf("⚠ %v", err)
		}
		ch <- "Syncing knowledge base..."
		pages, _ := m.wiki.ListPages()
		for _, page := range pages {
			content, err := m.wiki.Read(page)
			if err != nil {
				continue
			}
			h := wiki.HashBytes([]byte(content))
			_ = m.rag.IndexPage(ctx, page, content, h)
		}
		_ = m.store.Save()
		// Re-watch any new subdirs added
		rewatchSubdirs(m.watcher, rawDir)
		_, chunks := m.store.Stats()
		ch <- "___DONE___:" + fmt.Sprintf("%s\nKnowledge base: %d chunks indexed", summary, chunks)
	}()
	return makeProgressListener(ch)
}

func (m *Model) cmdAnalyzeFrom(rawDir string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelOp = cancel
	ch := make(chan string, 20)
	go func() {
		defer close(ch)
		defer cancel()
		summary, err := m.analyzer.AnalyzeFrom(ctx, rawDir, func(s string) { ch <- s })
		_ = m.store.Save()
		if err != nil {
			ch <- fmt.Sprintf("⚠ %v", err)
		}
		ch <- "___DONE___:" + summary
	}()
	return makeProgressListener(ch)
}

func (m *Model) cmdSync() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pages, err := m.wiki.ListPages()
		if err != nil {
			return resultMsg{err: err}
		}
		var synced, skipped int
		for _, page := range pages {
			content, err := m.wiki.Read(page)
			if err != nil {
				continue
			}
			h := wiki.HashBytes([]byte(content))
			if m.store.PageHash(page) == h {
				skipped++
				continue
			}
			if err := m.rag.IndexPage(ctx, page, content, h); err != nil {
				continue
			}
			synced++
		}
		_ = m.store.Save()
		_, chunks := m.store.Stats()
		return resultMsg{content: fmt.Sprintf(
			"Sync complete: %d re-indexed, %d unchanged\nKB: %d chunks total", synced, skipped, chunks)}
	}
}

func (m *Model) cmdLint() tea.Cmd {
	return func() tea.Msg {
		report, err := m.analyzer.LintWiki(context.Background())
		if err != nil {
			return resultMsg{err: err}
		}
		return resultMsg{content: report}
	}
}

// cmdFixWiki resolves a wiki page by path or fuzzy name, applies a user-described
// correction, and routes to the confirmation flow.
// Usage: /fixwiki <path-or-name> <correction description>
func (m *Model) cmdFixWiki(args string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Split: first token is the page identifier, rest is the correction
		parts := strings.SplitN(args, " ", 2)
		target := parts[0]
		correction := ""
		if len(parts) > 1 {
			correction = strings.TrimSpace(parts[1])
		}

		// Resolve the wiki page
		wikiPath, err := m.resolveWikiPage(target)
		if err != nil {
			return resultMsg{err: fmt.Errorf("page not found: %w\n\nTry /fixwiki with a path like: wiki/concepts/my-topic.md", err)}
		}

		if correction == "" {
			return resultMsg{content: fmt.Sprintf(
				"Found: %s\n\nDescribe what to fix, then run:\n  /fixwiki %s <your correction>",
				wikiPath, target)}
		}

		return buildCorrectionMsg(ctx, m, wikiPath, correction)
	}
}

// cmdAmendments lists all amendment records in knowledge-base/amendments/.
func (m *Model) cmdAmendments() tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Join(m.cfg.Paths.KnowledgeBase, "amendments")
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			return resultMsg{content: "No amendments recorded yet.\n\nAmendments are created when you correct wiki content via /fixwiki or by describing a correction in chat."}
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Amendments  (%s)\n", dir))
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			// Read first few lines to extract status and wiki_page
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			content := string(data)
			status := extractFrontmatterField(content, "status")
			wikiPage := extractFrontmatterField(content, "wiki_page")
			icon := "○"
			switch status {
			case "applied":
				icon = "✓"
			case "force-applied":
				icon = "⚡"
			}
			lines = append(lines, fmt.Sprintf("  %s  %s\n     %s", icon, e.Name(), wikiPage))
		}

		lines = append(lines, "\nTo view an amendment: !cat knowledge-base/amendments/<filename>")
		return resultMsg{content: strings.Join(lines, "\n")}
	}
}

// cmdGap creates a research stub page for a topic not yet in the wiki.
func (m *Model) cmdGap(topic string) tea.Cmd {
	return func() tea.Msg {
		slug := slugify(strings.ToLower(topic))
		wikiPath := "wiki/sources/gap-" + slug + ".md"
		today := time.Now().UTC().Format("2006-01-02")

		content := fmt.Sprintf(`---
type: gap
title: "%s"
tags: [gap, research-needed]
created: %s
updated: %s
---

# Research Gap: %s

> This page was flagged as missing from the wiki. Add sources to raw/ and run /pull to fill it in.

## What We're Looking For
[Describe what information is needed about this topic]

## Suggested Sources
[Note any sources, papers, or links that might cover this]

## Why It Matters
[Why does this belong in the wiki?]
`, topic, today, today, topic)

		if err := m.wiki.Write(wikiPath, content); err != nil {
			return resultMsg{err: fmt.Errorf("creating gap page: %w", err)}
		}
		m.wiki.AppendLog(fmt.Sprintf("## [%s] gap | %s\nFlagged as missing: %s", today, topic, wikiPath)) //nolint:errcheck

		// Index the stub so it shows up in searches
		h := wiki.HashBytes([]byte(content))
		_ = m.rag.IndexPage(context.Background(), wikiPath, content, h)
		_ = m.store.Save()

		return resultMsg{content: fmt.Sprintf(
			"Research gap flagged: %s\n\nStub created at: %s\n\nWhen you have a source, drop it into raw/ and run /pull.",
			topic, wikiPath)}
	}
}

// buildCorrectionMsg is the shared core of handleCorrection and cmdFixWiki.
// If wikiPath is empty, it finds the best match via RAG.
func buildCorrectionMsg(ctx context.Context, m *Model, wikiPath, correction string) tea.Msg {
	var err error
	if wikiPath == "" {
		wikiPath, _, err = m.rag.TopResult(ctx, correction)
		if err != nil {
			return resultMsg{err: fmt.Errorf("no matching wiki page found: %w", err)}
		}
	}

	currentContent, err := m.wiki.Read(wikiPath)
	if err != nil {
		return resultMsg{err: fmt.Errorf("reading %s: %w", wikiPath, err)}
	}

	// Excerpt for the amendment record (first 800 chars)
	excerpt := currentContent
	if len(excerpt) > 800 {
		excerpt = excerpt[:800] + "\n...[truncated]"
	}

	// Analyze: consistent or contradictory?
	analysis, consistent, err := m.analyzer.AnalyzeAmendment(ctx, wikiPath, currentContent, correction)
	if err != nil {
		analysis = "(analysis unavailable)"
		consistent = true
	}

	// Generate corrected page content
	newContent, err := m.analyzer.CorrectPage(ctx, wikiPath, currentContent, correction)
	if err != nil {
		return resultMsg{err: fmt.Errorf("generating correction: %w", err)}
	}

	// Build summary shown to user
	consistencyLine := "✓  Consistent with existing content."
	if !consistent {
		consistencyLine = "⚠  Contradicts current wiki content. Type force to apply anyway."
	}
	summary := fmt.Sprintf(
		"Page      : %s\nAnalysis  : %s\n\n%s\n\nType confirm  to apply  ·  force  to override  ·  anything else to cancel",
		wikiPath, consistencyLine, analysis)

	return correctionReadyMsg{
		wikiPath:        wikiPath,
		newContent:      newContent,
		originalExcerpt: excerpt,
		proposedChange:  correction,
		analysis:        analysis,
		isConsistent:    consistent,
		summary:         summary,
	}
}

// resolveWikiPage finds a wiki page by exact path, partial path, or fuzzy name match.
func (m *Model) resolveWikiPage(target string) (string, error) {
	pages, err := m.wiki.ListPages()
	if err != nil {
		return "", err
	}

	// 1. Exact match
	for _, p := range pages {
		if p == target || p == "wiki/"+target {
			return p, nil
		}
	}

	// 2. Suffix match (e.g. "transformer" → "wiki/concepts/transformer-architecture.md")
	lower := strings.ToLower(target)
	lower = strings.TrimSuffix(lower, ".md")
	var candidates []string
	for _, p := range pages {
		base := strings.TrimSuffix(filepath.Base(p), ".md")
		if strings.Contains(strings.ToLower(base), lower) ||
			strings.Contains(strings.ToLower(p), lower) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("ambiguous — multiple matches:\n  %s\n\nBe more specific", strings.Join(candidates, "\n  "))
	}

	return "", fmt.Errorf("no wiki page matching %q", target)
}

func (m *Model) cmdConfig(arg string) tea.Cmd {
	return func() tea.Msg {
		switch arg {
		case "key", "apikey", "api-key":
			current := config.GetAPIKey()
			masked := ""
			if len(current) > 8 {
				masked = current[:4] + strings.Repeat("•", len(current)-8) + current[len(current)-4:]
			} else if current != "" {
				masked = strings.Repeat("•", len(current))
			}
			if masked == "" {
				masked = "(not set)"
			}
			return resultMsg{content: fmt.Sprintf(
				"Current API key: %s\n\nTo update: edit %s\nThen restart brain.",
				masked, config.EnvPath())}
		case "":
			_, chunks := m.store.Stats()
			rawExists := "✓ exists"
			if _, err := os.Stat(m.cfg.Paths.Raw); err != nil {
				rawExists = "✗ missing — run /pull to create"
			}
			return resultMsg{content: fmt.Sprintf(
				"Config dir  : %s\n"+
					"  config.yaml : %s\n"+
					"  .env        : %s\n\n"+
					"Project dir : %s\n"+
					"  raw/        : %s  [%s]\n"+
					"  wiki/       : %s\n\n"+
					"Model       : %s\n"+
					"Embeddings  : %s\n"+
					"KB chunks   : %d\n\n"+
					"Subcommands:\n"+
					"  /config key   — show masked API key\n"+
					"  /config reset — remove config, re-run setup on restart\n"+
					"  /logout       — same as reset (removes entire config dir)",
				config.ConfigDir(),
				config.ConfigFilePath(),
				config.EnvPath(),
				m.cfg.ProjectPath,
				m.cfg.Paths.Raw, rawExists,
				m.cfg.Paths.Wiki,
				m.cfg.LLM.Model,
				m.cfg.Embeddings.Model,
				chunks)}
		case "reset":
			if err := config.Logout(); err != nil {
				return resultMsg{err: fmt.Errorf("reset: %w", err)}
			}
			return resultMsg{content: fmt.Sprintf("Config removed: %s\nRestart brain to run setup again.", config.ConfigDir())}
		default:
			return resultMsg{content: fmt.Sprintf("Unknown config option: %s\nTry: /config  /config key  /config reset", arg)}
		}
	}
}

func (m *Model) cmdSave(title string) tea.Cmd {
	answer, refs := m.lastAnswer, m.lastRefs
	if answer == "" {
		return func() tea.Msg {
			return resultMsg{content: "No answer to save — ask a question first."}
		}
	}
	return func() tea.Msg {
		slug := slugify(title)
		wikiPath := "wiki/synthesis/" + slug + ".md"
		today := time.Now().UTC().Format("2006-01-02")
		content := fmt.Sprintf("---\ntype: synthesis\ntitle: %s\ntags: []\nsources: [%s]\ncreated: %s\nupdated: %s\n---\n\n# %s\n\n%s\n",
			title, strings.Join(refs, ", "), today, today, title, answer)
		if err := m.wiki.Write(wikiPath, content); err != nil {
			return resultMsg{err: fmt.Errorf("writing synthesis page: %w", err)}
		}
		m.wiki.AppendLog(fmt.Sprintf("## [%s] query | %s\nFiled: %s.", today, title, wikiPath)) //nolint:errcheck
		h := wiki.HashBytes([]byte(content))
		_ = m.rag.IndexPage(context.Background(), wikiPath, content, h)
		_ = m.store.Save()
		return resultMsg{content: "Saved → " + wikiPath}
	}
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	if !m.ready {
		return "Loading...\n"
	}
	header := m.renderHeader()
	content := m.viewport.View()
	divider := dividerStyle.Render(strings.Repeat("─", m.width))
	var footer string
	switch m.state {
	case stateLoading:
		footer = spinnerStyle.Render(m.spinner.View()) + " " + systemMsgStyle.Render(m.loadingOp)
	case stateStreaming:
		footer = spinnerStyle.Render(m.spinner.View()) + " " + systemMsgStyle.Render(m.loadingOp)
	case stateConfirming:
		warning := ""
		if m.pendingUpdate != nil && !m.pendingUpdate.isConsistent {
			warning = errorMsgStyle.Render("⚠ contradicts source  ") + "  "
		}
		footer = warning + userLabelStyle.Render("confirm") + helpStyle.Render(" · ") +
			errorMsgStyle.Render("force") + helpStyle.Render(" (override) · cancel  > ") + m.textInput.View()
	default:
		footer = inputPromptStyle.Render("> ") + m.textInput.View() +
			"   " + helpStyle.Render("↑↓ history · wheel scroll · ctrl+y copy")
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content, divider, footer)
}

func (m *Model) renderHeader() string {
	// Left: brain icon + name
	left := headerStyle.Render("  🧠  TheSecondBrain  ")

	// Right: neuro-node stats + watch indicator
	_, chunks := m.store.Stats()
	watchDot := ""
	if m.watcher != nil {
		watchDot = "  ◉" // active watcher indicator
	}
	right := headerStatStyle.Render(fmt.Sprintf(
		"  ◈ wiki:%d   ◈ kb:%d%s  ",
		m.wiki.PageCount(), chunks, watchDot))

	gap := strings.Repeat(" ", maxInt(0, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	bg := lipgloss.NewStyle().Background(colorGreen).Render(gap)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, bg, right)
}

func (m *Model) renderMessages() string {
	var sb strings.Builder
	for _, msg := range m.msgs {
		sb.WriteString(m.renderMsg(msg))
		sb.WriteString("\n")
	}
	if m.state == stateStreaming && len(m.streamingContent) > 0 {
		ts := systemMsgStyle.Render(time.Now().Format("15:04"))
		sb.WriteString(fmt.Sprintf("%s %s\n%s\n", ts, brainLabelStyle.Render("Brain"), m.streamingContent))
	}
	return sb.String()
}

func (m *Model) renderMsg(msg chatMsg) string {
	ts := systemMsgStyle.Render(msg.at.Format("15:04"))
	switch msg.role {
	case "user":
		return fmt.Sprintf("%s %s\n%s", ts, userLabelStyle.Render("You"), msg.content)
	case "brain":
		lines := strings.Split(msg.content, "\n")
		for i, line := range lines {
			t := strings.TrimLeft(line, " ")
			if strings.HasPrefix(t, "→ ") {
				lines[i] = refStyle.Render("  ") + refPathStyle.Render(t)
			}
		}
		return fmt.Sprintf("%s %s\n%s", ts, brainLabelStyle.Render("Brain"), strings.Join(lines, "\n"))
	case "error":
		return errorMsgStyle.Render("⚠  " + msg.content)
	default:
		return systemMsgStyle.Render("   " + msg.content)
	}
}

// ── file watcher ──────────────────────────────────────────────────────────────

func startWatcher(rawPath string) *fsnotify.Watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	// Watch raw/ and all existing subdirs
	if err := w.Add(rawPath); err != nil {
		w.Close()
		return nil
	}
	filepath.WalkDir(rawPath, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			w.Add(path) //nolint:errcheck
		}
		return nil
	})
	return w
}

func watchRaw(w *fsnotify.Watcher) tea.Cmd {
	var listen tea.Cmd
	listen = func() tea.Msg {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				base := filepath.Base(event.Name)
				if !strings.HasPrefix(base, ".") && !strings.HasSuffix(base, "~") {
					return rawFileChangedMsg{}
				}
			}
			return listen()
		case _, ok := <-w.Errors:
			if !ok {
				return nil
			}
			return listen()
		}
	}
	return listen
}

func rewatchSubdirs(w *fsnotify.Watcher, rawPath string) {
	if w == nil {
		return
	}
	filepath.WalkDir(rawPath, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			w.Add(path) //nolint:errcheck
		}
		return nil
	})
}

func deferred(d time.Duration, msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return msg
	}
}

// ── streaming + progress listeners ────────────────────────────────────────────

func makeStreamListener(ch <-chan rag.StreamMsg) tea.Cmd {
	var listen tea.Cmd
	listen = func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		if msg.Err != nil {
			return resultMsg{err: msg.Err}
		}
		if msg.Done {
			return streamDoneMsg{refs: msg.Refs}
		}
		return streamChunkMsg{content: msg.Chunk, next: listen}
	}
	return listen
}

func makeProgressListener(ch <-chan string) tea.Cmd {
	var listen tea.Cmd
	listen = func() tea.Msg {
		content, ok := <-ch
		if !ok {
			return progressDoneMsg{}
		}
		if strings.HasPrefix(content, "___DONE___:") {
			return progressDoneMsg{summary: strings.TrimPrefix(content, "___DONE___:")}
		}
		return progressMsg{content: content, next: listen}
	}
	return listen
}

// ── vault structure ────────────────────────────────────────────────────────────

func ensureVaultStructure(cfg *config.Config) {
	dirs := []string{
		cfg.Paths.Raw,
		filepath.Join(cfg.Paths.Wiki, "sources"),
		filepath.Join(cfg.Paths.Wiki, "entities"),
		filepath.Join(cfg.Paths.Wiki, "concepts"),
		filepath.Join(cfg.Paths.Wiki, "synthesis"),
		filepath.Join(cfg.Paths.KnowledgeBase, "embeddings"),
		filepath.Join(cfg.Paths.KnowledgeBase, "metadata"),
		filepath.Join(cfg.Paths.KnowledgeBase, "output"),
		filepath.Join(cfg.Paths.KnowledgeBase, "amendments"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755) //nolint:errcheck
	}
	// Bootstrap wiki files if missing
	writeIfAbsent(filepath.Join(cfg.Paths.Wiki, "index.md"),
		"# Wiki Index\nLast updated: — | Pages: 0\n\n## Sources\n## Entities\n## Concepts\n## Synthesis\n")
	writeIfAbsent(filepath.Join(cfg.Paths.Wiki, "log.md"),
		"# Wiki Log\n\nAppend-only record of all operations.\n\n---\n\n")
	writeIfAbsent(filepath.Join(cfg.Paths.KnowledgeBase, "metadata", "sources.json"), "{}\n")
}

func writeIfAbsent(path, content string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.WriteFile(path, []byte(content), 0644) //nolint:errcheck
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m *Model) addMsg(role, content string) {
	m.msgs = append(m.msgs, chatMsg{role: role, content: content, at: time.Now()})
	// Do NOT force atBottom here — user may have scrolled up to read history.
	// GotoBottom only fires when atBottom is already true (user is at the bottom).
}

func countFiles(root string) int {
	count := 0
	filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func detectNestedWikiRoot(wikiRoot string) string {
	nested := filepath.Join(wikiRoot, "wiki")
	info, err := os.Stat(nested)
	if err != nil || !info.IsDir() {
		return ""
	}

	foundMarkdown := false
	filepath.WalkDir(nested, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			foundMarkdown = true
			return fs.SkipAll
		}
		return nil
	})
	if foundMarkdown {
		return nested
	}
	return ""
}

// resolvePath expands ~ to the home directory and makes the path absolute.
func resolvePath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func resolveSourceDir(args []string, projectPath, defaultPath string) string {
	if len(args) == 0 {
		return defaultPath
	}
	if isCurrentDirArg(args[0]) {
		return projectPath
	}
	return resolvePath(args[0])
}

func isCurrentDirArg(arg string) bool {
	switch arg {
	case "-cd", "--cd", "--current-dir":
		return true
	default:
		return false
	}
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// applyWikiUpdate writes the correction, records an amendment, and re-indexes.
func (m *Model) applyWikiUpdate(status string) tea.Cmd {
	upd := m.pendingUpdate
	m.pendingUpdate = nil
	m.state = stateLoading
	m.loadingOp = "Applying correction..."
	kbPath := m.cfg.Paths.KnowledgeBase
	return func() tea.Msg {
		if err := m.wiki.Write(upd.wikiPath, upd.newContent); err != nil {
			return resultMsg{err: fmt.Errorf("writing update: %w", err)}
		}
		h := wiki.HashBytes([]byte(upd.newContent))
		_ = m.rag.IndexPage(context.Background(), upd.wikiPath, upd.newContent, h)
		_ = m.store.Save()

		amendPath := writeAmendmentFile(kbPath, upd, status)
		today := time.Now().UTC().Format("2006-01-02")
		m.wiki.AppendLog(fmt.Sprintf("## [%s] correction | %s\nStatus: %s. Amendment: %s", //nolint:errcheck
			today, upd.wikiPath, status, amendPath))

		return resultMsg{content: fmt.Sprintf(
			"✓ %s  %s\nAmendment recorded: %s",
			status, upd.wikiPath, amendPath)}
	}
}

// writeAmendmentFile writes an audit record of the correction to knowledge-base/amendments/.
func writeAmendmentFile(kbPath string, upd *wikiUpdateState, status string) string {
	dir := filepath.Join(kbPath, "amendments")
	os.MkdirAll(dir, 0755) //nolint:errcheck

	now := time.Now().UTC()
	slug := slugify(upd.proposedChange)
	if len(slug) > 40 {
		slug = slug[:40]
	}
	if slug == "" {
		slug = slugify(filepath.Base(upd.wikiPath))
	}
	filename := fmt.Sprintf("%s-%s.md", now.Format("20060102-150405"), slug)
	fullPath := filepath.Join(dir, filename)

	consistency := "✓ CONSISTENT"
	if !upd.isConsistent {
		consistency = "⚠ CONTRADICTORY"
	}
	decisionLine := fmt.Sprintf("[x] %s on %s", strings.Title(status), now.Format("2006-01-02")) //nolint:staticcheck

	content := fmt.Sprintf(`---
status: %s
wiki_page: %s
created: %s
---

# Amendment: %s

## Original (wiki content at time of change)
%s

## Proposed Change
%s

## System Analysis
%s

%s

## Decision
%s
`, status, upd.wikiPath, now.Format("2006-01-02"),
		strings.TrimSuffix(filepath.Base(upd.wikiPath), ".md"),
		upd.originalExcerpt,
		upd.proposedChange,
		consistency,
		upd.analysis,
		decisionLine)

	os.WriteFile(fullPath, []byte(content), 0644) //nolint:errcheck

	rel, _ := filepath.Rel(filepath.Dir(kbPath), fullPath)
	return rel
}

// recentHistory returns the last n conversation turns for follow-up context.
func (m *Model) recentHistory(n int) []rag.ConvMsg {
	if len(m.convHistory) <= n {
		return m.convHistory
	}
	return m.convHistory[len(m.convHistory)-n:]
}

// isCorrectionIntent detects if the user is trying to correct wiki content.
func isCorrectionIntent(text string) bool {
	lower := strings.ToLower(text)
	for _, trigger := range []string{
		"that's wrong", "that is wrong", "thats wrong",
		"not correct", "not right", "is incorrect",
		"update wiki", "update the wiki", "correct the wiki", "fix the wiki",
		"fix that entry", "wrong information", "wrong in the wiki",
		"should say", "should read", "should be corrected",
	} {
		if strings.Contains(lower, trigger) {
			return true
		}
	}
	return false
}

// extractFilePath finds the first file-like token in a message (starts with /, ~/, ./).
func extractFilePath(text string) string {
	for _, word := range strings.Fields(text) {
		// Strip trailing punctuation
		word = strings.TrimRight(word, ".,;:!?\"')")
		if strings.HasPrefix(word, "/") || strings.HasPrefix(word, "~/") || strings.HasPrefix(word, "./") {
			return word
		}
	}
	return ""
}

// extractFrontmatterField reads a YAML frontmatter field value from markdown content.
func extractFrontmatterField(content, field string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
		}
	}
	return ""
}

func maskKey(key string) string {
	if key == "" {
		return "(not set — check ~/.config/secondbrain/.env)"
	}
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:4] + strings.Repeat("•", len(key)-8) + key[len(key)-4:]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatError(err error) string {
	if err == nil {
		return ""
	}
	s := strings.ToLower(err.Error())
	var hint string
	switch {
	case strings.Contains(s, "401") || strings.Contains(s, "unauthorized"):
		hint = "Check LLM_COMPATIBLE_API_KEY in ~/.config/secondbrain/.env"
	case strings.Contains(s, "connection refused") || strings.Contains(s, "no such host"):
		hint = "Cannot reach the API — check your internet connection."
	case strings.Contains(s, "context deadline") || strings.Contains(s, "timeout"):
		hint = "Request timed out — try again."
	case strings.Contains(s, "no such file") || strings.Contains(s, "not found"):
		hint = "Path not found — run /pull to initialize the wiki."
	case strings.Contains(s, "parsing") || strings.Contains(s, "json"):
		hint = "Unexpected LLM output — try again."
	}
	if hint != "" {
		return err.Error() + "\n\n  Suggestion: " + hint
	}
	return err.Error()
}

func tipsMessage(rawPath string) string {
	return fmt.Sprintf(`Welcome to TheSecondBrain  ◆

  Your wiki is empty. Here's how to get started:

  1. Drop any file into:
       %s
     (docs, notes, images, code, and other supported text files)
     Files are auto-analyzed when added.

     Note: PDFs are not supported yet in this build.

  2. Run /pull to process files immediately.

  3. Ask questions in plain English.
     The brain searches your wiki and streams answers live.

  4. /save <title>  to keep any answer as a wiki page.

  5. /lint  to health-check your wiki.

  6. !<command>  to run shell commands from your vault.
     Example: !ls raw/

  Shortcuts:
    ↑ ↓       navigate command history
    Mouse wheel  scroll chat
    Ctrl+Y    copy last answer to clipboard
    Ctrl+C    quit

  Type /tips to see this again anytime.`, rawPath)
}

func helpText() string {
	return `Commands:

  /pull             Scan raw/ → extract knowledge → wiki + knowledge-base
  /pull --current-dir  Scan the project root instead of raw/
  /save <title>     Save last answer as wiki/synthesis/<slug>.md
  /sync             Re-embed changed wiki pages (after manual edits)
  /analyze          Force re-analyze raw/ (reprocess all files)
  /analyze --current-dir  Re-analyze the project root instead of raw/
  /gap <topic>      Flag a missing topic — creates a research stub in wiki/sources/
  /fixwiki <name> <fix>  Correct a wiki page by name or path
                         Example: /fixwiki transformer activation should be ReLU not sigmoid
  /lint             Wiki health check (links, orphans, stubs)
  /status           Show vault stats, paths, and API key status
  /config           Show all settings
  /config key       Show masked API key
  /config reset     Clear config and re-run setup on next restart
  /logout           Remove stored API key
  /tips             Show quick-start guide
  /help             Show this help

  !<command>        Run a shell command from your vault root
                    Examples: !ls raw/   !cat wiki/index.md   !open .

  (anything else)   Ask a question — answer streams from your wiki
                    Mention a file path and it will be added to raw/

Keyboard:
  ↑ / ↓        Navigate command history
  Mouse wheel  Scroll chat
  Ctrl+Y       Copy last answer to clipboard
  Ctrl+C       Quit`
}
