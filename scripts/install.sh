#!/bin/bash
# Sovereign Stack — One-line installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Achilles1089/sovereign-stack/main/scripts/install.sh | bash

set -euo pipefail

REPO="Achilles1089/sovereign-stack"
BINARY="sovereign"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo ""
echo -e "${CYAN}  ⚡ Sovereign Stack — Installer${NC}"
echo "  ─────────────────────────────────"
echo ""

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo -e "${RED}  Unsupported architecture: $ARCH${NC}"
    exit 1
    ;;
esac

case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo -e "${RED}  Unsupported OS: $OS${NC}"
    exit 1
    ;;
esac

echo "  Platform: ${OS}/${ARCH}"

# Get latest release tag
echo "  Checking latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": ?"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "  No releases found. Building from source..."
  
  # Check Go installation
  if ! command -v go &>/dev/null; then
    echo -e "${RED}  Go is required to build from source.${NC}"
    echo "  Install from: https://go.dev/dl/"
    exit 1
  fi

  # Build from source
  TMPDIR=$(mktemp -d)
  echo "  Cloning repository..."
  git clone --depth 1 "https://github.com/${REPO}.git" "$TMPDIR/sovereign-stack" 2>/dev/null
  
  echo "  Building..."
  cd "$TMPDIR/sovereign-stack"
  go build -o "$BINARY" .
  
  echo "  Installing to ${INSTALL_DIR}..."
  if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY" "$INSTALL_DIR/"
  else
    sudo mv "$BINARY" "$INSTALL_DIR/"
  fi
  
  rm -rf "$TMPDIR"
else
  # Download release binary
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}-${OS}-${ARCH}"
  
  echo "  Downloading ${LATEST}..."
  TMPFILE=$(mktemp)
  curl -fsSL "$DOWNLOAD_URL" -o "$TMPFILE"
  chmod +x "$TMPFILE"
  
  echo "  Installing to ${INSTALL_DIR}..."
  if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "$INSTALL_DIR/$BINARY"
  else
    sudo mv "$TMPFILE" "$INSTALL_DIR/$BINARY"
  fi
fi

# Verify
if command -v sovereign &>/dev/null; then
  echo ""
  echo -e "${GREEN}  ✓ Sovereign Stack installed successfully!${NC}"
  echo ""
  echo "  Get started:"
  echo "    sovereign init      — Set up your server"
  echo "    sovereign --help    — See all commands"
  echo ""
else
  echo -e "${RED}  Installation may have failed. Check ${INSTALL_DIR}/${BINARY}${NC}"
fi
