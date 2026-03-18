# Configuration Reference

## plaincode.yaml

```yaml
version: 1

project:
  spec_dir: spec              # Directory containing spec files
  state_dir: .plaincode       # Build state and receipts
  default_language: go         # Default language for new specs

defaults:
  backend: cli:claude          # Default backend when spec doesn't specify
  approval: patch              # Default approval profile
  retry_limit: 3               # Max repair loop retries

providers:
  cli:claude:
    kind: cli-claude
    binary: claude             # Path or name of the CLI binary
  cli:codex:
    kind: cli-codex
    binary: codex
  cli:gemini:
    kind: cli-gemini
    binary: gemini
  cli:copilot:
    kind: cli-copilot
    binary: copilot
  cli:cursor:
    kind: cli-cursor
    binary: cursor-cli
  cli:opencode:
    kind: cli-opencode
    binary: opencode
```

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

## Defaults

When `plaincode init` creates a new project:

| Setting | Default |
|---|---|
| `spec_dir` | `spec` |
| `state_dir` | `.plaincode` |
| `default_language` | `go` |
| `backend` | `openai:gpt-4o` |
| `approval` | `patch` |
| `retry_limit` | `3` |

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
