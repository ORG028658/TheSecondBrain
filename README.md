# TheSecondBrain

**A personal knowledge vault that compounds over time — powered by LLMs, lived in the terminal.**

> *Inspired by the [LLM Wiki pattern](inspiration/llm-wiki.md) by Andrej Karpathy: instead of re-deriving knowledge on every query, the LLM incrementally builds and maintains a persistent, interlinked wiki — so knowledge accumulates rather than evaporates.*

---

## What Is This?

Most AI document tools work like search: drop files in, ask questions, the LLM retrieves relevant chunks and answers. The knowledge is never kept. Ask the same question tomorrow and it rediscovers from scratch.

**TheSecondBrain is different.** When you add a source, the LLM reads it and *writes wiki pages* — extracting concepts, entities, key learnings, and wiring them together with cross-references. Every question you ask enriches the wiki further. The knowledge compounds.

You curate sources and ask questions. The LLM writes and maintains everything else.

---

## Project Status

**Status:** Beta

TheSecondBrain is currently tuned for **internal/team use first** and careful external experimentation. It is useful today, but it is **not production-grade software yet**.

Current expectations:

- Core wiki path handling is tested and canonicalized around `wiki/...`
- CI runs build, vet, tests, and formatting checks on PRs
- Docs are kept aligned with current behavior
- Release tags produce downloadable binaries for macOS and Linux

Known limitations:

- **PDF ingestion is not supported yet**
- Vault repair for previously corrupted `wiki/wiki/...` layouts is warning-only for now
- This remains a single-user local tool, not a multi-user or hosted product

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      brain  (TUI)                           │
│          Go + Bubble Tea  ·  runs in current directory      │
└───────────────────┬─────────────────────────────────────────┘
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
   /analyze       Query      /fixwiki
  (ingest)       (RAG)      (correct)
        │           │           │
        ▼           ▼           │
┌───────────┐  ┌──────────┐    │
│   wiki/   │  │knowledge │◄───┘
│           │  │  -base/  │
│ sources/  │  │          │
│ entities/ │  │embeddings│
│ concepts/ │  │amendments│
│synthesis/ │  │ metadata │
│ index.md  │  └──────────┘
│  log.md   │
└───────────┘
        ▲
        │  LLM writes all wiki pages
        │
┌───────────┐
│   raw/    │  ← You drop anything here
│           │     (docs, images, code,
│ supported │      notes, repos)
│   files   │
└───────────┘

Global config: ~/.config/secondbrain/  (API key, model settings)
Project:       ./  (current directory — like git)
```

**Three layers:**

| Layer | Who owns it | What lives here |
|-------|-------------|-----------------|
| `raw/` | You | Source files — immutable, never modified by the brain |
| `wiki/` | LLM | Structured knowledge — concepts, entities, summaries, synthesis |
| `knowledge-base/` | System | Embeddings, metadata, amendment audit trail |

**Two config scopes:**

- **Global** (`~/.config/secondbrain/`) — API key, model settings. Shared across all projects.
- **Project** (current directory) — `raw/`, `wiki/`, `knowledge-base/`. Like a git repository: `brain` operates in wherever you invoke it.

---

## Top 3 Use Cases

### 1. Deep Research
Reading papers, articles, and reports on a topic over weeks? Drop each one into `raw/` as you go. The brain builds an interlinked wiki of concepts, entities, and findings — cross-referencing everything automatically. By the end you have a structured knowledge base, not a pile of highlights.

**Flow:** Drop paper → `/pull` → wiki pages created for concepts + authors + findings → ask follow-up questions → `/save` compelling syntheses

### 2. Codebase Documentation
Point the brain at a cloned repository. It reads the architecture, patterns, modules, and dependencies — and writes a living wiki that stays current as the code evolves. New teammates onboard by querying the brain rather than excavating the code.

**Flow:** Clone repo into `raw/` → `/pull` → wiki/projects/repo-name.md created with architecture, patterns, open questions → ask "how does the auth flow work?"

### 3. Team Knowledge Base
Feed the brain Slack threads, meeting transcripts, design docs, and customer call notes. It maintains a shared wiki that no one has to manually update — because the LLM does the bookkeeping no one wants to do.

**Flow:** Export Slack/Notion → drop into `raw/` → wiki grows with decisions, entities, and concepts → team queries the brain instead of searching Slack

---

## Getting Started

### Install

```bash
# Prerequisites: Go 1.22+
go install github.com/ORG028658/TheSecondBrain/tui@latest
```

This installs the `brain` binary into your Go bin directory (`$GOBIN` or `$(go env GOPATH)/bin`).

If that directory is not already on your `PATH`, add it:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

For local development from a checked-out repo, you can still use:

```bash
bash install.sh
```

### First Run

```bash
mkdir my-project && cd my-project
brain
```

On first launch, a setup wizard asks for your **Rakuten AI Gateway key**. It creates:
- `~/.config/secondbrain/config.yaml` — global settings
- `~/.config/secondbrain/.env` — API key (never committed)
- `raw/`, `wiki/`, `knowledge-base/` — in the current directory

### Supported Platforms

- macOS
- Linux
- Go version: see [`tui/go.mod`](tui/go.mod)

### Supported Inputs

Currently supported:

- markdown, text, HTML, and common config/data formats
- source code and repository trees
- common image formats (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`)

