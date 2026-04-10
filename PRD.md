# TheSecondBrain — Product Requirements Document

**Version:** 1.0  
**Date:** 2026-04-10  
**Status:** In Development

---

## 1. Problem Statement

Knowledge workers accumulate sources — papers, docs, articles, code — but fail to synthesize them. Existing AI tools (ChatGPT uploads, NotebookLM, RAG pipelines) re-derive answers from raw files on every query. Nothing is built up. Ask a nuanced follow-up question and the system has no memory of what it already figured out.

**The gap:** there is no tool that *compiles* knowledge from sources into a maintained, evolving wiki — one that gets smarter over time, not just faster at searching.

---

## 2. Product Vision

> A personal knowledge vault that compounds. The LLM writes and maintains everything. You curate sources and ask questions.

TheSecondBrain is a terminal-native tool that transforms a directory of raw files into a structured, interlinked wiki — maintained by an LLM, queryable in natural language, with a full audit trail for every change.

---

## 3. Target Users

| Persona | Use Case |
|---------|----------|
| **Researcher** | Building a knowledge base on a topic over weeks/months from papers and articles |
| **Developer** | Documenting an existing codebase or keeping project knowledge current |
| **Knowledge Worker** | Capturing and synthesising meeting notes, Slack threads, design docs |
| **Team Lead** | Maintaining shared team knowledge with full change traceability |

---

## 4. Core Requirements

### 4.1 Ingestion

| ID | Requirement | Priority |
|----|-------------|----------|
| ING-1 | Accept any file type in `raw/` (text, code, images, HTML) | P0 |
| ING-2 | Walk nested directories recursively | P0 |
| ING-3 | Skip unchanged files using content hash comparison | P0 |
| ING-4 | Produce 5–15 wiki pages per source (source + entities + concepts) | P0 |
| ING-5 | Auto-analyze new files within 3 seconds of being dropped | P1 |
| ING-6 | Analyze images using vision API and integrate descriptions | P1 |
| ING-7 | Log every ingest operation to `wiki/log.md` | P1 |

### 4.2 Wiki Quality

| ID | Requirement | Priority |
|----|-------------|----------|
| WIK-1 | All wiki pages have YAML frontmatter (type, title, tags, sources, dates) | P0 |
| WIK-2 | All internal references use `[[WikiLink]]` syntax | P0 |
| WIK-3 | Wiki pages extract *how things work*, not just *what they are* | P0 |
| WIK-4 | `wiki/index.md` auto-maintained after every ingest | P0 |
| WIK-5 | `wiki/log.md` append-only, every operation recorded | P0 |
| WIK-6 | `/lint` detects: orphan pages, broken links, stubs, contradictions | P1 |

### 4.3 Query

| ID | Requirement | Priority |
|----|-------------|----------|
| QRY-1 | Answers sourced strictly from wiki — never general LLM knowledge | P0 |
| QRY-2 | Chunks below minimum similarity threshold filtered before LLM | P0 |
| QRY-3 | Each cited reference includes LLM-calculated confidence score | P1 |
| QRY-4 | Last 6 conversation turns passed for follow-up awareness | P1 |
| QRY-5 | Conversational, varied responses when info is not in wiki | P1 |
| QRY-6 | Answers can be saved as synthesis pages (`/save <title>`) | P1 |

### 4.4 Corrections & Audit Trail

| ID | Requirement | Priority |
|----|-------------|----------|
| COR-1 | Detect correction intent in natural language | P1 |
| COR-2 | `/fixwiki <name> <correction>` for explicit corrections | P0 |
| COR-3 | Fuzzy page matching by name, path prefix, or slug | P1 |
| COR-4 | LLM analyzes consistency before applying any correction | P0 |
| COR-5 | User can force-apply against system recommendation | P0 |
| COR-6 | Every correction creates a record in `knowledge-base/amendments/` | P0 |
| COR-7 | Amendment record contains: original, proposed, analysis, status, date | P0 |
| COR-8 | `/amendments` lists all records with status icons | P1 |

