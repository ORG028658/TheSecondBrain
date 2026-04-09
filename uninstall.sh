#!/usr/bin/env bash
# TheSecondBrain — uninstall script
# Removes the installed 'brain' binary, app config, and optionally the vault.

set -euo pipefail

BINARY_NAME="brain"
DEFAULT_INSTALL_PATH="/usr/local/bin/$BINARY_NAME"

if [ -n "${XDG_CONFIG_HOME:-}" ]; then
  CONFIG_DIR="${XDG_CONFIG_HOME}/secondbrain"
else
  CONFIG_DIR="${HOME}/.config/secondbrain"
fi

CONFIG_FILE="${CONFIG_DIR}/config.yaml"
ENV_FILE="${CONFIG_DIR}/.env"

REMOVE_VAULT=false
ASSUME_YES=false

usage() {
  cat <<'EOF'
Usage: ./uninstall.sh [options]

Options:
  --all, --remove-vault   Remove the configured vault directory too
  -y, --yes              Do not prompt for confirmation
  -h, --help             Show this help

Default behavior:
  - removes the installed 'brain' binary
  - removes ~/.config/secondbrain (or $XDG_CONFIG_HOME/secondbrain)
  - preserves your vault data unless you explicitly opt in
EOF
}

log() {
  printf '  %s\n' "$1"
}

confirm() {
  local prompt="$1"
  if [ "$ASSUME_YES" = true ]; then
    return 0
  fi

  local reply
  printf "%s [y/N]: " "$prompt"
  read -r reply
  case "$reply" in
    y|Y|yes|YES)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

remove_file() {
  local path="$1"
  if [ ! -e "$path" ]; then
    return 0
  fi

  if rm -f "$path" 2>/dev/null; then
    log "✓ Removed $path"
    return 0
  fi

  log "Removing $path requires sudo"
  sudo rm -f "$path"
  log "✓ Removed $path"
}

remove_dir() {
  local path="$1"
  if [ ! -e "$path" ]; then
    return 0
  fi

  if rm -rf "$path" 2>/dev/null; then
    log "✓ Removed $path"
    return 0
  fi

  log "Removing $path requires sudo"
  sudo rm -rf "$path"
  log "✓ Removed $path"
}

find_binary_path() {
  if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    command -v "$BINARY_NAME"
    return 0
  fi

  if [ -e "$DEFAULT_INSTALL_PATH" ]; then
    printf '%s\n' "$DEFAULT_INSTALL_PATH"
    return 0
  fi

  return 1
}

for arg in "$@"; do
  case "$arg" in
    --all|--remove-vault)
      REMOVE_VAULT=true
      ;;
    -y|--yes)
      ASSUME_YES=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'Unknown option: %s\n\n' "$arg" >&2
      usage >&2
      exit 1
      ;;
  esac
done

log "◆ TheSecondBrain — uninstall"
log ""

BIN_PATH=""
if BIN_PATH="$(find_binary_path)"; then
  log "Found binary: $BIN_PATH"
else
  log "No installed '$BINARY_NAME' binary found in PATH or at $DEFAULT_INSTALL_PATH"
fi

VAULT_PATH=""
if [ -f "$CONFIG_FILE" ]; then
  VAULT_PATH="$(sed -n 's/^vault_path:[[:space:]]*//p' "$CONFIG_FILE" | head -n 1 | sed 's/^"//; s/"$//')"
  if [ -n "$VAULT_PATH" ] && [ "${VAULT_PATH#~/}" != "$VAULT_PATH" ]; then
    VAULT_PATH="${HOME}/${VAULT_PATH#~/}"
  fi
  log "Found config: $CONFIG_FILE"
fi

if [ "$ASSUME_YES" = false ]; then
  echo ""
  if [ -n "$VAULT_PATH" ]; then
    log "Vault path: $VAULT_PATH"
  else
    log "Vault path: not found in config"
  fi
  echo ""
fi

if ! confirm "Proceed with uninstall?"; then
  log "Cancelled"
  exit 0
fi

if [ -n "$BIN_PATH" ]; then
  remove_file "$BIN_PATH"
fi

if [ -e "$CONFIG_DIR" ]; then
  remove_dir "$CONFIG_DIR"
else
  log "Config directory not present: $CONFIG_DIR"
fi

if [ "$REMOVE_VAULT" = true ]; then
  if [ -n "$VAULT_PATH" ] && [ -e "$VAULT_PATH" ]; then
    remove_dir "$VAULT_PATH"
  elif [ -n "$VAULT_PATH" ]; then
    log "Vault directory not present: $VAULT_PATH"
  else
    log "Skipped vault removal: no configured vault path found"
  fi
else
  if [ -n "$VAULT_PATH" ]; then
    log "Preserved vault: $VAULT_PATH"
    log "Run ./uninstall.sh --all to remove it too"
  fi
fi

echo ""
log "✓ Uninstall complete"
