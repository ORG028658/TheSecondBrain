# Release Process

## Versioning

Use semantic version tags:

- `v0.x.y` while the project remains beta

## Release Checklist

1. Update docs if user-visible behavior changed.
2. Run:

```bash
cd tui
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

3. Confirm README install instructions still work.
4. Create and push a tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

5. GitHub Actions will build release archives for supported platforms and attach them to the release.

## Install Options

Latest:

```bash
go install github.com/ORG028658/TheSecondBrain/tui@latest
```

Versioned:

```bash
go install github.com/ORG028658/TheSecondBrain/tui@v0.1.0
```

## Backup and Recovery

Back up these paths regularly:

- project `wiki/`
- project `knowledge-base/`
- project `raw/` if the original source material is not stored elsewhere
- `~/.config/secondbrain/` if you want to preserve config and API settings

What can usually be regenerated:

- `knowledge-base/embeddings/store.json`
- `knowledge-base/metadata/sources.json`

What should be treated as source-of-truth content:

- `wiki/`
- `raw/`
- `knowledge-base/amendments/`

## Compatibility Notes

- Primary target: macOS and Linux
- Required Go version: the version declared in `tui/go.mod`
- PDFs are not supported yet in the current beta
