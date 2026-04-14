package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTempWiki(t *testing.T) *Wiki {
	t.Helper()
	dir := t.TempDir()
	return New(filepath.Join(dir, "wiki"))
}

// ── guardPath / path traversal ────────────────────────────────────────────────

func TestWrite_PathTraversal(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	// Note: Go's filepath.Join strips leading slashes from inner segments,
	// so "/etc/passwd" resolves as "{root}/etc/passwd" — not a traversal.
	// The real traversal vector is "../" sequences which DO escape the root.
	cases := []struct {
		name    string
		relPath string
	}{
		{"parent dir escape", "../../etc/passwd"},
		{"double dot in middle", "sources/../../../etc/passwd"},
		{"many dots", "../../../../../etc/shadow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := w.Write(tc.relPath, "malicious")
			if err == nil {
				t.Errorf("Write(%q) should have been rejected but succeeded", tc.relPath)
				return // prevent nil dereference below
			}
			if !strings.Contains(err.Error(), "escapes vault root") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWrite_ValidPaths(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	cases := []string{
		"index.md",
		"wiki/index.md",
		"sources/my-source.md",
		"wiki/sources/my-source.md",
		"concepts/sub/deep.md",
		"wiki/concepts/sub/deep.md",
	}
	for _, relPath := range cases {
		t.Run(relPath, func(t *testing.T) {
			if err := w.Write(relPath, "# content"); err != nil {
				t.Errorf("Write(%q) rejected a valid path: %v", relPath, err)
			}
		})
	}
}

func TestRead_PathTraversal(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	_, err := w.Read("../../etc/passwd")
	if err == nil {
		t.Error("Read should have rejected path traversal")
	}
	if !strings.Contains(err.Error(), "escapes vault root") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	const content = "# Hello\n\nThis is a test page."
	if err := w.Write("sources/test.md", content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	got, err := w.Read("wiki/sources/test.md")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != content {
		t.Errorf("round-trip mismatch:\n  got:  %q\n  want: %q", got, content)
	}
}

func TestWrite_WikiPrefixDoesNotNest(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	if err := w.Write("wiki/sources/prefixed.md", "# prefixed"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(w.root, "sources", "prefixed.md")); err != nil {
		t.Fatalf("expected canonical file location: %v", err)
	}
	if _, err := os.Stat(filepath.Join(w.root, "wiki", "sources", "prefixed.md")); !os.IsNotExist(err) {
		t.Fatalf("unexpected nested wiki/wiki path created")
	}
}

func TestListPages_ReturnsCanonicalWikiPrefix(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	if err := w.Write("sources/a.md", "# A"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := w.Write("wiki/concepts/b.md", "# B"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got, err := w.ListPages()
	if err != nil {
		t.Fatalf("ListPages failed: %v", err)
	}

	want := []string{"wiki/concepts/b.md", "wiki/sources/a.md"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ListPages mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

func TestExistsAndContentHashAcceptCanonicalPaths(t *testing.T) {
	w := newTempWiki(t)
	os.MkdirAll(w.root, 0755)

	const content = "# Canonical\n"
	if err := w.Write("wiki/entities/test.md", content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if !w.Exists("wiki/entities/test.md") {
		t.Fatalf("Exists should accept canonical wiki/ path")
	}
	if got := w.ContentHash("wiki/entities/test.md"); got != HashBytes([]byte(content)) {
		t.Fatalf("ContentHash mismatch: got %q", got)
	}
}
