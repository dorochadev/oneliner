#!/usr/bin/env bash
set -e

# Build the oneliner binary
if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. Please install Go and try again."
  exit 1
fi

echo "Building oneliner..."
go build -o oneliner .

# Choose install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  echo "You do not have write permissions to $INSTALL_DIR. Trying with sudo."
  sudo mv oneliner "$INSTALL_DIR/oneliner"
else
  mv oneliner "$INSTALL_DIR/oneliner"
fi

echo "oneliner installed to $INSTALL_DIR/oneliner"
echo "You can now run 'oneliner' from your terminal."
