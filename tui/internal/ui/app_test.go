package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
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
