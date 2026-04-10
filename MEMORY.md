# TheSecondBrain — Development Memory

Append-only log of development sessions, decisions, and progress.  
Format: `## [YYYY-MM-DD] Phase | Summary`

---

## [2026-04-09] Phase 1 | Architecture Design & Initial Scaffold

**Status:** Complete

### Decisions Made
- **Pattern:** Three-layer architecture — `raw/` (immutable sources), `wiki/` (LLM-maintained), `knowledge-base/` (RAG layer)
- **Directory structure:** Nested wiki with topic-based subdirs — `wiki/projects/`, `wiki/topics/`, `wiki/concepts/`, `wiki/people/`
- **LLM split:** Rakuten AI Gateway for embeddings (OpenAI-compatible), Claude/OpenAI for analysis and orchestration
- **TUI:** Go + Bubble Tea — fast single binary, no runtime deps
- **Vector store:** Flat JSON with cosine similarity — zero external dependencies, swappable later
- **Trigger:** `/pull` command (not file-watcher initially)

### Files Created
- `CLAUDE.md` — LLM schema and wiki conventions
- `config.yaml`, `.env.template`, `.gitignore`
- `sync.sh` — standalone git sync helper
- `tui/` — complete Go application scaffold
  - `internal/config/`, `internal/store/`, `internal/embeddings/`
  - `internal/rag/`, `internal/analyzer/`, `internal/wiki/`
  - `internal/ui/app.go`, `internal/ui/styles.go`

### Key Technical Notes
- `strings.Builder` in Go panics when copied by value — Bubble Tea copies the model struct on every Update; use plain `string` fields instead
- Bubble Tea pointer receivers (`*Model`) prevent struct copies entirely — the canonical fix

---

## [2026-04-09] Phase 2 | Wiki Structure Alignment with LLM Wiki Pattern

**Status:** Complete

### Decisions Made
- Revised wiki subdirectory structure to match reference pattern:
  - **Before:** `projects/`, `topics/`, `concepts/`, `people/`
  - **After:** `sources/`, `entities/`, `concepts/`, `synthesis/`
- `wiki/index.md` (renamed from `_index.md`) — master catalog
- `wiki/log.md` (moved from project root) — operation history
- `knowledge-base/amendments/` — new audit trail directory

### Key Feature: Multi-Page Ingest
- **Before:** 1 raw file → 1 wiki page
- **After:** 1 raw file → 5–15 wiki pages (source summary + entity pages + concept pages)
- LLM returns `{"pages": [...], "log_entry": "..."}` array response
- All pages wikilinked with `[[PageName]]` syntax
- All pages have YAML frontmatter (type, title, tags, sources, created, updated)

### Knowledge Extraction Standard
- LLM is a **knowledge extractor, not a summarizer**
- Must state *how things work*, not just *what they are*
- No copy-paste from source — original synthesis in every page
- Prompt includes explicit examples of bad vs. good output

---

## [2026-04-09] Phase 3 | UX & Usability Pass

**Status:** Complete

### Features Added
- **File watcher** (`fsnotify`) — auto-analyzes files dropped into `raw/` after 3-second debounce
- **Auto-create directories** — `ensureVaultStructure()` runs on every startup
- **Shell passthrough** — `!<command>` runs via `bash -c` from project directory (pipes, `&&`, `cd` work)
- **Command history** — `↑`/`↓` arrows navigate previous inputs
- **Clipboard** — `Ctrl+Y` copies last answer (`atotto/clipboard`)
- **Scroll fix** — `atBottom bool` flag: user can scroll up freely, new messages don't force-scroll back down
- **Random verbals** — varied thinking phrases (14 options) during analysis
- **Brain logo** — `🧠  TheSecondBrain` with `◈` neuro-node stats in header
- **Error hints** — every error includes a specific fix suggestion

### Install
- `install.sh` — builds binary and installs to `/usr/local/bin/brain`
- First run: setup wizard asks for API key only (no vault path — uses CWD)

---

## [2026-04-09 → 2026-04-10] Phase 4 | Config Architecture Refactor

**Status:** Complete

### Problem
- Config stored `vault_path` pointing to a fixed directory
- Shell commands ran from the wrong directory
- `!ls raw` showed files but `/analyze` found 0 — path mismatch

### Solution: Git-style Project Model
- **Global config** (`~/.config/secondbrain/config.yaml`) — API key, model settings only. Never project-specific.
- **Project path** = CWD where `brain` is invoked — all `raw/`, `wiki/`, `knowledge-base/` live here
- `brain` in `/Users/me/project-a` → uses `project-a/raw/`, `project-a/wiki/`
- `brain` in `/Users/me/project-b` → uses `project-b/raw/`, `project-b/wiki/`
- Same global API key and model settings for all projects

### New Commands
- `/config` — shows full config dir path, project dir, model settings
- `/config reset` — removes `~/.config/secondbrain/` entirely
- `/logout` — same as reset
- `/status` — now shows `Project dir` and `Config dir` separately

