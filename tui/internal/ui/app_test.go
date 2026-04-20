package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/rag"
	"github.com/ORG028658/TheSecondBrain/tui/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEnsureVaultStructureCreatesExpectedFiles(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}

	ensureVaultStructure(cfg)

	for _, path := range []string{
		cfg.Paths.Raw,
		filepath.Join(cfg.Paths.Wiki, "sources"),
		filepath.Join(cfg.Paths.Wiki, "entities"),
		filepath.Join(cfg.Paths.Wiki, "concepts"),
		filepath.Join(cfg.Paths.Wiki, "synthesis"),
		filepath.Join(cfg.Paths.KnowledgeBase, "embeddings"),
		filepath.Join(cfg.Paths.KnowledgeBase, "metadata"),
		filepath.Join(cfg.Paths.KnowledgeBase, "output"),
		filepath.Join(cfg.Paths.KnowledgeBase, "amendments"),
		filepath.Join(cfg.Paths.Wiki, "index.md"),
		filepath.Join(cfg.Paths.Wiki, "log.md"),
		filepath.Join(cfg.Paths.KnowledgeBase, "metadata", "sources.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestDetectNestedWikiRootReturnsNestedPathWhenMarkdownExists(t *testing.T) {
	wikiRoot := filepath.Join(t.TempDir(), "wiki")
	nested := filepath.Join(wikiRoot, "wiki", "sources")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "orphan.md"), []byte("# orphan"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	got := detectNestedWikiRoot(wikiRoot)
	want := filepath.Join(wikiRoot, "wiki")
	if got != want {
		t.Fatalf("detectNestedWikiRoot mismatch: got %q want %q", got, want)
	}
}

func TestDetectNestedWikiRootIgnoresEmptyOrNonMarkdownNestedDir(t *testing.T) {
	wikiRoot := filepath.Join(t.TempDir(), "wiki")
	nested := filepath.Join(wikiRoot, "wiki")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "note.txt"), []byte("not markdown"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if got := detectNestedWikiRoot(wikiRoot); got != "" {
		t.Fatalf("expected no nested warning, got %q", got)
	}
}

func TestResolveSourceDirSupportsCurrentDirFlags(t *testing.T) {
	project := "/tmp/project"
	raw := "/tmp/project/raw"

	for _, arg := range []string{"-cd", "--cd", "--current-dir"} {
		if got := resolveSourceDir([]string{arg}, project, raw); got != project {
			t.Fatalf("resolveSourceDir(%q) = %q, want %q", arg, got, project)
		}
	}
}

func TestResolveSourceDirFallsBackToDefaultOrExplicitPath(t *testing.T) {
	project := "/tmp/project"
	raw := "/tmp/project/raw"

	if got := resolveSourceDir(nil, project, raw); got != raw {
		t.Fatalf("resolveSourceDir(nil) = %q, want %q", got, raw)
	}

	if got := resolveSourceDir([]string{"docs"}, project, raw); !filepath.IsAbs(got) {
		t.Fatalf("resolveSourceDir explicit path should be absolute, got %q", got)
	}
}

func TestAvailableViewportHeightAccountsForWrappedFooter(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}

	m := NewModel(cfg)
	if m.watcher != nil {
		defer m.watcher.Close()
	}
	m.store = store.NewFresh(filepath.Join(cfg.Paths.KnowledgeBase, "embeddings", "store.json"))
	m.width = 40
	m.height = 12
	m.textInput.SetValue(strings.Repeat("x", 80))

	if footerHeight := m.measureWrappedHeight(m.renderFooter()); footerHeight < 2 {
		t.Fatalf("expected wrapped footer on narrow terminal, got height %d", footerHeight)
	}

	got := m.availableViewportHeight()
	if got >= m.height-3 {
		t.Fatalf("expected wrapped footer to reduce viewport below naive height, got %d", got)
	}
}

