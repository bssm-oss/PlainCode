# Project Structure

```text
PlainCode/
├── cmd/
│   ├── plaincode/              # CLI entrypoint and user-facing commands
│   │   ├── main.go             # Command routing
│   │   ├── init.go             # Project scaffolding
│   │   ├── help.go             # Localized CLI help
│   │   ├── runtime.go          # run / stop / status
│   │   ├── logs.go             # runtime log and event inspection
│   │   └── test.go             # spec verification command
│   └── plaincoded/             # Daemon entrypoint
├── internal/
│   ├── app/                    # Build orchestration and spec loading
│   ├── backend/                # CLI/API backend adapters and registry
│   ├── config/                 # plaincode.yaml loading and defaults
│   ├── contextpack/            # Prompt/context assembly for builds
│   ├── execenv/                # Binary lookup and PATH normalization
│   ├── graph/                  # Dirty detection and dependency ordering
│   ├── hooks/                  # Build lifecycle hooks
│   ├── mcp/                    # MCP registry support
│   ├── policy/                 # Approval profile mapping
│   ├── receipt/                # Build receipt persistence
│   ├── runtime/                # Managed process/docker lifecycle
│   ├── server/                 # HTTP daemon surface
│   ├── skills/                 # AGENTS.md and SKILL.md loading
│   ├── spec/
│   │   ├── ast/                # Parsed spec types
│   │   ├── imports/            # Import resolution
│   │   ├── ir/                 # Normalized spec representation
│   │   └── parser/             # Markdown + YAML frontmatter parser
│   ├── takeover/               # Code → spec workflows
│   ├── telemetry/              # Profiling hooks
│   ├── validate/
│   │   ├── coverage/           # Coverage providers
│   │   ├── lint/               # Shared-file lint checks
│   │   ├── repair/             # Repair-loop context and artifacts
│   │   ├── speccheck/          # `plaincode test` oracle runner
│   │   └── test/               # `tests.command` execution
│   └── workspace/
│       ├── fsguard/            # Ownership validation
│       ├── patch/              # Patch application
│       ├── snapshot/           # Workspace rollback for retry safety
│       └── worktree/           # Optional git worktree helpers
├── docs/                       # User and design documentation
├── examples/                   # Example projects
├── pkg/                        # Public packages
├── prompts/                    # Embedded system prompt templates
├── schemas/                    # JSON schemas
└── tests/e2e/                  # Manual real-backend smoke fixtures
```

## Key Execution Paths

The most important paths for current day-to-day use are:

- `cmd/plaincode/main.go` → CLI entrypoint
- `internal/app/builder.go` → spec build orchestration
- `internal/runtime/manager.go` → process/docker runtime lifecycle
- `internal/validate/speccheck/checker.go` → `plaincode test`
- `internal/receipt/store.go` → build artifacts and metadata

## State Written At Runtime

PlainCode writes project-local state under `.plaincode/`:

- `.plaincode/builds/<build-id>/` — build receipts and validation artifacts
- `.plaincode/coverage/` — coverage output
- `.plaincode/runs/` — runtime state, logs, and event timelines

These directories are part of the product's normal operation and should usually be gitignored in downstream projects.