Not supported yet:

- PDF ingestion

### Basic Workflow

```bash
# 1. Drop files into raw/
cp ~/Downloads/research-notes.md raw/
cp -r ~/code/my-android-app raw/

# 2. Process (or files are auto-analyzed within 3 seconds of being dropped)
/pull

# Alternative: ingest an existing directory without copying files into raw/
# brain reads from the project root itself — useful for codebases, mono-repos, etc.
brain --current-dir      # session-level: all pulls use the project root as source
/pull --current-dir      # one-shot: single pull from the project root

# 3. Ask questions
What design patterns does my-android-app use?

# 4. Keep useful answers
/save Android Architecture Overview
```

---

## Features

### Ingest Pipeline
- **Many text, code, config, and image types** — markdown, text, code (`.kt`, `.py`, `.go`), HTML, YAML, JSON, and common images
- **Nested folders** — the entire `raw/` tree is walked, any depth
- **Multi-page ingest** — one source typically creates 5–15 wiki pages: a source summary, entity pages (people, tools, companies), concept pages, all interlinked
- **Auto-watch** — file watcher monitors `raw/`; new files trigger analysis automatically after a 3-second debounce
- **Hash-based change detection** — unchanged files are skipped; only new/modified files are processed
- **Image analysis** — images are described by vision AI and integrated into relevant wiki pages
- **Explicitly unsupported right now** — PDFs are skipped instead of partially parsed

### Wiki Structure
```
wiki/
  sources/       ← One summary page per raw source
  entities/      ← People, organisations, products, tools
  concepts/      ← Ideas, patterns, theories, techniques
  synthesis/     ← Filed query results and analyses
  index.md       ← Auto-maintained master catalog
  log.md         ← Append-only operation history
```

Every wiki page has:
- **YAML frontmatter** — type, title, tags, sources, created, updated
- **`[[WikiLink]]` syntax** — internal cross-references between pages
- **Knowledge extraction, not summarisation** — concepts are explained, not described

### RAG Query Engine
- **Similarity threshold filtering** — chunks below `min_similarity` (default 0.25) are discarded before reaching the LLM, removing noise
- **LLM confidence-scored references** — each cited reference includes an AI-calculated confidence % and reason (not just cosine similarity — the LLM scores based on actual usage)
- **Conversation history** — last 6 turns passed to every query for follow-up awareness
- **Strict wiki-only answers** — the LLM never uses outside knowledge; if it's not in the wiki it says so

### Wiki Corrections & Amendment Audit Trail
- **Natural language corrections** — say "that's wrong, it should be X" and the brain finds the relevant page, proposes a correction, and asks for confirmation
- **`/fixwiki <name> <correction>`** — explicit correction by page name or fuzzy match (e.g. `/fixwiki transformer activation should be ReLU`)
- **Contradiction detection** — before applying any correction, the LLM analyses whether it contradicts the current content (`CONSISTENT` / `CONTRADICTORY`)
- **Force-apply** — type `force` to override the system's recommendation; your data, your call
- **Amendment records** — every correction is recorded in `knowledge-base/amendments/YYYYMMDD-HHmmss-slug.md` with: original content, proposed change, system analysis, and outcome
- **`/amendments`** — list all amendment records with status icons (`✓` applied, `⚡` force-applied)

### Research Gap Tracking
- **`/gap <topic>`** — flags a missing topic and creates a research stub in `wiki/sources/` with sections for what's needed, suggested sources, and why it matters
- Varied, conversational "not in wiki" responses — not a fixed error message
- Stubs are indexed so they surface in future searches

