#!/usr/bin/env bash
# update-docs.sh — sync all docs, scripts, and in-app strings with current TUI changes
#
# Changes reflected:
#   1.  create-wiki.yml: `go build -o brain .` → `go build -o dist/brain ./cmd/brain`
#   2.  sync.sh: /analyze → /pull + --current-dir note
#   3.  CONTRIBUTING.md / RELEASE.md: `go build ./...` → `go build ./cmd/brain`
#   4.  README.md keyboard table: add Esc, 1/2/3 pane-switch rows
#   5.  README.md diagram: /analyze label → /pull (ingest)
#   6.  README.md Commands table: add /analyze --current-dir row
#   7.  README.md TUI features: add Sidebar bullet
#   8.  README.md scroll description: reflect footer hint
#   9.  app.go tipsMessage: add Esc + pane shortcuts
#   10. app.go helpText: add Esc + pane shortcuts
#   11. rebuild binary: go build -o dist/brain ./cmd/brain
#
# Safe to re-run — every check uses a unique marker string.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"

info()    { printf '  \033[34m◆\033[0m  %s\n' "$*"; }
success() { printf '  \033[32m✓\033[0m  %s\n' "$*"; }
skip()    { printf '  \033[33m–\033[0m  %s\n' "$*"; }

# Portable in-place sed (macOS needs an empty-string backup arg).
sedi() {
  if sed --version 2>/dev/null | grep -q GNU; then
    sed -i "$@"
  else
    sed -i '' "$@"
  fi
}

echo ""
echo "  ◆  TheSecondBrain — update-docs"
echo ""

# ── 1. create-wiki.yml ────────────────────────────────────────────────────────
FILE="$REPO_ROOT/.github/workflows/create-wiki.yml"
if grep -q 'go build -o brain \.' "$FILE"; then
  sedi 's|go build -o brain \.|go build -o dist/brain ./cmd/brain|g' "$FILE"
  sedi 's|sudo mv brain /usr/local/bin/brain|sudo mv dist/brain /usr/local/bin/brain|g' "$FILE"
  success "create-wiki.yml — fixed build command"
else
  skip "create-wiki.yml — already up to date"
fi

# ── 2. sync.sh ────────────────────────────────────────────────────────────────
FILE="$REPO_ROOT/sync.sh"
if grep -q 'Run /analyze in the TUI' "$FILE"; then
  sedi 's|Run /analyze in the TUI to update the wiki from changed files\.|Run /pull in the TUI to update the wiki from changed files.\n      (or /pull --current-dir if you launched brain from the project root)|' "$FILE"
  success "sync.sh — updated end message"
else
  skip "sync.sh — already up to date"
fi

# ── 3. CONTRIBUTING.md — build commands ───────────────────────────────────────
FILE="$REPO_ROOT/CONTRIBUTING.md"
if grep -q '^go build \./\.\.\.' "$FILE"; then
  sedi 's|^go build \./\.\.\.$|go build ./cmd/brain|g' "$FILE"
  success "CONTRIBUTING.md — fixed go build command"
else
  skip "CONTRIBUTING.md — already up to date"
fi

# ── 4. RELEASE.md — build command ─────────────────────────────────────────────
FILE="$REPO_ROOT/RELEASE.md"
if grep -q '^go build \./\.\.\.' "$FILE"; then
  sedi 's|^go build \./\.\.\.$|go build ./cmd/brain|g' "$FILE"
  success "RELEASE.md — fixed go build command"
else
  skip "RELEASE.md — already up to date"
fi

# ── 5. README.md — architecture diagram label ─────────────────────────────────
FILE="$REPO_ROOT/README.md"
if grep -q '/analyze.*ingest' "$FILE"; then
  sedi 's|/analyze.*\(ingest\)|/pull   (ingest)|' "$FILE"
  success "README.md — fixed architecture diagram label"
else
  skip "README.md diagram — already up to date"
fi

# ── 6. README.md — keyboard shortcuts table ───────────────────────────────────
# Add sidebar pane keys, Esc cancel, and scroll hint if not already present.
if ! grep -q 'Switch sidebar pane' "$FILE"; then
  python3 - "$FILE" <<'PYEOF'
import sys, re
path = sys.argv[1]
text = open(path).read()
old = '| `Ctrl+C` | Quit |'
new = ('| `Ctrl+C` | Quit |\n'
       '| `Esc` | Cancel current operation (query, pull, analyze) or wiki confirmation |\n'
       '| `1` / `2` / `3` | Switch sidebar pane: Chat / Commands / Status (when input is empty) |')
text = text.replace(old, new, 1)
open(path, 'w').write(text)
PYEOF
  success "README.md — added Esc and pane-switch shortcuts"
else
  skip "README.md keyboard table — already up to date"
fi

# ── 7. README.md — TUI features section ──────────────────────────────────────
if ! grep -q 'Sidebar' "$FILE"; then
  python3 - "$FILE" <<'PYEOF'
