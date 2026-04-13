package analyzer

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ORG028658/TheSecondBrain/tui/internal/config"
	"github.com/ORG028658/TheSecondBrain/tui/internal/wiki"
	openai "github.com/sashabaranov/go-openai"
)

type Analyzer struct {
	llm  *openai.Client
	cfg  *config.Config
	wiki *wiki.Wiki
}

// AnalysisResponse is what the LLM returns for a single source — multiple pages.
type AnalysisResponse struct {
	Pages    []PageResult `json:"pages"`
	LogEntry string       `json:"log_entry"`
}

// PageResult is one wiki page to create or update.
type PageResult struct {
	WikiPath string `json:"wiki_path"`
	Content  string `json:"content"`
	Action   string `json:"action"` // "create", "update", "skip"
}

type SourceMeta struct {
	Hash       string   `json:"hash"`
	AnalyzedAt string   `json:"analyzed_at"`
	WikiPages  []string `json:"wiki_pages"`
}

func New(cfg *config.Config, w *wiki.Wiki) *Analyzer {
	c := openai.DefaultConfig(os.Getenv("LLM_COMPATIBLE_API_KEY"))
	c.BaseURL = cfg.LLM.BaseURL
	return &Analyzer{
		llm:  openai.NewClientWithConfig(c),
		cfg:  cfg,
		wiki: w,
	}
}

// AnalyzeAll scans raw/ and processes new or changed files.
// Each file may produce multiple wiki pages (source + entities + concepts).
func (a *Analyzer) AnalyzeAll(ctx context.Context, progress func(string)) (string, error) {
	rawPath := a.cfg.Paths.Raw
	progress(fmt.Sprintf("Scanning: %s", rawPath))

	// Verify the raw directory exists
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		os.MkdirAll(rawPath, 0755) //nolint:errcheck
		return "raw/ directory was empty — it has been created at " + rawPath, nil
	}

	sourcesPath := filepath.Join(a.cfg.Paths.KnowledgeBase, "metadata", "sources.json")
	sources := loadSources(sourcesPath)

	var created, updated, skipped int

	err := filepath.WalkDir(rawPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		rel, _ := filepath.Rel(a.cfg.Paths.Raw, path)
		h := fileHash(path)

		if meta, ok := sources[rel]; ok && meta.Hash == h {
			skipped++
			return nil
		}

		progress(fmt.Sprintf("Analyzing: %s", rel))
		resp, err := a.analyzeFile(ctx, path, rel)
		if err != nil {
			progress(fmt.Sprintf("⚠ %s: %v", rel, err))
			return nil
		}

		var wikiPages []string
		for _, page := range resp.Pages {
			if page.Action == "skip" {
				continue
			}
			if err := a.wiki.Write(page.WikiPath, page.Content); err != nil {
				progress(fmt.Sprintf("⚠ write %s: %v", page.WikiPath, err))
				continue
			}
			wikiPages = append(wikiPages, page.WikiPath)
			if page.Action == "create" {
				created++
				progress(fmt.Sprintf("  ✓ created  %s", page.WikiPath))
			} else {
				updated++
				progress(fmt.Sprintf("  ✓ updated  %s", page.WikiPath))
			}
		}

		if len(wikiPages) == 0 {
			skipped++
		} else {
			sources[rel] = SourceMeta{Hash: h, AnalyzedAt: nowISO(), WikiPages: wikiPages}
			if resp.LogEntry != "" {
				a.wiki.AppendLog(resp.LogEntry) //nolint:errcheck
			}
		}
		return nil
	})

	saveSources(sourcesPath, sources)
	a.rebuildIndex()

	return fmt.Sprintf("Done — %d created, %d updated, %d skipped", created, updated, skipped), err
}

// LintWiki asks the LLM to health-check the wiki.
func (a *Analyzer) LintWiki(ctx context.Context) (string, error) {
	indexContent, _ := a.wiki.ReadIndex()
	pages, err := a.wiki.ListPages()
	if err != nil {
		return "", err
	}

	var allContent strings.Builder
	for _, p := range pages {
		content, _ := a.wiki.Read(p)
		allContent.WriteString(fmt.Sprintf("\n\n=== %s ===\n%s", p, content))
	}

	prompt := fmt.Sprintf(`Perform a health check on this wiki.

Index:
%s

All pages:
%s

Check for:
- Broken wikilinks ([[PageName]] links to pages that do not exist)
- Orphan pages (no inbound wikilinks from other wiki pages)
- Stub pages (very thin content, fewer than 3 meaningful items)
- Missing pages (concepts or entities mentioned in multiple places but lacking their own page)
- Contradictions between pages
- Stale content superseded by newer sources
- Index gaps (pages not listed in index.md)

Return a markdown health report with sections: ## Critical / ## Minor / ## Suggestions`,
		indexContent, allContent.String())

	resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     a.cfg.LLM.Model,
		MaxTokens: 2048,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	report := resp.Choices[0].Message.Content
	a.wiki.AppendLog(fmt.Sprintf("## [%s] lint\n%s", today(), firstLine(report))) //nolint:errcheck
	return report, nil
}