### Root Cause of 0-files Bug
- Vault at `~/SecondBrain/raw/` was empty
- User had files in `TheSecondBrain/raw/` (the dev project dir)
- Fix: copied files to the correct vault location; added `Scanning: <path>` output to analyzer

---

## [2026-04-10] Phase 5 | Conversation Intelligence & Wiki Corrections

**Status:** Complete

### Features Added

#### Conversation Context
- Last 6 turns (3 exchanges) passed to every LLM query
- Follow-up questions ("what about its performance?") work correctly
- Conversation history stored in `Model.convHistory []rag.ConvMsg`

#### Wiki Corrections
- **Natural language detection** — phrases like "that's wrong", "fix the wiki", "not correct" trigger correction flow
- **`/fixwiki <name> <correction>`** — explicit correction by name, slug, or path
- **Fuzzy page matching** — finds `wiki/concepts/transformer-architecture.md` from "transformer"
- **Contradiction analysis** — LLM analyses CONSISTENT vs CONTRADICTORY before applying
- **Force-apply** — `force` at confirmation prompt overrides system recommendation
- **`/gap <topic>`** — creates research stub in `wiki/sources/gap-*.md`
- **`/amendments`** — lists all amendment records with status icons

#### Amendment Audit Trail
- Every correction creates `knowledge-base/amendments/YYYYMMDD-HHmmss-slug.md`
- Record contains: original excerpt, proposed change, system analysis, consistency verdict, status
- Statuses: `applied`, `force-applied`
- Contradiction shown in footer: `⚠ contradicts source` when `isConsistent = false`

#### Strict Wiki-Only Answers
- System prompt updated: never use outside knowledge
- Varied "not in wiki" responses (5 rotating phrases)
- Suggests `/gap <topic>` when info is missing
- Conversational tone — not a fixed error message

---

## [2026-04-10] Phase 6 | Relevancy & Reference Quality

**Status:** Complete

### Problem
- Incorrect references included alongside correct ones
- All top-K chunks listed as references regardless of relevance
- No differentiation between "used this source" and "this source appeared in context"

### Solution: Two-Layer Relevancy

#### Layer 1: Similarity Threshold Filtering
- New config field: `rag.min_similarity: 0.25`
- Chunks below threshold dropped before reaching LLM
- Prevents noise from entering the context window entirely
- Configurable: raise to 0.35–0.40 for stricter relevance

#### Layer 2: LLM Confidence Scoring
- System prompt updated: LLM self-scores each reference (0–100%) based on actual contribution to answer
- Format: `→ wiki/path.md  [92%] — directly addresses the question`
- LLM instructed to omit sources it didn't use — not just list everything in context
- Cosine similarity score passed to LLM in context: `[wiki: path | similarity: 87%]`

### Why LLM Scoring > Cosine Similarity
- Cosine measures topical similarity, not answer contribution
- LLM knows which chunks it actually drew from to compose the answer
- Result: references reflect usage, not just relevance ranking

---

## Current Status (2026-04-10)

### Build Status
```
go build ./...  →  BUILD OK
go vet ./...    →  VET OK
```

### Features Complete
- [x] Multi-page ingest (5–15 pages per source)
- [x] YAML frontmatter + wikilinks on all pages
- [x] Auto-watch file watcher
- [x] Streaming output with follow-up context
- [x] Similarity threshold filtering
- [x] LLM confidence-scored references
- [x] Wiki corrections with contradiction detection
- [x] Amendment audit trail
- [x] Force-apply override
- [x] Research gap tracking (`/gap`)
- [x] Synthesis filing (`/save`)
- [x] Command history + clipboard + shell passthrough
- [x] Setup wizard (first run)
- [x] CWD-based project model (like git)
- [x] `/config`, `/logout`, `/amendments`, `/fixwiki`, `/gap`
- [x] README.md, PRD.md

### Known Limitations
- PDF files: skipped (manual text extraction required)
- No incremental re-analysis (full file re-analyzed on change)
- Force-scroll: fixed with `atBottom` flag but mouse scroll requires `tea.WithMouseCellMotion()`

### File Counts
```
tui/internal/  →  9 Go files  ~52K total
wiki/          →  8+ pages (grows with usage)
knowledge-base/amendments/  →  per correction
```

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-09 | Flat JSON vector store (not sqlite-vec) | Personal wiki scale (<5K chunks); zero deps; swappable |
| 2026-04-09 | Pointer receivers on `*Model` | Prevents `strings.Builder` copy panic in Bubble Tea |
| 2026-04-09 | No Anthropic SDK — all via go-openai | Single API key (Rakuten AI Gateway); OpenAI-compatible |
| 2026-04-09 | Multi-page ingest via JSON array response | Reference pattern requires 10–15 pages per source |
| 2026-04-10 | CWD = project path (no `vault_path` in config) | git model — predictable, no setup friction |
| 2026-04-10 | LLM self-scores references (not cosine scores) | Cosine measures topical similarity; LLM knows what it used |
| 2026-04-10 | Amendment audit always created on correction | Full traceability; user can force-apply but record is always kept |
