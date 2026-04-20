package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/wiki"
)

// ── shouldAnalyzeFile ─────────────────────────────────────────────────────────

func TestShouldAnalyzeFile_TextFiles(t *testing.T) {
	accept := []string{
		"main.go", "app.ts", "index.tsx", "component.vue",
		"styles.css", "config.yaml", "schema.sql",
		"main.py", "App.kt", "Main.java", "lib.rs",
		"README.md", "notes.txt", "Makefile", "Dockerfile",
	}
	for _, name := range accept {
		t.Run(name, func(t *testing.T) {
			if !shouldAnalyzeFile(name) {
				t.Errorf("shouldAnalyzeFile(%q) = false, want true", name)
			}
		})
	}
}

func TestShouldAnalyzeFile_BinaryAndGenerated(t *testing.T) {
	reject := []string{
		"app.exe", "lib.so", "lib.dylib", "image.bin",
		"archive.zip", "icon.ico",
		// .png/.jpg are accepted — they go through the vision analysis path
		// package-lock.json is .json and accepted — exclude via .brainignore if needed
		"app.wasm",
		"font.woff2", "font.ttf",
		"data.parquet", "model.onnx",
	}
	for _, name := range reject {
		t.Run(name, func(t *testing.T) {
			if shouldAnalyzeFile(name) {
				t.Errorf("shouldAnalyzeFile(%q) = true, want false (should be skipped)", name)
			}
		})
	}
}

func TestShouldAnalyzeFile_Images(t *testing.T) {
	// Images go through vision path — should return true
	images := []string{"photo.jpg", "diagram.png", "screenshot.webp", "logo.gif"}
	for _, name := range images {
		t.Run(name, func(t *testing.T) {
			if !shouldAnalyzeFile(name) {
				t.Errorf("shouldAnalyzeFile(%q) = false, want true (image → vision path)", name)
			}
		})
	}
}

func TestShouldAnalyzeFile_PDFSkipped(t *testing.T) {
	if shouldAnalyzeFile("document.pdf") {
		t.Error("PDF should be skipped (no text extraction support)")
	}
}

// ── matchesIgnorePattern ──────────────────────────────────────────────────────

func TestMatchesIgnorePattern_GlobBasename(t *testing.T) {
	patterns := []string{"*.min.js", "*.generated.ts"}

	cases := []struct {
		path  string
		match bool
	}{
		{"src/app.min.js", true},
		{"dist/bundle.min.js", true},
		{"api.generated.ts", true},
		{"src/app.js", false},
		{"src/app.ts", false},
	}
	for _, tc := range cases {
		got := matchesIgnorePattern(tc.path, patterns)
		if got != tc.match {
			t.Errorf("matchesIgnorePattern(%q) = %v, want %v", tc.path, got, tc.match)
		}
	}
}

func TestMatchesIgnorePattern_DirectoryPattern(t *testing.T) {
	patterns := []string{"fixtures/", "testdata/"}

	cases := []struct {
		path  string
		match bool
	}{
		{"fixtures/data.json", true},
		{"src/fixtures/mock.json", true},
		{"testdata/input.txt", true},
		{"src/main.go", false},
		{"test/helpers.go", false},
	}
	for _, tc := range cases {
		got := matchesIgnorePattern(tc.path, patterns)
		if got != tc.match {
			t.Errorf("matchesIgnorePattern(%q) = %v, want %v", tc.path, got, tc.match)
		}
	}
}

func TestMatchesIgnorePattern_EmptyPatterns(t *testing.T) {
	if matchesIgnorePattern("anything/file.go", nil) {
		t.Error("nil patterns should never match")
	}
	if matchesIgnorePattern("anything/file.go", []string{}) {
		t.Error("empty patterns should never match")
	}
}

// ── loadBrainIgnore ───────────────────────────────────────────────────────────

func TestLoadBrainIgnore_MissingFile(t *testing.T) {
	patterns := loadBrainIgnore(t.TempDir())
	if patterns != nil {
		t.Error("missing .brainignore should return nil, not empty slice")
	}
}

func TestLoadBrainIgnore_ParsesCorrectly(t *testing.T) {
	dir := t.TempDir()
	content := "# comment\n\n*.min.js\nfixtures/\n\n# another comment\n*.generated.ts\n"
	os.WriteFile(filepath.Join(dir, ".brainignore"), []byte(content), 0644)

	patterns := loadBrainIgnore(dir)
	want := []string{"*.min.js", "fixtures/", "*.generated.ts"}
	if len(patterns) != len(want) {
		t.Fatalf("got %d patterns, want %d: %v", len(patterns), len(want), patterns)
	}
	for i, p := range patterns {
		if p != want[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, p, want[i])
		}
	}
}

// ── smartTruncate ─────────────────────────────────────────────────────────────

func TestSmartTruncate_ShortContent(t *testing.T) {
	content := "short content"
	got, truncated := smartTruncate(content, 80000)
	if truncated {
		t.Error("short content should not be truncated")
	}
	if got != content {
		t.Error("short content should be returned unchanged")
	}
}

func TestSmartTruncate_CutsAtBlankLine(t *testing.T) {
	// Build content that exceeds limit with a blank line in the search window
	base := strings.Repeat("x", 76000) // below limit
	paragraph1 := "\n\nfunc foo() {\n\treturn 42\n}\n"
	padding := strings.Repeat("y", 4000) // push total over 80000
	content := base + paragraph1 + padding

	got, truncated := smartTruncate(content, 80000)
	if !truncated {
		t.Error("content over limit should be truncated")
	}
	if strings.Contains(got, padding) {
		t.Error("truncated content should not contain padding after the cut")
	}
	if !strings.Contains(got, "truncated") {
		t.Error("truncated content should contain a truncation marker")
	}
}

func TestSmartTruncate_NeverExceedsLimit(t *testing.T) {
	// Worst case: no blank lines at all — should still not exceed maxChars significantly
	content := strings.Repeat("a", 100000)
	got, truncated := smartTruncate(content, 80000)
	if !truncated {
		t.Error("content over limit should be truncated")
	}
	// Allow for the truncation marker appended at the end
	if len(got) > 80000+100 {
		t.Errorf("truncated content too long: %d chars", len(got))
	}
}

func TestShouldSkipManagedDir_WhenScanningProjectRoot(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}
	a := New(cfg, wiki.New(cfg.Paths.Wiki))

	for _, dir := range []string{cfg.Paths.Raw, cfg.Paths.Wiki, cfg.Paths.KnowledgeBase} {
		if !a.shouldSkipManagedDir(dir, project) {
			t.Fatalf("expected %s to be skipped when scanning %s", dir, project)
		}
	}
}

func TestShouldSkipManagedDir_WhenScanningRawOnly(t *testing.T) {
	project := t.TempDir()
	cfg := &config.Config{
		ProjectPath: project,
		Paths: config.PathsConfig{
			Raw:           filepath.Join(project, "raw"),
			Wiki:          filepath.Join(project, "wiki"),
			KnowledgeBase: filepath.Join(project, "knowledge-base"),
		},
	}
	a := New(cfg, wiki.New(cfg.Paths.Wiki))

	if a.shouldSkipManagedDir(cfg.Paths.Raw, cfg.Paths.Raw) {
		t.Fatal("raw source root should not skip itself")
	}
	for _, dir := range []string{cfg.Paths.Wiki, cfg.Paths.KnowledgeBase} {
		if !a.shouldSkipManagedDir(dir, cfg.Paths.Raw) {
			t.Fatalf("expected %s to be skipped when scanning raw", dir)
		}
	}
}