### Synthesis Pages
- **`/save <title>`** — saves the last answer as `wiki/synthesis/slug.md` with proper frontmatter and source citations
- Filed syntheses are re-indexed and become searchable knowledge
- All saves logged to `wiki/log.md`

### TUI
- **Streaming output** — answers appear token by token
- **Scroll** — `PgUp`/`PgDn` to scroll chat history; auto-follows new messages unless you've scrolled up
- **Command history** — `↑`/`↓` arrows to navigate previous inputs (like a shell)
- **Clipboard** — `Ctrl+Y` copies the last answer
- **Shell passthrough** — `!<command>` runs any shell command from the project directory (pipes, `&&`, `cd` all work)
- **File-in-chat** — mention a file path (e.g. `/path/to/doc.md`) and it's automatically copied to `raw/` with an explanation
- **Brain logo** with live stats in the header — wiki page count, KB chunk count, watcher indicator

---

## Commands Reference

| Command | Description |
|---------|-------------|
| `/pull` | Full pipeline: scan `raw/` → extract knowledge → update `wiki/` → sync embeddings |
| `/pull --current-dir` | Same as `/pull` but uses the project root as the source directory instead of `raw/` |
| `/analyze` | Force re-analyze `raw/` (reprocess all files) |
| `/sync` | Re-embed changed wiki pages (after manual edits) |
| `/save <title>` | Save last answer as `wiki/synthesis/<slug>.md` |
| `/fixwiki <name> <fix>` | Correct a wiki page by name or path |
| `/gap <topic>` | Flag a missing topic — creates a research stub |
| `/amendments` | List all amendment audit records |
| `/lint` | Wiki health check — broken links, orphans, stubs, contradictions |
| `/status` | Show project dir, raw file count, wiki pages, KB chunks, API key status |
| `/config` | Show global config (dir, model, embeddings, paths) |
| `/config key` | Show masked API key |
| `/config reset` | Remove global config — next launch triggers setup |
| `/logout` | Same as `/config reset` — removes `~/.config/secondbrain/` |
| `/tips` | Show quick-start guide |
| `/help` | Show all commands |
| `!<command>` | Run a shell command from the project directory |

---

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate command history |
| `PgUp` / `PgDn` | Scroll chat (stops auto-follow when scrolled up) |
| `Ctrl+Y` | Copy last answer to clipboard |
| `Ctrl+C` | Quit |
| `confirm` | (in confirmation prompts) Apply a wiki correction |
| `force` | (in confirmation prompts) Force-apply despite contradictions |

---

## Releases

Install the latest release:

```bash
go install github.com/ORG028658/TheSecondBrain/tui@latest
```

Install a specific version:

```bash
go install github.com/ORG028658/TheSecondBrain/tui@v0.1.0
```

Tagged releases also publish prebuilt archives for macOS and Linux. See [RELEASE.md](RELEASE.md) for the release process and backup guidance.

---

## Configuration

**Global** (`~/.config/secondbrain/config.yaml`):
```yaml
llm:
  model: "gpt-4o"
  max_tokens: 4096
  base_url: "https://api.ai.public.rakuten-it.com/openai/v1"

embeddings:
  model: "text-embedding-3-small"
  base_url: "https://api.ai.public.rakuten-it.com/openai/v1"

rag:
  chunk_size: 1500      # characters per chunk
  top_k: 5              # max chunks retrieved per query
  min_similarity: 0.25  # discard chunks below this cosine score (raise for stricter relevance)
```

**Secrets** (`~/.config/secondbrain/.env`):
```
LLM_COMPATIBLE_API_KEY=your_api_key_here
```

### Compatible API Providers

TheSecondBrain works with **any OpenAI-compatible API endpoint**. Change `base_url` and `model` in `config.yaml`, then set `LLM_COMPATIBLE_API_KEY` to that provider's key:

