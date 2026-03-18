# Project Structure

```
PlainCode/
├── cmd/
│   ├── plaincode/          # CLI entrypoint
│   │   └── main.go         # Command routing, flag parsing
│   └── plaincoded/         # Daemon entrypoint
│       └── main.go         # HTTP server startup
├── internal/
│   ├── app/                # Build orchestration
│   │   ├── builder.go      # 21-step build pipeline
│   │   ├── builder_test.go # E2E tests with mock backend
│   │   ├── loader.go       # Spec directory scanner
│   │   └── registry.go     # Backend auto-registration from config
│   ├── backend/
│   │   ├── core/           # Backend interface definitions
│   │   │   ├── backend.go  # Backend, CapabilitySet, ExecRequest/Result
│   │   │   ├── patch.go    # PatchOp: WriteFile, DeleteFile, RenameFile, ApplyDiff
│   │   │   └── registry.go # Registry: Register, Select, HealthCheckAll
│   │   ├── mock/           # Test backend (no API needed)
│   │   │   └── mock.go
│   │   └── cli/            # CLI backend adapters
│   │       ├── common.go   # ExecCLI, ParseFileBlocks, CheckBinary
│   │       ├── claude/     # claude --print -p <prompt>
│   │       ├── codex/      # codex exec --full-auto <prompt>
│   │       ├── gemini/     # gemini -p <prompt> --yolo
│   │       ├── copilot/    # copilot -p <prompt> --allow-all-tools
│   │       ├── cursor/     # cursor-cli generate --auto-run
│   │       └── opencode/   # opencode generate --auto
│   ├── config/             # plaincode.yaml loader
│   ├── contextpack/        # Prompt context assembly
│   ├── graph/              # Build dependency graph
│   ├── hooks/              # Build lifecycle hooks (8 events)
│   ├── mcp/                # MCP server registry
│   ├── policy/             # 5 approval profiles
│   ├── receipt/            # Build receipt store
│   ├── server/             # HTTP daemon (OpenAPI + SSE)
│   ├── skills/             # AGENTS.md / SKILL.md loader
│   ├── spec/
│   │   ├── ast/            # Spec type definitions
│   │   ├── ir/             # Normalized spec IR + resolver
│   │   ├── imports/        # Import resolver with cycle detection
│   │   └── parser/         # Markdown + YAML frontmatter parser
│   ├── takeover/           # Code → spec extraction
│   ├── validate/
│   │   ├── coverage/       # Language-specific coverage providers
│   │   ├── repair/         # Failure classification + repair loop
│   │   └── test/           # Test runner abstraction
│   └── workspace/
│       ├── fsguard/        # File ownership validator
│       ├── patch/          # Patch application engine
│       └── worktree/       # Git worktree manager
├── docs/                   # Documentation (you are here)
├── examples/               # Sample projects
├── prompts/                # System prompt templates
├── schemas/                # JSON schemas for receipts, specs
├── Makefile
├── install.sh
├── plaincode.yaml          # This project's own config
└── README.md
```

## Module Count

- **41 Go source files**
- **9 test suites** (all passing)
- **27 Go packages**
