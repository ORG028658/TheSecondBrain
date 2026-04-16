---
description: Flag a missing topic as a research gap — creates a tracked stub so the gap surfaces in future searches and lint checks
---

Create a research gap stub for the topic: **$ARGUMENTS**

1. Create `wiki/sources/gap-[slug].md` with this structure:

```yaml
---
type: source
title: "RESEARCH GAP: $ARGUMENTS"
tags: [gap, research-needed]
sources: []
created: [today's date]
updated: [today's date]
---
```

```markdown
# RESEARCH GAP: $ARGUMENTS

## What's Missing
[What specifically we don't know about this topic yet, and what questions remain unanswered]

## Why It Matters
[How filling this gap would enrich the current wiki and what decisions or understanding it would unlock]

## Suggested Sources
[Types of sources, papers, docs, or people who could fill this gap]

## Related
[[[WikiLinks]] to any pages that reference or are adjacent to this topic]
```

2. Update `wiki/index.md` — add under Sources with a `[GAP]` prefix so gaps are visually distinct
3. Append to `wiki/log.md`:
   ```
   ## [today's date] gap | $ARGUMENTS
   Created: sources/gap-[slug].md. Flagged as research gap.
   ```

If `$ARGUMENTS` is empty, ask the user what topic to flag.
