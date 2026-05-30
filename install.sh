#!/usr/bin/env bash
# distill installer
#
#   curl -fsSL https://raw.githubusercontent.com/MitudruDutta/distill/main/install.sh | bash
#
# Strategy:
#   1. Try the latest GitHub release for the host OS+arch (preferred).
#   2. Fall back to `go install` if Go >= 1.22 is on PATH.
#   3. Bail with a clear message otherwise.
#
# Install location: $DISTILL_PREFIX/bin (default: $HOME/.local/bin).
# Set DISTILL_VERSION=vX.Y.Z to pin a specific release.
set -euo pipefail

REPO="MitudruDutta/distill"
PREFIX="${DISTILL_PREFIX:-$HOME/.local}"
VERSION="${DISTILL_VERSION:-}"

log()  { printf '\033[1;34m▸\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# --- Detect platform ---------------------------------------------------------
uname_s=$(uname -s)
uname_m=$(uname -m)
case "$uname_s" in
  Linux)  os=linux ;;
  Darwin) os=darwin ;;
  *)      die "unsupported OS: $uname_s (try building from source: go install github.com/$REPO/cmd/distill@latest)" ;;
esac
case "$uname_m" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported arch: $uname_m" ;;
esac
log "detected $os/$arch"

# --- Resolve version ---------------------------------------------------------
if [ -z "$VERSION" ]; then
  if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
                | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1 || true)
  fi
fi

mkdir -p "$PREFIX/bin"

# --- Path 1: prebuilt release ------------------------------------------------
if [ -n "$VERSION" ]; then
  asset="distill_${VERSION#v}_${os}_${arch}.tar.gz"
  url="https://github.com/$REPO/releases/download/${VERSION}/${asset}"
  log "downloading $url"
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  if curl -fsSL "$url" -o "$tmp/$asset" 2>/dev/null; then
    tar -xzf "$tmp/$asset" -C "$tmp"
    install -m 0755 "$tmp/distill" "$PREFIX/bin/distill"
    log "installed $PREFIX/bin/distill (${VERSION})"
  else
    warn "release asset not found; falling back to source build"
    VERSION=""
  fi
fi

# --- Path 2: go install ------------------------------------------------------
if [ -z "$VERSION" ]; then
  if command -v go >/dev/null 2>&1; then
    log "building with go install (this takes ~10-30 s)"
    GOBIN="$PREFIX/bin" go install "github.com/$REPO/cmd/distill@latest"
    log "installed $PREFIX/bin/distill (built from source)"
  else
    die "no prebuilt release available and Go is not installed; install Go (https://go.dev/dl) and re-run"
  fi
fi

# --- Path-check + first-run hint --------------------------------------------
case ":$PATH:" in
  *":$PREFIX/bin:"*) ;;
  *) warn "$PREFIX/bin is not on your PATH. Add: export PATH=\"$PREFIX/bin:\$PATH\"" ;;
esac

"$PREFIX/bin/distill" --help >/dev/null 2>&1 || warn "binary did not run cleanly; check above output"

cat <<EOF

✓ distill installed.

Try it:
    distill --help
    echo 'a,b\n1,2' | distill -x csv
    distill ~/Downloads/some.pdf -o some.md

Use it as an MCP tool from your agent (Claude Desktop, Kiro CLI, etc.) by adding
to ~/.kiro/settings/mcp.json (or the equivalent client config):

  {
    "mcpServers": {
      "distill": { "command": "$PREFIX/bin/distill", "args": ["mcp"] }
    }
  }

Docs: https://github.com/$REPO#readme
EOF
