#!/bin/bash
set -e

echo "Building plaincode..."
go build -o plaincode ./cmd/plaincode/

# Try /usr/local/bin first, fall back to ~/go/bin
if cp plaincode /usr/local/bin/plaincode 2>/dev/null; then
    echo "✅ Installed to /usr/local/bin/plaincode"
else
    mkdir -p "$HOME/go/bin"
    cp plaincode "$HOME/go/bin/plaincode"
    echo "✅ Installed to ~/go/bin/plaincode"

    # Add to PATH if not already there
    if ! echo "$PATH" | grep -q "$HOME/go/bin"; then
        SHELL_RC="$HOME/.zshrc"
        [ -f "$HOME/.bashrc" ] && SHELL_RC="$HOME/.bashrc"
        if ! grep -q 'HOME/go/bin' "$SHELL_RC" 2>/dev/null; then
            echo 'export PATH=$PATH:$HOME/go/bin' >> "$SHELL_RC"
            echo "Added ~/go/bin to PATH in $SHELL_RC"
        fi
        export PATH=$PATH:$HOME/go/bin
    fi
fi

rm -f plaincode
echo ""
plaincode version
echo ""
echo "Done! Run: plaincode version"
echo "If 'command not found', run: source ~/.zshrc"