// AnalyzeAmendment checks whether a proposed correction is consistent with or
// contradicts the current wiki content. Returns analysis text and a consistency flag.
func (a *Analyzer) AnalyzeAmendment(ctx context.Context, wikiPath, currentContent, proposed string) (analysis string, consistent bool, err error) {
	excerpt := currentContent
	if len(excerpt) > 1500 {
		excerpt = excerpt[:1500] + "\n...[truncated]"
	}

	prompt := fmt.Sprintf(`Analyze a proposed correction to a wiki page.

Wiki page: %s

Current content:
%s

Proposed correction:
"%s"

Evaluate:
1. Is the proposed correction CONSISTENT or CONTRADICTORY with the existing content?
2. Is it logically sound?
3. Brief impact assessment.

Response format (exactly):
Line 1: CONSISTENT  or  CONTRADICTORY
Lines 2+: 2-3 sentences of reasoning.`, wikiPath, excerpt, proposed)

	resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     a.cfg.LLM.Model,
		MaxTokens: 400,
		Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: prompt}},
	})
	if err != nil {
		return "", false, err
	}
	analysis = resp.Choices[0].Message.Content
	consistent = strings.HasPrefix(strings.ToUpper(strings.TrimSpace(analysis)), "CONSISTENT")
	return analysis, consistent, nil
}

// CorrectPage rewrites a wiki page incorporating a user-verified correction.
// Returns the new page content (not written to disk — caller writes after confirmation).
func (a *Analyzer) CorrectPage(ctx context.Context, wikiPath, currentContent, correction string) (string, error) {
	prompt := fmt.Sprintf(`You are correcting a wiki page based on a user-verified correction.

Current page (%s):
%s

User correction:
%s

Rewrite the page incorporating the correction. Keep the same format, frontmatter, and sections. Only change what the correction specifies. Return ONLY the updated page markdown — no explanation, no JSON.`,
		wikiPath, currentContent, correction)

	resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     a.cfg.LLM.Model,
		MaxTokens: a.cfg.LLM.MaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// --- file routing ---

func (a *Analyzer) analyzeFile(ctx context.Context, path, relPath string) (*AnalysisResponse, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return a.analyzeImage(ctx, path, relPath)
	case ".pdf":
		return &AnalysisResponse{
			Pages:    []PageResult{{WikiPath: "", Action: "skip"}},
			LogEntry: "",
		}, nil
	default:
		return a.analyzeTextFile(ctx, path, relPath)
	}
}

func (a *Analyzer) analyzeTextFile(ctx context.Context, path, relPath string) (*AnalysisResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	if len(content) > 80000 {
		content = content[:80000] + "\n...[truncated]"
	}

	indexContent, _ := a.wiki.ReadIndex()
	userPrompt := fmt.Sprintf(
		"Ingest this file into the wiki.\n\nFile: %s\n\nContent:\n```\n%s\n```\n\nReturn ONLY valid JSON.",
		relPath, content)

	return a.callLLM(ctx, analyzerSystemPrompt(indexContent), nil, userPrompt)
}

func (a *Analyzer) analyzeImage(ctx context.Context, path, relPath string) (*AnalysisResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if ext == "jpg" {
		ext = "jpeg"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:image/%s;base64,%s", ext, b64)

	indexContent, _ := a.wiki.ReadIndex()
	userPrompt := fmt.Sprintf(
		"Ingest this image into the wiki. Describe its content thoroughly and extract knowledge.\nFile: %s\nReturn ONLY valid JSON.",
		relPath)

	imagePart := openai.ChatMessagePart{
		Type:     openai.ChatMessagePartTypeImageURL,
		ImageURL: &openai.ChatMessageImageURL{URL: dataURL},
	}
	textPart := openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: userPrompt,
	}
	return a.callLLM(ctx, analyzerSystemPrompt(indexContent), []openai.ChatMessagePart{imagePart, textPart}, "")
}

