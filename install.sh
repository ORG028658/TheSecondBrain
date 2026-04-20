#!/usr/bin/env bash
# TheSecondBrain — installer
#
# Usage (no Git or Go needed):
#   curl -fsSL https://raw.githubusercontent.com/ORG028658/TheSecondBrain/main/install.sh | bash
#
# What it does:
#   1. Detects your OS and chip (macOS Intel / Apple Silicon, Linux x86 / ARM)
#   2. Downloads the matching pre-built binary from the latest GitHub release
#   3. Installs it to ~/.local/bin/brain  (no sudo required)
#   4. Adds ~/.local/bin to your PATH if it isn't there already

set -euo pipefail

REPO="ORG028658/TheSecondBrain"
BINARY_NAME="brain"
INSTALL_DIR="$HOME/.local/bin"

# ── Pretty output ──────────────────────────────────────────────────────────────
info()    { printf '  \033[34m◆\033[0m  %s\n' "$*"; }
success() { printf '  \033[32m✓\033[0m  %s\n' "$*"; }
warn()    { printf '  \033[33m⚠\033[0m  %s\n' "$*"; }
die()     { printf '  \033[31m✗\033[0m  %s\n' "$*" >&2; exit 1; }

echo ""
echo "  ◆  TheSecondBrain — installer"
echo ""

# ── Detect OS and architecture ─────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin)
    GOOS="darwin"
    ;;
  Linux)
    GOOS="linux"
    ;;
  *)
    die "Unsupported OS: $OS. Only macOS and Linux are supported."
    ;;
esac

case "$ARCH" in
  x86_64)
    GOARCH="amd64"
    ;;
  arm64 | aarch64)
    GOARCH="arm64"
    ;;
  *)
    die "Unsupported architecture: $ARCH."
    ;;
esac

info "Detected: $OS ($ARCH)"

# ── Check for download tool ────────────────────────────────────────────────────
if command -v curl >/dev/null 2>&1; then
  DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
  DOWNLOADER="wget"
else
  die "Neither curl nor wget found. Please install one and re-run."
fi

# ── Fetch latest release tag ───────────────────────────────────────────────────
info "Fetching latest release..."

RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"

if [ "$DOWNLOADER" = "curl" ]; then
  LATEST=$(curl -fsSL "$RELEASE_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
else
  LATEST=$(wget -qO- "$RELEASE_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
fi

if [ -z "$LATEST" ]; then
  die "Could not fetch latest release from GitHub. Check your internet connection."
fi

info "Latest release: $LATEST"

# ── Build download URL ─────────────────────────────────────────────────────────
ARCHIVE="${BINARY_NAME}-${GOOS}-${GOARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"

# ── Download and extract ───────────────────────────────────────────────────────
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading $ARCHIVE..."

if [ "$DOWNLOADER" = "curl" ]; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE" || \
    die "Download failed. Release $LATEST may not have a build for $GOOS/$GOARCH yet."
else
  wget -qO "$TMP_DIR/$ARCHIVE" "$DOWNLOAD_URL" || \
    die "Download failed. Release $LATEST may not have a build for $GOOS/$GOARCH yet."
fi

tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"

# ── Install binary ─────────────────────────────────────────────────────────────
mkdir -p "$INSTALL_DIR"
EXTRACTED_BINARY="$TMP_DIR/${BINARY_NAME}-${GOOS}-${GOARCH}"

if [ ! -f "$EXTRACTED_BINARY" ]; then
  die "Could not find binary in archive. Expected: ${BINARY_NAME}-${GOOS}-${GOARCH}"
fi

chmod +x "$EXTRACTED_BINARY"
mv "$EXTRACTED_BINARY" "$INSTALL_DIR/$BINARY_NAME"
success "Installed to $INSTALL_DIR/$BINARY_NAME"

# ── Add ~/.local/bin to PATH if needed ────────────────────────────────────────
add_to_path() {
  local rc_file="$1"
  local dir="$2"
  local export_line="export PATH=\"${dir}:\$PATH\""

  if [ -f "$rc_file" ] && grep -qF "$dir" "$rc_file" 2>/dev/null; then
    return 0  # already present
  fi

  printf '\n# Added by TheSecondBrain installer\n%s\n' "$export_line" >> "$rc_file"
  success "Added $dir to PATH in $rc_file"
  return 1  # signal that we added it (needs reload)
}

NEEDS_RELOAD=false
SHELL_NAME="$(basename "${SHELL:-/bin/sh}")"

case "$SHELL_NAME" in
  zsh)
    add_to_path "$HOME/.zshrc" "$INSTALL_DIR" || NEEDS_RELOAD=true
    ;;
  bash)
    if [ "$(uname -s)" = "Darwin" ]; then
      add_to_path "$HOME/.bash_profile" "$INSTALL_DIR" || NEEDS_RELOAD=true
    else
      add_to_path "$HOME/.bashrc" "$INSTALL_DIR" || NEEDS_RELOAD=true
    fi
    ;;
  *)
    warn "Unknown shell ($SHELL_NAME). Add this to your shell config manually:"
    echo '    export PATH="$HOME/.local/bin:$PATH"'
    ;;
esac

# ── Done ───────────────────────────────────────────────────────────────────────
echo ""
success "TheSecondBrain $LATEST installed successfully."
echo ""

if [ "$NEEDS_RELOAD" = true ]; then
  echo "  Run this to start using it right now:"
  echo ""
  case "$SHELL_NAME" in
    zsh)  echo '    source ~/.zshrc && brain' ;;
    bash) echo '    source ~/.bash_profile && brain' ;;
    *)    echo '    Restart your terminal, then run: brain' ;;
  esac
else
  echo "  Run:  brain"
fi

echo ""
