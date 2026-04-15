---
description: Show vault stats — page counts by type, recent activity from the log, and quick structural health indicators
---

Report the current status of this knowledge vault.

1. **Count files** in each directory:
   - `raw/` — source files (ingested and pending)
   - `wiki/sources/` — processed source summaries
   - `wiki/entities/` — entity pages
   - `wiki/concepts/` — concept pages
   - `wiki/synthesis/` — saved analyses

2. **Recent activity** — read `wiki/log.md` and show the last 5 log entries

3. **Index check** — read `wiki/index.md`, count listed pages, and flag any discrepancy between the listed count and actual file count on disk

4. **Quick health** — check for obvious structural issues:
   - Is `wiki/index.md` present and non-empty?
   - Is `wiki/log.md` present and has entries?
   - Are there files in `raw/` not yet reflected in `wiki/sources/`? (potential un-ingested sources)

Format as a clean status summary. End with: *"Run `/lint` for a full health check."*
