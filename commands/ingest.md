---
description: Ingest a source file from raw/ into the wiki — creates a source page and all related entity and concept pages
---

Perform a full INGEST operation on the file `raw/$ARGUMENTS`.

Follow the INGEST protocol from TheSecondBrain CLAUDE.md exactly:

1. **Read** the file at `raw/$ARGUMENTS` in full
2. **Check** `wiki/index.md` for existing related pages to update rather than duplicate
3. **Create** `wiki/sources/[slug].md` — the source summary page
4. **Create or update** entity pages in `wiki/entities/` for every significant person, org, product, tool, or place
5. **Create or update** concept pages in `wiki/concepts/` for every significant idea, pattern, theory, or technique
6. **Add `[[WikiLinks]]`** — every page you write must link to all related pages
7. **Update** `wiki/index.md` — add all new pages to the catalog
8. **Append** to `wiki/log.md`:
   ```
   ## [today's date] ingest | [source title]
   Created: sources/[slug].md. Updated: [list all touched pages].
   ```

If `$ARGUMENTS` is empty, list the files in `raw/` and ask which one to ingest.

**Extraction rules:**
- A single source typically touches 5–15 wiki pages
- Extract *how things work*, not just *what they are*
- Never copy-paste from the source — synthesise in your own words
- The non-obvious insight is the most valuable thing to capture
