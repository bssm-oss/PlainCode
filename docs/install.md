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

## Tool Path Resolution

For spec execution paths such as `plaincode test`, `plaincode run`, and Go coverage collection, PlainCode also searches common tool install locations when PATH is minimal:

- `/usr/local/go/bin`
- `/usr/local/bin`
- `/opt/homebrew/bin`
- `/Applications/Docker.app/Contents/Resources/bin`
- `/usr/bin`, `/bin`, `/usr/sbin`, `/sbin`

This mainly helps with `go` and `docker` in Desktop/macOS/Homebrew environments.

Provider CLIs such as `codex`, `claude`, or `gemini` should still either:
- be available on PATH, or
- be configured with an absolute `binary:` path in `plaincode.yaml`

## Uninstall

```bash
rm ~/.local/bin/plaincode
# or
rm ~/go/bin/plaincode
```
