# Roadmap: TheSecondBrain

**Created:** 2026-04-16
**Milestone:** M1 — Feature Expansion
**Phases:** 3 | **Requirements mapped:** 18/18 ✓

Each phase is independently executable once the core Go internal packages exist.
They share no runtime dependencies on each other — any ordering works.

---

## Phase 1: Persona & Skill Profiles

**Goal:** Let users define a `profile.yaml` that shapes extraction quality and answer style,
making the brain domain-aware and expert-level by default.

**Requirements:** PROFILE-01, PROFILE-02, PROFILE-03, PROFILE-04, PROFILE-05, PROFILE-06

**Depends on:** Core internal packages (`internal/config`, `internal/analyzer`, `internal/rag`)

**Success Criteria:**
1. A `profile.yaml` placed in the project root changes what entities and concepts the brain
   extracts from a raw file (verifiable by comparing ingest output with/without profile)
2. `/status` command output includes the active profile name (e.g., `Profile: Sage`)
3. A vault with no `profile.yaml` behaves identically to the pre-profile baseline —
   no errors, no changed behavior, no missing output

**Key files to create/modify:**
- `profile.yaml` (new — user-created, documented in README)
- `tui/internal/config/profile.go` (new — profile loader)
- `tui/internal/analyzer/analyzer.go` (modify — inject profile into ingest prompt)
- `tui/internal/rag/rag.go` (modify — inject profile into query prompt)
- `tui/internal/ui/status.go` (modify — show profile name in header)

---

## Phase 2: MCP Server Mode

**Goal:** Expose the wiki as MCP tools over stdio so any LLM client (Claude Code, Cursor,
Cline) can search, read, and write to the brain without a human at the terminal.

**Requirements:** MCP-01, MCP-02, MCP-03, MCP-04, MCP-05, MCP-06, MCP-07

**Depends on:** Core internal packages (independent of Phase 1 and Phase 3)

**Success Criteria:**
1. `brain --mcp` starts without error and prints `MCP server ready` to stderr,
   then waits for JSON-RPC 2.0 messages on stdin
2. A Claude Code session with `brain --mcp` configured as an MCP server can call
   `search_brain("query")` and receive ranked wiki chunks with page paths and scores
3. `add_to_raw("test.md", "# Test\nContent here")` drops a file in `raw/` and
   the file watcher picks it up within the 3-second debounce window

**Key files to create/modify:**
- `tui/cmd/mcp/main.go` (new — MCP server entry point)
- `tui/internal/mcp/server.go` (new — JSON-RPC 2.0 stdio transport)
- `tui/internal/mcp/tools.go` (new — search_brain, read_page, add_to_raw, list_pages)
- `tui/main.go` (modify — route `--mcp` flag to MCP server instead of TUI)
- `README.md` (modify — add MCP setup section with Claude Code config example)

**MCP tool signatures:**
```
search_brain(query: string) → [{wiki_path, chunk_text, score}]
read_page(wiki_path: string) → {content: string}
add_to_raw(filename: string, content: string) → {ok: bool, path: string}
list_pages(type?: string) → [{wiki_path, title, type, tags}]
```

---

## Phase 3: Knowledge Graph & /explore

**Goal:** Surface the implicit wikilink graph already encoded in `[[WikiLink]]` syntax as
navigable structure (`/explore`), exportable Mermaid diagram (`/graph`), and health checks
(`/lint` orphan/broken-link detection).

**Requirements:** GRAPH-01, GRAPH-02, GRAPH-03, GRAPH-04, GRAPH-05

**Depends on:** Core internal packages (independent of Phase 1 and Phase 2)

**Success Criteria:**
1. `/explore golang` returns a traversal narrative showing the path through connected wiki
   pages, with each hop justified by the LLM (e.g., "Go → Bubble Tea → TUI patterns")
2. `/graph` writes a valid Mermaid `graph TD` diagram to
   `knowledge-base/output/graph.md` with one node per wiki page and one edge per wikilink
3. `/lint` correctly identifies at least the known orphan pages (pages with no inbound
   links) and at least one broken wikilink from the seed wiki

**Key files to create/modify:**
- `tui/internal/graph/graph.go` (new — parse `[[WikiLink]]` syntax, build adjacency list)
- `tui/internal/graph/explore.go` (new — LLM-guided 2-hop traversal)
- `tui/internal/graph/export.go` (new — Mermaid diagram export)
- `tui/internal/graph/lint.go` (new — orphan and broken-link detection)
- `tui/internal/ui/commands.go` (modify — wire `/explore`, `/graph`, `/lint` commands)

**Graph algorithm:**
- Build adjacency list by parsing `[[PageTitle]]` from all `.md` files under `wiki/`
- Resolve links by fuzzy-matching title against existing page filenames
- `/explore`: BFS from seed page, LLM scores each candidate neighbor, pick top-3 per hop
- `/graph`: Walk all edges, emit `PageA --> PageB` in Mermaid format
- `/lint`: Nodes with in-degree 0 = orphans; link targets not in page set = broken

---

## Requirement Coverage

| Phase | Requirements | Count |
|-------|-------------|-------|
| Phase 1 | PROFILE-01 through PROFILE-06 | 6 |
| Phase 2 | MCP-01 through MCP-07 | 7 |
| Phase 3 | GRAPH-01 through GRAPH-05 | 5 |
| **Total** | | **18** |

All 18 v1 requirements mapped. Coverage: 100% ✓

---
*Roadmap created: 2026-04-16*
