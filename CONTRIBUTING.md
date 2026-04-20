# Contributing

Thanks for contributing to TheSecondBrain.

## Project Status

This project is currently a **beta for internal/team use and careful external experimentation**. Correctness, docs accuracy, and regression protection matter more than feature velocity.

## Local Setup

```bash
go install github.com/ORG028658/TheSecondBrain/tui@latest
```

For development from a clone:

```bash
cd tui
go test ./...
go vet ./...
go build ./cmd/brain
```

## Quality Bar

- Keep `wiki/...` as the canonical page identifier everywhere.
- Do not introduce writes under `wiki/wiki/...`.
- Update docs when behavior changes.
- Prefer small, focused changes over broad rewrites.
- Add or update tests for correctness changes, especially around path handling, vault bootstrap, and KB sync behavior.

## Pull Requests

Before opening a PR:

```bash
cd tui
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/brain
```

PRs should include:

- What changed
- Why it changed
- How it was verified
- Any user-visible behavior or doc changes

## Scope Guidance

Good contributions:

- correctness fixes
- tests and CI improvements
- documentation alignment
- install/release improvements
- UX polish that does not weaken invariants

Changes that should start with an issue/discussion first:

- vault layout changes
- new persistence formats
- API/provider behavior changes
- major prompt/schema rewrites
- broad feature additions such as new ingest formats
