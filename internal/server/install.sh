#!/bin/sh
# PeerClaw CLI installer
# Usage: curl -fsSL https://peerclaw.ai/install.sh | sh
#
# Downloads the latest peerclaw CLI binary for your platform.

set -e

REPO="peerclaw/peerclaw-cli"
INSTALL_DIR="${PEERCLAW_INSTALL_DIR:-/usr/local/bin}"
BINARY="peerclaw"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Get latest release tag
echo "Detecting latest version..."
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$TAG" ]; then
  TAG="latest"
fi
echo "Version: $TAG"

# Download binary (GoReleaser produces .tar.gz archives)
ARCHIVE="${BINARY}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"
echo "Downloading ${BINARY} for ${OS}/${ARCH}..."

TMPDIR=$(mktemp -d)
TMP="${TMPDIR}/${BINARY}"
trap 'rm -rf "$TMPDIR"' EXIT

if curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$URL"; then
  tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR" "$BINARY"
else
  echo "Error: failed to download from $URL" >&2
  echo "" >&2
  echo "You can also install with Go:" >&2
  echo "  go install github.com/peerclaw/peerclaw-cli/cmd/peerclaw@latest" >&2
  exit 1
fi

chmod +x "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (may require sudo)..."
  sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "peerclaw installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Verify: peerclaw version"
