# Installation

## Quick Install (Recommended)

```bash
git clone https://github.com/bssm-oss/PlainCode.git
cd PlainCode
./install.sh
```

This builds the binary and installs it to `~/.local/bin/plaincode`.

## go install

```bash
go install github.com/bssm-oss/PlainCode/cmd/plaincode@latest
```

> **Note**: This installs to `~/go/bin/`. If `plaincode: command not found`, add to your shell config:
> ```bash
> echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
> source ~/.zshrc
> ```

## Build from Source

```bash
git clone https://github.com/bssm-oss/PlainCode.git
cd PlainCode
make build        # Creates ./plaincode binary
make install      # Runs ./install.sh
make test         # Runs all tests
```

## Verify Installation

```bash
plaincode version
# plaincode 0.1.0-dev
```

## Prerequisites

- **Go 1.23+** (for building)
- At least one AI CLI tool installed:
  - `claude` (Claude Code)
  - `codex` (OpenAI Codex)
  - `gemini` (Google Gemini)
  - `copilot` (GitHub Copilot)
  - `cursor-cli` (Cursor)
  - `opencode` (OpenCode)

Check which tools are available:
```bash
plaincode providers doctor
```

## Uninstall

```bash
rm ~/.local/bin/plaincode
# or
rm ~/go/bin/plaincode
```
