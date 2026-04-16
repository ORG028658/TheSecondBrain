---
description: Run a wiki health check — detect broken links, orphan pages, stubs, missing pages, and contradictions
---

Perform a LINT health check on this wiki vault.

Scan all pages in `wiki/sources/`, `wiki/entities/`, `wiki/concepts/`, and `wiki/synthesis/`.

Check for each of the following:

| Issue | How to detect |
|-------|--------------|
| **Broken wikilinks** | `[[PageName]]` syntax where no file with that title exists in the wiki |
| **Orphan pages** | Pages that no other wiki page links to |
| **Stubs** | Pages with fewer than 3 meaningful content sections filled in |
| **Missing pages** | A concept or entity mentioned in 2+ pages but with no dedicated page of its own |
| **Contradictions** | Conflicting factual claims across pages |
| **Index gaps** | Pages that exist on disk but are missing from `wiki/index.md` |

Return a report structured as:

```markdown
## Critical
[Issues that break navigation or hide contradictions — must fix]

## Minor
[Quality issues — stubs, orphan pages, index gaps]

## Suggestions
[Topics worth creating, sources worth finding, connections worth making]
```

Then append to `wiki/log.md`:
```
## [today's date] lint | Health check
[Brief summary: X critical, Y minor, Z suggestions]
```
