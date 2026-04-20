# TheSecondBrain

## What This Is

A terminal-native personal knowledge management system that uses LLMs to automatically
extract, structure, and maintain a persistent, interlinked wiki from raw documents, code,
and notes dropped into a `raw/` folder. Runs as a single Go binary (`brain`).

## Core Value

Knowledge compounds — every raw file ingested creates multiple cross-referenced wiki pages
that grow richer with every source and query, instead of evaporating after each LLM session.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] User can define a persona profile that shapes how the brain extracts and answers
- [ ] LLM clients (Claude Code, Cursor, Cline) can use the brain via MCP tools
- [ ] User can traverse the implicit wikilink graph with `/explore` and `/graph` commands

### Out of Scope

- Web UI — terminal-native by design
- Multi-user / cloud sync — single-user, local files only
- Mobile support — not in scope

## Context

**Tech stack:** Go 1.22+, charmbracelet/bubbletea + bubbles + lipgloss, sashabaranov/go-openai,
fsnotify, gopkg.in/yaml.v3, atotto/clipboard. Single binary, no runtime dependencies.

**Current state:** Architecture is fully designed (CLAUDE.md, PRD.md). Go scaffolding exists
(tui/main.go, go.mod). Internal packages (internal/config, internal/ui, internal/analyzer,
internal/rag, internal/wiki, internal/store) need to be built to spec before features land.

**LLM provider:** Rakuten AI Gateway (OpenAI-compatible). Single API key for both
chat completion (gpt-4o) and embeddings (text-embedding-3-small).

**Wiki structure:** raw/ → wiki/sources/, wiki/entities/, wiki/concepts/, wiki/synthesis/
→ knowledge-base/embeddings/store.json (flat JSON vector store).

## Constraints

- **Tech stack:** Go only — no additional runtimes, no Python, no Node
- **API:** OpenAI-compatible endpoint (Rakuten) — no Anthropic SDK
- **Transport:** MCP server must support stdio (requirement for Claude Code integration)
- **Backwards compatibility:** All features must degrade gracefully if profile.yaml is absent

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Flat JSON vector store | Personal scale (<5K chunks), zero external deps, swappable | — Pending |
| CWD = project path (like git) | No setup friction, supports multiple vaults | — Pending |
| profile.yaml injected into prompts (not a DB) | Simple, human-editable, version-controllable | — Pending |
| MCP via stdio transport | Required by Claude Code; broadest client support | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-16 after initialization*
