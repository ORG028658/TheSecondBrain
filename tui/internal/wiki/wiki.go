package wiki

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Wiki struct {
	root string
}

type PageInfo struct {
	RelPath     string
	Title       string
	Description string
}

func New(root string) *Wiki {
	return &Wiki{root: root}
}

func (w *Wiki) Write(relPath, content string) error {
	fullPath, err := w.fullPath(relPath)
	if err != nil {
		return err
	}
	if err := w.guardPath(fullPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

func (w *Wiki) Read(relPath string) (string, error) {
	fullPath, err := w.fullPath(relPath)
	if err != nil {
		return "", err
	}
	if err := w.guardPath(fullPath); err != nil {
		return "", err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (w *Wiki) fullPath(relPath string) (string, error) {
	normalized, err := normalizeRelPath(relPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(w.root, normalized), nil
}

func normalizeRelPath(relPath string) (string, error) {
	trimmed := strings.TrimSpace(relPath)
	if trimmed == "" {
		return "", fmt.Errorf("wiki: empty path")
	}

	slashed := filepath.ToSlash(trimmed)
	slashed = strings.TrimPrefix(slashed, "./")
	slashed = strings.TrimPrefix(slashed, "/")
	slashed = strings.TrimPrefix(slashed, "wiki/")

	cleaned := filepath.Clean(filepath.FromSlash(slashed))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("wiki: empty path")
	}
	return cleaned, nil
}

func canonicalPagePath(relPath string) string {
	return "wiki/" + filepath.ToSlash(relPath)
}

// guardPath ensures the resolved absolute path stays inside the wiki root.
// This prevents LLM-supplied paths like "../../.ssh/authorized_keys" from
// escaping the vault directory.
func (w *Wiki) guardPath(fullPath string) error {
	cleanRoot := filepath.Clean(w.root) + string(os.PathSeparator)
	abs, err := filepath.Abs(fullPath)
	if err != nil || !strings.HasPrefix(abs, cleanRoot) {
		return fmt.Errorf("wiki: path escapes vault root: %s", fullPath)
	}
	return nil
}

func (w *Wiki) Exists(relPath string) bool {
	fullPath, err := w.fullPath(relPath)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullPath)
	return err == nil
}

func (w *Wiki) ContentHash(relPath string) string {
	fullPath, err := w.fullPath(relPath)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	return HashBytes(data)
}

func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

// ListPages returns all .md pages excluding index.md and log.md, sorted alphabetically.
func (w *Wiki) ListPages() ([]string, error) {
	var pages []string
	err := filepath.WalkDir(w.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(w.root, path)
		if rel != "index.md" && rel != "log.md" {
			pages = append(pages, canonicalPagePath(rel))
		}
		return nil
	})
	sort.Strings(pages)
	return pages, err
}

func (w *Wiki) ReadIndex() (string, error) {
	return w.Read("index.md")
}

// AppendLog appends an entry to wiki/log.md (creates if missing).
func (w *Wiki) AppendLog(entry string) error {
	logPath := filepath.Join(w.root, "log.md")

	// Create with header if it doesn't exist
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		header := "# Wiki Log\n\nAppend-only record of all wiki operations.\n\n---\n\n"
		if err := os.WriteFile(logPath, []byte(header), 0644); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + strings.TrimSpace(entry) + "\n")
	return err
}

// UpdateIndex regenerates index.md from the provided page list.
func (w *Wiki) UpdateIndex(pages []PageInfo) error {
	var sb strings.Builder
	sb.WriteString("# Wiki Index\n")
	sb.WriteString(fmt.Sprintf("Last updated: %s | Pages: %d\n\n", time.Now().Format("2006-01-02"), len(pages)))

	groups := map[string][]PageInfo{}
	for _, p := range pages {
		normalized := strings.TrimPrefix(filepath.ToSlash(p.RelPath), "wiki/")
		parts := strings.SplitN(normalized, "/", 2)
		if len(parts) == 0 {
			continue
		}
		groups[parts[0]] = append(groups[parts[0]], p)
	}

	for _, group := range []string{"sources", "entities", "concepts", "synthesis"} {
		ps, ok := groups[group]
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s\n", strings.Title(group))) //nolint:staticcheck
		for _, p := range ps {
			sb.WriteString(fmt.Sprintf("- [[%s]](%s) — %s\n", p.Title, p.RelPath, p.Description))
		}
		sb.WriteString("\n")
	}

	return w.Write("index.md", sb.String())
}

// PageCount returns the number of wiki content pages (excluding index.md and log.md).
func (w *Wiki) PageCount() int {
	pages, err := w.ListPages()
	if err != nil {
		return 0
	}
	return len(pages)
}