### 4.5 TUI

| ID | Requirement | Priority |
|----|-------------|----------|
| TUI-1 | Terminal UI using Go + Bubble Tea | P0 |
| TUI-2 | Streaming output (token by token) | P0 |
| TUI-3 | Scrollable chat history (mouse wheel + PgUp/PgDn) | P0 |
| TUI-4 | Auto-follow new messages unless user has scrolled up | P0 |
| TUI-5 | Command history navigation (↑/↓ arrows) | P1 |
| TUI-6 | Clipboard copy of last answer (Ctrl+Y) | P1 |
| TUI-7 | Shell passthrough (`!<command>` runs from project dir) | P1 |
| TUI-8 | File-in-chat detection (mention a path → auto-copy to `raw/`) | P2 |
| TUI-9 | Setup wizard on first run (API key only) | P0 |
| TUI-10 | Live stats in header (wiki page count, KB chunks, watcher status) | P1 |

### 4.6 Configuration

| ID | Requirement | Priority |
|----|-------------|----------|
| CFG-1 | Global config at `~/.config/secondbrain/` (API key, model settings) | P0 |
| CFG-2 | Project path = CWD (like git — `brain` operates where invoked) | P0 |
| CFG-3 | `raw/`, `wiki/`, `knowledge-base/` created automatically in project dir | P0 |
| CFG-4 | `/logout` removes entire global config dir | P0 |
| CFG-5 | `/config reset` triggers setup on next launch | P0 |

---

## 5. Non-Goals (v1.0)

- No web UI or browser interface
- No multi-user / cloud sync (single-user, local files)
- No PDF text extraction (manual extraction required)
- No Git integration for the vault itself (user's responsibility)
- No scheduled/cron-based syncing (event-driven only)
- No native mobile support

---

## 6. Technical Architecture

### Stack

| Component | Technology |
|-----------|-----------|
| TUI framework | `charmbracelet/bubbletea` + `bubbles` + `lipgloss` |
| LLM (analysis + answers) | OpenAI-compatible API — `gpt-4o` via Rakuten AI Gateway |
| Embeddings | `text-embedding-3-small` via Rakuten AI Gateway |
| Vector store | Flat JSON + cosine similarity (no external deps) |
| File watching | `fsnotify/fsnotify` |

### Data Flow

```
raw/ file
  → [Analyzer: LLM] → wiki/ pages (5–15 per source)
  → [Embeddings: Rakuten] → knowledge-base/embeddings/store.json
  → [Query: LLM + cosine search] → answer with confidence-scored refs
  → [Correction: LLM] → knowledge-base/amendments/ audit record
```

### Similarity Pipeline

1. Embed question → cosine search → top-K results
2. Filter: drop chunks below `min_similarity` threshold (default 0.25)
3. Send filtered context to LLM with similarity scores visible
4. LLM answers and self-scores each reference by actual contribution

---

## 7. Success Metrics

| Metric | Target |
|--------|--------|
| Pages created per source | 5–15 |
| Reference precision | LLM only cites pages it actually used |
| Setup time (first run) | < 60 seconds |
| Analysis latency per file | < 30 seconds |
| Build status | Always clean (`go build ./...`) |
| Amendment audit coverage | 100% of applied corrections recorded |

---

## 8. Open Items / Future Considerations

- [ ] PDF text extraction (pdfcpu or similar)
- [ ] `/share` command to export wiki as static HTML
- [ ] Support for multiple vaults (configured vault list vs. CWD-only)
- [ ] Incremental re-analysis (only changed sections of large files)
- [ ] Plugin system for custom analyzers (e.g. YouTube transcript → wiki)
- [ ] `/search <query>` — pure keyword search without LLM (for large wikis)
- [ ] Graph view export (wiki link graph as dot/mermaid)