// callLLM sends to the LLM and parses the multi-page AnalysisResponse.
func (a *Analyzer) callLLM(ctx context.Context, system string, parts []openai.ChatMessagePart, text string) (*AnalysisResponse, error) {
	var userMsg openai.ChatCompletionMessage
	if len(parts) > 0 {
		userMsg = openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, MultiContent: parts}
	} else {
		userMsg = openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: text}
	}

	resp, err := a.llm.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     a.cfg.LLM.Model,
		MaxTokens: a.cfg.LLM.MaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			userMsg,
		},
	})
	if err != nil {
		return nil, err
	}

	raw := extractJSON(resp.Choices[0].Message.Content)
	var result AnalysisResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w\nRaw: %.300s", err, raw)
	}
	return &result, nil
}

func (a *Analyzer) rebuildIndex() {
	pages, err := a.wiki.ListPages()
	if err != nil {
		return
	}
	var infos []wiki.PageInfo
	for _, p := range pages {
		content, err := a.wiki.Read(p)
		if err != nil {
			continue
		}
		infos = append(infos, wiki.PageInfo{
			RelPath:     p,
			Title:       extractTitle(content),
			Description: extractFirstLine(content),
		})
	}
	a.wiki.UpdateIndex(infos) //nolint:errcheck
}

// --- system prompt ---

func analyzerSystemPrompt(indexContent string) string {
	return fmt.Sprintf(`You are a knowledge extractor maintaining a personal wiki vault.

For each raw source you process, create MULTIPLE wiki pages:
1. wiki/sources/[slug].md — summary page for this source
2. wiki/entities/[slug].md — one page per significant person, org, product, tool, or place
3. wiki/concepts/[slug].md — one page per significant idea, pattern, theory, or technique

A single source typically results in 5–15 pages total.

KNOWLEDGE EXTRACTION RULES:
- Extract the actual knowledge — state concepts directly, in your own words
- Never copy-paste sentences from the source
- Never write "the author explains..." or "this file contains..."
- Each concept page must explain HOW IT WORKS, not just what it is
- Include [[WikiLink]] syntax for ALL internal references between pages
- Every page needs YAML frontmatter: type, title, tags, sources, created, updated

WIKILINKS: Use [[PageTitle]] for every reference to another wiki page.
Example: "This pattern relies on [[Dependency Injection]] as described in [[Clean Architecture]]."

YAML FRONTMATTER (required on every page):
---
type: source|entity|concept|synthesis
title: Page Title
tags: [tag1, tag2]
sources: [raw/filename]
created: %s
updated: %s
---

DEDUPLICATION: Check the current wiki index below. If a page already exists for this entity/concept, action="update" and merge new knowledge in. If nothing new to add, action="skip".

Current wiki index:
%s

REQUIRED JSON OUTPUT (no markdown fences, no prose outside the JSON):
{
  "pages": [
    {
      "wiki_path": "wiki/sources/slug.md",
      "content": "---\ntype: source\n...\n---\n\n# Title\n...",
      "action": "create"
    },
    {
      "wiki_path": "wiki/entities/name.md",
      "content": "---\ntype: entity\n...\n---\n\n# Name\n...",
      "action": "create"
    },
    {
      "wiki_path": "wiki/concepts/concept-name.md",
      "content": "---\ntype: concept\n...\n---\n\n# Concept Name\n...",
      "action": "update"
    }
  ],
  "log_entry": "## [%s] ingest | Source Title\nCreated: sources/slug.md. Updated: concepts/x, entities/y."
}`, today(), today(), indexContent, today())
}

// --- helpers ---

func fileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"```json", "```"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			s = strings.TrimSuffix(strings.TrimSpace(s), "```")
			s = strings.TrimSpace(s)
			break
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Untitled"
}

func extractFirstLine(content string) string {
	inFrontmatter := false
	for _, line := range strings.Split(content, "\n") {
		if line == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}
		if inFrontmatter || strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) > 100 {
			return line[:100] + "..."
		}
		return line
	}
	return ""
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func today() string {
	return time.Now().UTC().Format("2006-01-02")
}

func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

type sourcesFile map[string]SourceMeta

func loadSources(path string) sourcesFile {
	sources := sourcesFile{}
	data, err := os.ReadFile(path)
	if err != nil {
		return sources
	}
	json.Unmarshal(data, &sources) //nolint:errcheck
	return sources
}

func saveSources(path string, sources sourcesFile) {
	os.MkdirAll(filepath.Dir(path), 0755) //nolint:errcheck
	data, _ := json.MarshalIndent(sources, "", "  ")
	os.WriteFile(path, data, 0644) //nolint:errcheck
}
