# Requirements: TheSecondBrain

**Defined:** 2026-04-16
**Core Value:** Knowledge compounds — ingested sources create persistent, cross-referenced wiki pages

## v1 Requirements

### Persona & Skill Profiles

- [ ] **PROFILE-01**: User can create a `profile.yaml` in the project root with: name, role,
  focus (list of domains), tone (precise|conversational|terse), expertise_level, and ignore list
- [ ] **PROFILE-02**: Brain injects profile context into the ANALYZE (ingest) LLM prompt,
  focusing entity/concept extraction on the declared domain focus areas
- [ ] **PROFILE-03**: Brain injects profile context into the QUERY LLM prompt, shaping
  answer depth and terminology to match the declared expertise level and tone
- [ ] **PROFILE-04**: Brain skips or de-emphasizes content types matching the profile's
  `ignore` list during ingest (e.g., ignore: [marketing copy, boilerplate comments])
- [ ] **PROFILE-05**: `/status` command and TUI header show the active profile name
- [ ] **PROFILE-06**: System works identically with no profile.yaml present (graceful default)

### MCP Server Mode

- [ ] **MCP-01**: `brain --mcp` flag starts an MCP server instead of the TUI
- [ ] **MCP-02**: MCP server exposes `search_brain(query: string)` tool that returns
  ranked wiki chunks with cosine scores and page paths
- [ ] **MCP-03**: MCP server exposes `read_page(wiki_path: string)` tool returning
  full page markdown content
- [ ] **MCP-04**: MCP server exposes `add_to_raw(filename: string, content: string)` tool
  that writes a file to raw/ and triggers ingestion
- [ ] **MCP-05**: MCP server exposes `list_pages(type?: string)` tool returning index entries
  optionally filtered by type (source|entity|concept|synthesis)
- [ ] **MCP-06**: Server uses stdio transport (JSON-RPC 2.0 over stdin/stdout) for
  compatibility with Claude Code, Cursor, Cline, and other MCP clients
- [ ] **MCP-07**: Server resolves the project root from CWD at startup (same as TUI mode)

### Knowledge Graph

- [ ] **GRAPH-01**: `/explore <topic>` command traverses the wikilink graph from a starting
  page, with LLM selecting the most relevant connections up to 2 hops
- [ ] **GRAPH-02**: `/graph` command exports the full wikilink adjacency list as a Mermaid
  diagram written to `knowledge-base/output/graph.md`
- [ ] **GRAPH-03**: `/lint` command uses the graph to detect and report orphan pages
  (pages with no inbound wikilinks from other wiki pages)
- [ ] **GRAPH-04**: `/lint` command uses the graph to detect broken wikilinks (links
  pointing to pages that do not exist in wiki/)
- [ ] **GRAPH-05**: Graph is built by parsing `[[WikiLink]]` syntax from all wiki pages
  at command time (no persistent index required)

## v2 Requirements

### Persona

- **PROFILE-07**: Multiple named profiles with `/profile use <name>` switching
- **PROFILE-08**: Profile-specific ignore thresholds and extraction depth tuning

### MCP

- **MCP-08**: `get_page_graph(page: string)` tool returning wikilink neighbors (after GRAPH ships)
- **MCP-09**: SSE transport option for web-based MCP clients

### Knowledge Graph

- **GRAPH-06**: Interactive graph navigation in TUI with arrow-key traversal
- **GRAPH-07**: `/graph --filter <tag>` to export subgraphs by tag

## Out of Scope

| Feature | Reason |
|---------|--------|
| PDF text extraction | Manual extraction required; native PDF parsing is v2 |
| Static HTML export (`/share`) | Useful but not core to any of the 3 features |
| Plugin system | Backlog item; not required for these features |
| Multi-vault support | CWD-as-project is sufficient; multi-vault is future |
| Real-time collaboration | Single-user by design |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PROFILE-01 | Phase 1 | Pending |
| PROFILE-02 | Phase 1 | Pending |
| PROFILE-03 | Phase 1 | Pending |
| PROFILE-04 | Phase 1 | Pending |
| PROFILE-05 | Phase 1 | Pending |
| PROFILE-06 | Phase 1 | Pending |
| MCP-01 | Phase 2 | Pending |
| MCP-02 | Phase 2 | Pending |
| MCP-03 | Phase 2 | Pending |
| MCP-04 | Phase 2 | Pending |
| MCP-05 | Phase 2 | Pending |
| MCP-06 | Phase 2 | Pending |
| MCP-07 | Phase 2 | Pending |
| GRAPH-01 | Phase 3 | Pending |
| GRAPH-02 | Phase 3 | Pending |
| GRAPH-03 | Phase 3 | Pending |
| GRAPH-04 | Phase 3 | Pending |
| GRAPH-05 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 18 total
- Mapped to phases: 18
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-16*
*Last updated: 2026-04-16 after initial definition*