| Provider | `base_url` | Example model |
|----------|-----------|---------------|
| [OpenAI](https://platform.openai.com) | `https://api.openai.com/v1` | `gpt-4o` |
| [Groq](https://groq.com) | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile` |
| [Azure OpenAI](https://azure.microsoft.com/products/ai-services/openai-service) | `https://<resource>.openai.azure.com/openai` | your deployment name |
| [Ollama](https://ollama.com) (local, free) | `http://localhost:11434/v1` | `llama3.2` |
| [Together AI](https://together.ai) | `https://api.together.xyz/v1` | `meta-llama/Llama-3-70b` |
| Rakuten AI Gateway | `https://api.ai.public.rakuten-it.com/openai/v1` | `gpt-4o` |

> **Anthropic / Claude:** Anthropic's native API is not OpenAI-compatible. Access Claude models through [AWS Bedrock](https://aws.amazon.com/bedrock/) or [Google Vertex AI](https://cloud.google.com/vertex-ai), both of which expose OpenAI-compatible endpoints.

---

## Directory Layout

```
<your-project>/
├── raw/                          ← Drop any files here (immutable)
├── wiki/
│   ├── sources/                  ← One summary per raw source
│   ├── entities/                 ← People, orgs, products, tools
│   ├── concepts/                 ← Ideas, patterns, theories
│   ├── synthesis/                ← Saved query results
│   ├── index.md                  ← Auto-maintained catalog
│   └── log.md                    ← Operation history
└── knowledge-base/
    ├── embeddings/store.json     ← Vector store (flat JSON, cosine search)
    ├── metadata/sources.json     ← Content hashes for change detection
    ├── amendments/               ← Correction audit trail
    └── output/                   ← Reports and exports

~/.config/secondbrain/
├── config.yaml                   ← Global model + RAG settings
└── .env                          ← API key (600 permissions, never committed)
```

---

## Troubleshooting

- `brain: command not found`
  Add `$(go env GOPATH)/bin` or `$GOBIN` to your `PATH`.
- `401` or unauthorized errors
  Check `~/.config/secondbrain/.env` and confirm `LLM_COMPATIBLE_API_KEY` is valid for the configured provider.
- Warning about nested `wiki/wiki/...`
  This usually means the vault was touched by an older path bug. New writes are normalized, but you should inspect and migrate nested markdown files before trusting mixed results.
- PDF files do nothing
  PDF ingestion is not implemented yet in this beta.

## Backup and Recovery

Back up these paths regularly:

- your project `wiki/`
- your project `knowledge-base/`
- your project `raw/` if the original source material is not stored elsewhere
- `~/.config/secondbrain/` if you want to preserve config and API settings

Treat `wiki/` and `knowledge-base/amendments/` as source-of-truth user data. `knowledge-base/embeddings/store.json` can usually be rebuilt with `/pull` or `/sync`.

---

## How It Works — The Pipeline

```
Drop file into raw/
        │
        ▼  (auto after 3s, or /pull)
[Analyzer — LLM]
  · Reads file content (text, code, or vision for images)
  · Checks wiki/index.md for existing related pages
  · Returns JSON: array of pages to create/update + log entry
        │
        ▼
wiki/ pages written
  wiki/sources/slug.md        ← source summary
  wiki/entities/name.md       ← per entity mentioned
  wiki/concepts/name.md       ← per concept covered
  (typically 5–15 pages per source)
        │
        ▼  (/sync or auto after /pull)
[Embeddings — Rakuten AI Gateway]
  · Chunks each wiki page by paragraph
  · Embeds via text-embedding-3-small
  · Stores vectors in knowledge-base/embeddings/store.json
        │
        ▼  (on question)
[RAG Query]
  · Embeds question
  · Cosine search → top-K chunks filtered by min_similarity
  · LLM answers from wiki context only
  · LLM scores each reference by actual contribution
  · Conversation history passed for follow-up awareness
```

---

## The LLM Schema

`CLAUDE.md` at the project root is the brain's operating manual. It defines:
- **10 canonical rules** — never touch `raw/`, always use wikilinks, always update the index, etc.
- **Page templates** — for sources, entities, concepts, and synthesis pages
- **Four operations** — Ingest, Query, Lint, and Onboarding with step-by-step workflows
- **Knowledge extraction standard** — the LLM extracts *how things work*, not just *what things are*

This file is automatically loaded into every Claude Code session that opens the project directory.

---

## Dependencies

| Component | Library |
|-----------|---------|
| TUI | `charmbracelet/bubbletea` + `bubbles` + `lipgloss` |
| LLM (wiki + RAG answers) | OpenAI-compatible API (Rakuten AI Gateway → `gpt-4o`) |
| Embeddings | Rakuten AI Gateway → `text-embedding-3-small` |
| Vector store | Flat JSON with cosine similarity (zero dependencies) |
| File watching | `fsnotify/fsnotify` |
| Config | `gopkg.in/yaml.v3` + `joho/godotenv` |
| Clipboard | `atotto/clipboard` |

---

## Project Docs

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [ROADMAP.md](ROADMAP.md)
- [RELEASE.md](RELEASE.md)
- [PRIVACY.md](PRIVACY.md)
