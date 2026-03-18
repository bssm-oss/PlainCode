#!/bin/bash
set -e

echo "Building plaincode..."
go build -o plaincode ./cmd/plaincode/

# Install to ~/.local/bin (same as freeapi, claude, etc.)
mkdir -p "$HOME/.local/bin"
mv plaincode "$HOME/.local/bin/plaincode"

echo "✅ Installed to ~/.local/bin/plaincode"
echo ""
"$HOME/.local/bin/plaincode" version
echo ""
echo "Run: plaincode version"