func TestRenderMessagesAddsBottomSafeGap(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}

	m := NewModel(cfg)
	if m.watcher != nil {
		defer m.watcher.Close()
	}
	m.msgs = []chatMsg{{role: "brain", content: "hello", at: time.Now()}}

	rendered := m.renderMessages()
	if !strings.HasSuffix(rendered, strings.Repeat("\n", messageBottomSafeGap)) {
		t.Fatalf("expected renderMessages to end with %d newline spacer(s)", messageBottomSafeGap)
	}
}

func TestIsCorrectionIntentRecognisesTriggerPhrases(t *testing.T) {
	for _, phrase := range []string{
		"that's wrong",
		"that is wrong",
		"thats wrong",
		"not correct",
		"not right",
		"is incorrect",
		"update wiki",
		"update the wiki",
		"correct the wiki",
		"fix the wiki",
		"fix that entry",
		"wrong information",
		"wrong in the wiki",
		"should say",
		"should read",
		"should be corrected",
	} {
		if !isCorrectionIntent(phrase) {
			t.Errorf("isCorrectionIntent(%q) = false, want true", phrase)
		}
	}
	for _, phrase := range []string{
		"what is a transformer?",
		"explain attention",
		"summarize the wiki",
	} {
		if isCorrectionIntent(phrase) {
			t.Errorf("isCorrectionIntent(%q) = true, want false", phrase)
		}
	}
}

func TestExtractFilePath(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/absolute/path/file.go", "/absolute/path/file.go"},
		{"~/home/file.md", "~/home/file.md"},
		{"./relative/file.txt", "./relative/file.txt"},
		{"look at /some/file.go please", "/some/file.go"},
		{"no path here", ""},
		{"just text", ""},
		{"/path/with/trailing,", "/path/with/trailing"},
	}
	for _, c := range cases {
		if got := extractFilePath(c.input); got != c.want {
			t.Errorf("extractFilePath(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"transformer architecture", "transformer-architecture"},
		{"foo_bar-baz", "foo-bar-baz"},
		{"Special! @chars#", "special-chars"},
		{"  leading trailing  ", "leading-trailing"},
	}
	for _, c := range cases {
		if got := slugify(c.input); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatErrorInjectsHints(t *testing.T) {
	cases := []struct {
		err      string
		contains string
	}{
		{"401 unauthorized", "LLM_COMPATIBLE_API_KEY"},
		{"connection refused to host", "internet connection"},
		{"context deadline exceeded", "timed out"},
		{"no such file or directory", "/pull"},
		{"error parsing json response", "again"},
	}
	for _, c := range cases {
		result := formatError(fmt.Errorf("%s", c.err))
		if !strings.Contains(result, c.contains) {
			t.Errorf("formatError(%q) missing hint %q, got: %q", c.err, c.contains, result)
		}
	}
}

func TestAppendConvHistoryCapsAt40(t *testing.T) {
	var h []rag.ConvMsg
	for i := 0; i < 50; i++ {
		h = appendConvHistory(h, rag.ConvMsg{Role: "user", Content: "msg"})
	}
	if len(h) != maxConvHistory {
		t.Fatalf("convHistory len = %d, want %d", len(h), maxConvHistory)
	}
}

func TestEscInConfirmingCancels(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}
	m := NewModel(cfg)
	if m.watcher != nil {
		defer m.watcher.Close()
	}
	m.state = stateConfirming
	m.pendingUpdate = &wikiUpdateState{wikiPath: "wiki/concepts/test.md"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(*Model)

	if result.state != stateIdle {
		t.Fatalf("state = %v after Esc in stateConfirming, want stateIdle", result.state)
	}
	if result.pendingUpdate != nil {
		t.Fatal("pendingUpdate should be nil after Esc cancel")
	}
}

func TestResultMsgUpdatesLastCopyTextWithoutRefs(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}

	model := NewModel(cfg)
	if model.watcher != nil {
		defer model.watcher.Close()
	}

	updated, _ := model.Update(resultMsg{content: "local help output"})
	m := updated.(*Model)

	if m.lastCopyText != "local help output" {
		t.Fatalf("lastCopyText = %q, want %q", m.lastCopyText, "local help output")
	}
	if m.lastAnswer != "" {
		t.Fatalf("lastAnswer should stay empty for non-referenced result, got %q", m.lastAnswer)
	}
}
