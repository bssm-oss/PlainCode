# PlainCode Architecture

## Design Philosophy

PlainCode is not a code generator. It is a **spec-first build orchestrator** that treats specifications as the source of truth and AI as a swappable backend.

Three axioms drive every design decision:

1. **Specs are the source of truth.** Code is a build artifact derived from specs.
2. **AI is a pluggable backend.** No provider lock-in. API and CLI agents are interchangeable.
3. **Builds are deterministic and auditable.** Every build produces a receipt with full traceability.

## System Layers

```
┌─────────────────────────────────────────────┐
│                CLI / Daemon                  │
│           (cmd/forge, cmd/forged)            │
├─────────────────────────────────────────────┤
│              Config Loader                   │
│           (internal/config)                  │
├─────────────────────────────────────────────┤
│            Spec Pipeline                     │
│   Parser → IR → Import Resolver → Graph      │
├─────────────────────────────────────────────┤
│           Build Orchestrator                 │
│   Context Pack → Backend → Patch → Validate  │
├───────────────┬─────────────────────────────┤
│  Backend      │    Workspace                 │
│  ┌─────────┐  │    ┌──────────────────┐      │
│  │ API     │  │    │ Git Worktree     │      │
│  │ OpenAI  │  │    │ Patch Engine     │      │
│  │ Anthro  │  │    │ FS Guard         │      │
│  │ Gemini  │  │    │ Snapshot         │      │
│  ├─────────┤  │    └──────────────────┘      │
│  │ CLI     │  │                              │
│  │ Claude  │  │    Policy Engine             │
│  │ Codex   │  │    ┌──────────────────┐      │
│  │ Gemini  │  │    │ Approval Profiles│      │
│  │ Copilot │  │    │ Permission Map   │      │
│  │ OCode   │  │    │ Driver Translate │      │
│  │ Cursor  │  │    └──────────────────┘      │
│  └─────────┘  │                              │
├───────────────┴─────────────────────────────┤
│           Validation Layer                   │
│   Test Runner → Coverage → Repair Loop       │
├─────────────────────────────────────────────┤
│           Receipt / Trace                    │
│   Build Receipts → Event Log → Audit        │
└─────────────────────────────────────────────┘
```

## Data Flow

```
Spec Markdown
    ↓ parse (goldmark + yaml.v3)
Spec AST
    ↓ resolve imports, normalize paths
Spec IR
    ↓ topological sort, dirty detection
Build Graph
    ↓ for each dirty spec:
    │   ↓ create git worktree
    │   ↓ assemble context pack
    │   ↓ select backend (API or CLI)
    │   ↓ execute with policy constraints
    │   ↓ normalize output → Patch Set
    │   ↓ validate against ownership map
    │   ↓ apply patches
    │   ↓ run tests
    │   ↓ repair loop if needed
    │   ↓ coverage check
    │   ↓ save receipt
Build Receipt
```

## Key Design Decisions

### Why Go?

- Single binary distribution, zero runtime dependencies.
- `os/exec` provides safe subprocess execution without shell injection.
- `net/http` handles both API clients and daemon server.
- `pprof` enables measurement-driven optimization.
- Cross-compilation is trivial for pure Go (no cgo).

### Why not C for the parser?

The bottleneck in spec-to-code systems is not parsing. It is:
- LLM API latency (seconds to minutes)
- Test execution (seconds to minutes)
- Repair loops (multiple rounds)

A C parser would save microseconds in a pipeline measured in minutes. We optimize where it matters: parallelism, caching, incremental builds.

### Why a policy engine instead of raw flags?

Each AI CLI tool has its own permission model:
- Claude: `--permission-mode`, `--allowedTools`
- Codex: `--full-auto`, `--sandbox`
- Gemini: `--approval-mode`
- Copilot: `--allow-tool`, `--allow-all`

Exposing these directly creates an unmanageable surface. Instead, PlainCode defines 5 abstract profiles (plan, patch, workspace-auto, sandbox-auto, full-trust) and each backend driver translates them to provider-specific options.

### Why three-tier file ownership?

Without ownership control, an AI backend could modify any file in the repository. This is dangerous in monorepos and mixed-mode projects. The owned/shared/readonly model ensures:
- Specs cannot conflict on file ownership
- Shared files get extra validation (lint, import check)
- Readonly files are never accidentally modified
- Cross-spec conflicts are detected at build time

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `config` | Load plaincode.yaml, merge defaults, validate |
| `spec/parser` | Parse Markdown + YAML frontmatter into AST |
| `spec/ast` | Spec type definitions (no I/O) |
| `spec/ir` | Normalized, resolved spec representation |
| `spec/imports` | Resolve spec imports with cycle detection |
| `graph` | Build dependency graph, topo sort, dirty detection |
| `workspace/worktree` | Git worktree creation and cleanup |
| `workspace/patch` | Patch abstraction (write, delete, rename, diff) |
| `workspace/fsguard` | File ownership validation |
| `contextpack` | Assemble prompt context for backends |
| `backend/core` | Backend interface, registry, event types |
| `backend/openai` | OpenAI Responses API adapter |
| `backend/anthropic` | Anthropic native SDK adapter |
| `backend/gemini` | Google GenAI SDK adapter |
| `backend/cli/*` | Local CLI agent adapters |
| `policy` | Approval profiles and permission engine |
| `validate/test` | Test runner abstraction |
| `validate/coverage` | Language-specific coverage providers |
| `validate/repair` | Failure analysis and repair loop |
| `takeover` | Code-to-spec extraction with verification |
| `receipt` | Build receipt storage and querying |
| `server` | HTTP daemon with OpenAPI and SSE |
| `skills` | AGENTS.md / SKILL.md loader |
| `mcp` | MCP server registry and bridge |
| `telemetry` | pprof hooks and metrics |
