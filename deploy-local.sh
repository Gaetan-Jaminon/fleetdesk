#!/bin/bash
set -euo pipefail

BLOB_URL="https://fluxyspackages.blob.core.windows.net/packages/fleetdesk_linux_amd64.tar.gz"
INSTALL_DIR="$HOME/.local/bin"

mkdir -p "$INSTALL_DIR"

echo "Downloading and installing fleetdesk..."
curl -sL "$BLOB_URL" | tar xz -C "$INSTALL_DIR" fleetdesk

echo "Done. $(fleetdesk --version 2>/dev/null || echo 'fleetdesk installed')"
