#!/usr/bin/env bash
# TheSecondBrain — install script
# Builds the 'brain' binary and installs it to /usr/local/bin

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TUI_DIR="$REPO_DIR/tui"
BINARY_NAME="brain"
INSTALL_DIR="/usr/local/bin"

echo ""
echo "  ◆  TheSecondBrain — installer"
echo ""

# Check Go
if ! command -v go &> /dev/null; then
  echo "  ✗ Go is not installed."
  echo "    Install it from: https://go.dev/dl/"
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "  ✓ Go $GO_VERSION found"

# Download dependencies
echo "  ↓ Downloading dependencies..."
cd "$TUI_DIR"
go mod tidy -e 2>/dev/null || go mod download

# Build
echo "  ⚙ Building..."
go build -ldflags="-s -w" -o "$BINARY_NAME" .
echo "  ✓ Built: $TUI_DIR/$BINARY_NAME"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
  echo "  ✓ Installed to $INSTALL_DIR/$BINARY_NAME"
else
  echo ""
  echo "  Installing to $INSTALL_DIR requires sudo:"
  sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
  echo "  ✓ Installed to $INSTALL_DIR/$BINARY_NAME"
fi

echo ""
echo "  ✓ Done. Run 'brain' from anywhere to start."
echo ""
