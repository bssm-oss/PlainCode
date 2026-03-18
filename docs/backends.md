# Backend Adapters

PlainCode supports 6 CLI backends. Each adapter translates abstract approval profiles into provider-specific flags.

## Supported Backends

| Backend | Binary | Structured Output | MCP | Tools |
|---|---|---|---|---|
| Claude Code | `claude` | Yes | Yes | Yes |
| Codex CLI | `codex` | Yes | Yes | Yes |
| Gemini CLI | `gemini` | Yes | Yes | Yes |
| Copilot CLI | `copilot` | No | Yes | Yes |
| Cursor CLI | `cursor-cli` | No | No | No |
| OpenCode | `opencode` | Yes | Yes | Yes |

## Invocation Patterns

### Claude Code
```
claude --print -p "<prompt>" --output-format json --permission-mode <mode>
```

### Codex CLI
```
codex exec --full-auto --output-last-message <file> "<prompt>"
```

### Gemini CLI
```
gemini -p "<prompt>" --output-format json --approval-mode <mode>
```

### Copilot CLI
```
copilot -p "<prompt>" --allow-all-tools --no-ask-user
```

### Cursor CLI
```
cursor-cli generate --auto-run "<prompt>"
```

### OpenCode
```
opencode generate --auto "<prompt>"
```

## Policy-to-Flag Translation

### Claude Code

| Profile | Flags |
|---|---|
| plan | `--permission-mode plan` |
| patch | `--permission-mode default` |
| workspace-auto | `--permission-mode auto` |
| full-trust | `--dangerously-skip-permissions` |

### Codex CLI

| Profile | Flags |
|---|---|
| plan | `--sandbox read-only` |
| patch | (default interactive) |
| workspace-auto | `--full-auto` |
| sandbox-auto | `--full-auto --sandbox` |
| full-trust | `--full-auto --dangerously-bypass-approvals-and-sandbox` |

### Gemini CLI

| Profile | Flags |
|---|---|
| plan | (read-only, no flags) |
| patch | `--approval-mode ask` |
| workspace-auto | `--approval-mode auto` |
| full-trust | `--yolo` |

### Copilot CLI

| Profile | Flags |
|---|---|
| plan | (default, no auto-approve) |
| patch | `--allow-tool read_file --allow-tool write_file` |
| workspace-auto | `--allow-all-tools --no-ask-user` |
| full-trust | `--allow-all --yolo` |

### Cursor CLI

| Profile | Flags |
|---|---|
| plan | `generate` (no --auto-run) |
| workspace-auto+ | `generate --auto-run` |

### OpenCode

| Profile | Flags |
|---|---|
| plan | `generate --dry-run` |
| patch | `generate` (default) |
| workspace-auto+ | `generate --auto` |

## Adding a New Backend

1. Create `internal/backend/cli/<name>/<name>.go`
2. Implement `core.Backend` interface: `ID()`, `Capabilities()`, `Execute()`, `HealthCheck()`
3. Export `BuildArgs()` for testability
4. Add kind mapping in `internal/app/registry.go`
5. Add tests in `internal/backend/cli/adapters_test.go`
