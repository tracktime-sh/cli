#!/bin/bash
set -e

REPO="tracktime-sh/cli"
INSTALL_DIR="${TRACKTIME_INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux)
    ;;
  mingw*|msys*|cygwin*|windows*)
    OS="windows"
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Construct binary name
if [ "$OS" = "windows" ]; then
  BINARY="tracktime-${OS}-${ARCH}.exe"
else
  BINARY="tracktime-${OS}-${ARCH}"
fi

# Get latest version
echo "Fetching latest version..."
LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi

echo "Latest version: $LATEST"

# Download URL
URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download binary
echo "Downloading $BINARY..."
TEMP_FILE=$(mktemp)
if ! curl -sL "$URL" -o "$TEMP_FILE"; then
  echo "Failed to download $URL"
  rm -f "$TEMP_FILE"
  exit 1
fi

# Install binary
INSTALL_PATH="$INSTALL_DIR/tracktime"
if [ "$OS" = "windows" ]; then
  INSTALL_PATH="$INSTALL_DIR/tracktime.exe"
fi

mv "$TEMP_FILE" "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

echo ""
echo "tracktime $LATEST installed to $INSTALL_PATH"
echo ""

# Check if install dir is in PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    echo "Run 'tracktime login' to get started."
    ;;
  *)
    echo "Add $INSTALL_DIR to your PATH:"
    echo ""
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
    echo "Then run 'tracktime login' to get started."
    ;;
esac