import sys
path = sys.argv[1]
text = open(path).read()
old = '- **Brain logo** with live stats in the header'
new = ('- **Sidebar** — press `1`/`2`/`3` to switch between Chat, Commands, and Status panes\n'
       '- **Brain logo** with live stats in the header')
text = text.replace(old, new, 1)
open(path, 'w').write(text)
PYEOF
  success "README.md — added sidebar feature note"
else
  skip "README.md TUI features — already up to date"
fi

if ! grep -q 'scroll hint' "$FILE" && ! grep -q 'scrolled.*PgDn' "$FILE"; then
  python3 - "$FILE" <<'PYEOF'
import sys, re
path = sys.argv[1]
text = open(path).read()
old = '- **Scroll** — `PgUp`/`PgDn` to scroll chat history; auto-follows new messages unless you\'ve scrolled up'
new = '- **Scroll** — `PgUp`/`PgDn` to scroll chat history; a scroll hint appears in the footer when not at the bottom (auto-follows otherwise)'
text = text.replace(old, new, 1)
open(path, 'w').write(text)
PYEOF
  success "README.md — updated scroll description"
else
  skip "README.md scroll description — already up to date"
fi

# ── 8. README.md — Commands table: /analyze --current-dir row ─────────────────
if ! grep -q '/analyze --current-dir' "$FILE"; then
  python3 - "$FILE" <<'PYEOF'
import sys
path = sys.argv[1]
text = open(path).read()
old = '| `/analyze` | Force re-analyze `raw/` (reprocess all files) |'
new = ('| `/analyze` | Force re-analyze `raw/` (reprocess all files) |\n'
       '| `/analyze --current-dir` | Same as `/analyze` but uses the project root instead of `raw/` |')
text = text.replace(old, new, 1)
open(path, 'w').write(text)
PYEOF
  success "README.md — added /analyze --current-dir command row"
else
  skip "README.md /analyze --current-dir — already present"
fi

# ── 9. app.go — tipsMessage keyboard shortcuts ────────────────────────────────
FILE="$REPO_ROOT/tui/internal/ui/app.go"
if ! grep -q '1 / 2 / 3.*switch pane\|switch pane.*1 / 2 / 3' "$FILE"; then
  python3 - "$FILE" <<'PYEOF'
import sys
path = sys.argv[1]
text = open(path).read()
old = ('  Shortcuts:\n'
       '    ↑ ↓       navigate command history\n'
       '    PgUp / PgDn  scroll chat\n'
       '    Ctrl+Y    copy last answer to clipboard\n'
       '    Ctrl+C    quit')
new = ('  Shortcuts:\n'
       '    ↑ ↓          navigate command history\n'
       '    PgUp / PgDn  scroll chat (footer shows hint when scrolled up)\n'
       '    Ctrl+Y       copy last answer to clipboard\n'
       '    Esc          cancel current operation\n'
       '    1 / 2 / 3    switch pane: Chat / Commands / Status\n'
       '    Ctrl+C       quit')
if old in text:
    text = text.replace(old, new, 1)
    open(path, 'w').write(text)
    print('changed')
PYEOF
  success "app.go tipsMessage — added Esc and pane shortcuts"
else
  skip "app.go tipsMessage — already up to date"
fi

# ── 10. app.go — helpText keyboard shortcuts ──────────────────────────────────
if ! grep -q 'Switch pane.*Chat.*Commands.*Status' "$FILE"; then
  python3 - "$FILE" <<'PYEOF'
import sys
path = sys.argv[1]
text = open(path).read()
old = ('Keyboard:\n'
       '  ↑ / ↓        Navigate command history\n'
       '  PgUp / PgDn  Scroll chat\n'
       '  Ctrl+Y       Copy last answer to clipboard\n'
       '  Ctrl+C       Quit`')
new = ('Keyboard:\n'
       '  ↑ / ↓        Navigate command history\n'
       '  PgUp / PgDn  Scroll chat (footer shows hint when scrolled up)\n'
       '  Ctrl+Y       Copy last answer to clipboard\n'
       '  Esc          Cancel current operation\n'
       '  1 / 2 / 3    Switch pane: Chat / Commands / Status (when input is empty)\n'
       '  Ctrl+C       Quit`')
if old in text:
    text = text.replace(old, new, 1)
    open(path, 'w').write(text)
    print('changed')
PYEOF
  success "app.go helpText — added Esc and pane shortcuts"
else
  skip "app.go helpText — already up to date"
fi

# ── 11. rebuild binary ────────────────────────────────────────────────────────
info "Rebuilding binary..."
if cd "$REPO_ROOT/tui" && go build -o dist/brain ./cmd/brain 2>&1; then
  success "dist/brain rebuilt successfully"
else
  printf '  \033[31m✗\033[0m  Build failed — check go build output above\n' >&2
  exit 1
fi

echo ""
success "All done."
echo ""
