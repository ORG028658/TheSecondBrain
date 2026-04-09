#!/usr/bin/env bash
# sync.sh — standalone git sync helper
# Discovers and pulls all git repos under raw/
# This is the shell equivalent of the TUI /pull command.

set -euo pipefail

RAW_DIR="$(dirname "$0")/raw"

if [ ! -d "$RAW_DIR" ]; then
  echo "raw/ directory not found at $RAW_DIR"
  exit 1
fi

echo "Scanning $RAW_DIR for git repositories..."

found=0
while IFS= read -r git_dir; do
  repo_dir="$(dirname "$git_dir")"
  rel="$(realpath --relative-to="$RAW_DIR" "$repo_dir")"
  echo ""
  echo "→ $rel"
  if git -C "$repo_dir" pull 2>&1; then
    echo "  ✓ pulled"
  else
    echo "  ⚠ pull failed — authentication may be required."
    echo "    Please manually copy updated files into: $repo_dir"
    echo "    Then run /sync in the TUI to re-index."
  fi
  ((found++))
done < <(find "$RAW_DIR" -name ".git" -type d)

if [ "$found" -eq 0 ]; then
  echo "No git repositories found under raw/"
fi

echo ""
echo "Done. Run /analyze in the TUI to update the wiki from changed files."
