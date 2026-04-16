---
description: Save the last answer as a permanent synthesis page in wiki/synthesis/ with proper frontmatter and source citations
---

Save the last answer as a synthesis page titled: **$ARGUMENTS**

1. Create `wiki/synthesis/[slug].md` with this structure:

```yaml
---
type: synthesis
title: "$ARGUMENTS"
tags: [synthesis]
sources: [list the wiki pages cited in the answer]
created: [today's date]
updated: [today's date]
---
```

```markdown
# $ARGUMENTS

## Answer
[The synthesized answer rewritten for permanence — not as a chat reply but as a reference page,
with inline [[WikiLinks]] to every cited page. Write in present tense, declaratively.]

## Evidence
- [[ConceptA]] — [how it supported the answer]
- [[EntityB]] — [how it supported the answer]

## Follow-up Questions
[What this answer raises that is worth investigating next]
```

2. Update `wiki/index.md` — add under Synthesis with a one-line summary
3. Append to `wiki/log.md`:
   ```
   ## [today's date] save | $ARGUMENTS
   Created: synthesis/[slug].md. Sources: [list cited pages].
   ```

If `$ARGUMENTS` is empty, ask the user for a title before proceeding.
