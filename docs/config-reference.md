# Configuration Reference

## plaincode.yaml

```yaml
version: 1

project:
  spec_dir: spec              # Directory containing spec files
  state_dir: .plaincode       # Build state and receipts
  default_language: go         # Default language for new specs

defaults:
  backend: cli:codex           # Default backend when spec doesn't specify
  approval: patch              # Default approval profile
  retry_limit: 3               # Max repair loop retries

providers:
  cli:codex:
    kind: cli-codex
    binary: codex
```

`binary` may be either:
- a bare command name found on PATH
- an absolute path to the provider CLI binary

When PATH is minimal, PlainCode also looks in common developer tool locations such as `/usr/local/go/bin`, `/usr/local/bin`, `/opt/homebrew/bin`, and Docker Desktop's bundled binary directory.

## Provider Kinds

| Kind | Backend | Binary |
|---|---|---|
| `cli-claude` | Claude Code | `claude` |
| `cli-codex` | Codex CLI | `codex` |
| `cli-gemini` | Gemini CLI | `gemini` |
| `cli-copilot` | Copilot CLI | `copilot` |
| `cli-cursor` | Cursor CLI | `cursor-cli` |
| `cli-opencode` | OpenCode | `opencode` |
| `mock` | Mock (testing) | — |

For Desktop/macOS environments it can be safer to use an absolute path for provider binaries if your interactive shell and app shell do not share the same PATH.

## Defaults

When `plaincode init` creates a new project:

| Setting | Default |
|---|---|
| `spec_dir` | `spec` |
| `state_dir` | `.plaincode` |
| `default_language` | `go` |
| `backend` | `cli:codex` |
| `approval` | `patch` |
| `retry_limit` | `3` |

## State Directories

With the default `project.state_dir: .plaincode`, PlainCode stores:

- build receipts under `.plaincode/builds/<build-id>/`
- coverage artifacts under `.plaincode/coverage/`
- runtime state under `.plaincode/runs/`

Runtime artifacts created by `plaincode run`, `plaincode test`, and `plaincode logs` include:

- `<spec>.json` — latest runtime state snapshot
- `<spec>.log` — captured process output or Docker build/log output
- `<spec>.events.jsonl` — lifecycle event timeline

## Per-Spec Overrides

Each spec can override defaults in its frontmatter:

```yaml
---
backend:
  preferred:
    - cli:claude
    - cli:codex
approval: workspace-auto
budget:
  max_turns: 10
  max_cost_usd: 5
---
```
