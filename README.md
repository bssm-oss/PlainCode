# PlainCode

**A spec-first build orchestrator that compiles Markdown specs into real code using pluggable AI backends, validates and repairs the output, and enforces deterministic file ownership and build receipts.**

PlainCode is not a code generator. It is not a prompt wrapper. It is not a Claude wrapper.

PlainCode is an orchestrator that treats **specifications as the source of truth**, **AI as a swappable backend**, and **build outputs as reproducible, auditable artifacts**. It enables teams to adopt spec-driven development incrementally — without rewriting their entire codebase.

---

## Why Forge?

### The Problem

Current AI code generation tools share common weaknesses:

1. **Provider lock-in**: Tied to a single AI provider (Claude, GPT, etc.)
2. **No file governance**: AI can modify any file in your repository
3. **No build trail**: No record of what was generated, by which model, or why
4. **All-or-nothing adoption**: You either generate the whole project or nothing
5. **No verification loop**: Code is generated and hoped to work
6. **Installation friction**: Python runtimes, remote build services, heavy dependencies

### PlainCode's Answer

| Problem | Forge's Solution |
|---|---|
| Provider lock-in | 3 API backends + 6 CLI adapters, all interchangeable |
| No file governance | Three-tier ownership: owned / shared / readonly per spec |
| No build trail | Build receipts with full trace per build |
| All-or-nothing | Mixed mode: manage some specs, leave the rest alone |
| No verification | Test → repair → coverage loop built into every build |
| Installation friction | Single Go binary, zero runtime dependencies |

### What PlainCode Is NOT

- **Not a code generator.** Generators produce code and walk away. PlainCode compiles specs, validates results, repairs failures, tracks ownership, and leaves an audit trail.
- **Not a prompt wrapper.** PlainCode doesn't wrap a single AI tool. It orchestrates multiple backends through a unified policy engine.
- **Not an agent launcher.** PlainCode doesn't just call `claude` or `codex`. It manages the entire lifecycle: parse → graph → isolate → generate → validate → repair → receipt.

---

## Core Concepts

### Spec-First Development

Every code change starts with a spec. A spec is a Markdown file with structured YAML frontmatter that defines:
- What to build (purpose, behavior, invariants)
- Where to put it (file ownership)
- How to verify it (tests, coverage targets)
- What resources to use (backend, budget, approval level)

When you change a spec and run `plaincode build`, the system computes the diff, generates new code, validates it against tests and ownership rules, and saves a receipt.

### Three-Tier File Ownership

Each spec declares which files it controls:

- **owned**: Files exclusively managed by this spec. No other spec or manual edit should touch them.
- **shared**: Files that multiple specs may modify (e.g., `go.mod`). Extra validation applies: lint checks, import graph verification, conflict detection.
- **readonly**: Files the spec can reference but never modify. Useful for protecting stable APIs and legacy code.

### Mixed Mode

PlainCode doesn't require you to convert your entire codebase. You can:
1. Start with one spec for one module
2. Gradually add specs for more modules
3. Use `plaincode takeover` to extract specs from existing code
4. Coexist with manually-maintained code indefinitely

This is critical for real-world adoption in established repositories.

### Pluggable Backends

PlainCode supports two categories of AI backends:

**Remote API Backends** (direct model access):
| Backend | SDK | Status |
|---|---|---|
| OpenAI | Responses API via `openai-go` | Planned |
| Anthropic | Native Go SDK | Planned |
| Gemini | Google GenAI SDK | Planned |

**Local CLI Backends** (agent-based execution):
| Backend | Integration | Status |
|---|---|---|
| Claude Code | `claude --print --output-format json` | Planned |
| Codex CLI | `codex exec --json` | Planned |
| Gemini CLI | `gemini -p --output-format json` | Planned |
| Copilot CLI | `copilot -p --allow-tool` | Planned |
| OpenCode | CLI + server (OpenAPI 3.1) | Planned |
| Cursor | Generic CLI adapter | Planned |

All backends implement the same `Backend` interface. Switching from Claude to Codex is a config change, not a code change.

### Approval Profiles

Instead of exposing raw danger flags, PlainCode uses 5 abstract profiles:

| Profile | File Write | Shell | Network | Use Case |
|---|---|---|---|---|
| `plan` | deny | deny | deny | Dry-run only |
| `patch` | owned-only | ask | deny | Safe generation |
| `workspace-auto` | owned+shared | limited | allowlist | CI automation |
| `sandbox-auto` | allow | allow | allow | Isolated environments |
| `full-trust` | allow | allow | allow | Explicit opt-in only |

Each backend driver translates these profiles to provider-specific flags internally. See [docs/policy-engine.md](docs/policy-engine.md).

### Build Receipts

Every build produces a receipt at `.plaincode/builds/<build-id>/` containing:
- `receipt.json` — Full build metadata (spec hash, backend, model, tests, coverage, cost, duration)
- `events.ndjson` — Streaming event log
- `patches.json` — Applied file operations
- `tests.json` — Test results
- `coverage.json` — Coverage report
- `workspace.diff` — Full workspace diff

This ensures every code change is traceable to a specific spec version, backend, and build.

### Takeover v2

Converting existing code to specs is done through **round-trip verification**:

1. Analyze existing code (API surface, tests, coverage)
2. Generate spec draft
3. Delete code in isolated worktree
4. Rebuild from spec
5. Compare: test results, API delta, coverage delta
6. Compute confidence score
7. Promote only if above threshold

This prevents "spec rot" — specs that look right but can't actually reproduce the code. See [docs/takeover-v2.md](docs/takeover-v2.md).

---

## Quick Start

### Install

```bash
go install github.com/bssm-oss/PlainCode/cmd/forge@latest
```

### Initialize a Project

```bash
cd my-project
plaincode init
```

This creates:
- `plaincode.yaml` — project configuration
- `spec/` — directory for spec files
- `.plaincode/` — state directory (add to `.gitignore`)

### Write a Spec

Create `spec/hello/greeter.md`:

```markdown
---
id: hello/greeter
language: go
managed_files:
  owned:
    - internal/greeter/greeter.go
    - internal/greeter/greeter_test.go
  shared:
    - go.mod
backend:
  preferred:
    - cli:claude
    - openai:gpt-4o
approval: patch
tests:
  command: go test ./internal/greeter/...
coverage:
  target: 0.80
budget:
  max_turns: 5
  max_cost_usd: 2
---
# Purpose

A simple greeter module that generates personalized greeting messages.

## Functional behavior

- `Greet(name string) string` returns "Hello, {name}!" for non-empty names.
- `Greet("")` returns "Hello, World!".

## Test oracles

- `Greet("Alice")` == `"Hello, Alice!"`
- `Greet("")` == `"Hello, World!"`
```

### Build

```bash
# Parse and validate (available now)
plaincode build --spec hello/greeter --dry-run

# Full build (coming in Phase 2+)
plaincode build --spec hello/greeter
```

### Debug: Parse a Spec

```bash
plaincode parse-spec spec/hello/greeter.md
```

Outputs the parsed spec as JSON with all frontmatter fields and body sections.

---

## Project Configuration

### plaincode.yaml

```yaml
version: 1

project:
  spec_dir: spec
  state_dir: .plaincode
  default_language: go

defaults:
  backend: openai:gpt-4o
  approval: patch
  retry_limit: 3

providers:
  openai:gpt-4o:
    kind: openai
    model: gpt-4o
  anthropic:claude-sonnet:
    kind: anthropic
    model: claude-sonnet-4-20250514
  cli:claude:
    kind: cli-claude
    binary: claude
  cli:codex:
    kind: cli-codex
    binary: codex
```

See all configuration options in [docs/spec-format.md](docs/spec-format.md).

---

## Commands

### Core

| Command | Description |
|---|---|
| `plaincode init` | Initialize a new PlainCode project |
| `plaincode build [--spec <id>]` | Build one or all dirty specs |
| `plaincode build --dry-run` | Parse and validate only |
| `plaincode change -m "..."` | Fix implementation bug (spec stays unchanged) |
| `plaincode takeover <target>` | Extract spec from existing code |
| `plaincode coverage` | Run coverage analysis and gap filling |

### Inspection

| Command | Description |
|---|---|
| `plaincode providers list` | List configured backends |
| `plaincode providers doctor` | Health check all backends |
| `plaincode agents list` | List AGENTS.md and skills |
| `plaincode trace <build-id>` | Inspect build receipt |
| `plaincode explain <spec-id>` | Show spec dependencies and ownership |

### Platform

| Command | Description |
|---|---|
| `plaincode serve` | Start HTTP daemon (OpenAPI + SSE) |

### Development

| Command | Description |
|---|---|
| `plaincode parse-spec <file>` | Parse and dump a spec as JSON |
| `plaincode version` | Print version |

---

## Directory Layout

```
forge/
├── cmd/
│   ├── forge/              # CLI entrypoint
│   └── forged/             # Daemon entrypoint
├── internal/
│   ├── app/                # Command wiring, DI
│   ├── config/             # plaincode.yaml loader and validation
│   ├── spec/
│   │   ├── parser/         # Markdown + YAML frontmatter parser
│   │   ├── ast/            # Spec type definitions (no I/O)
│   │   ├── ir/             # Normalized, resolved spec representation
│   │   └── imports/        # Import resolver with cycle detection
│   ├── graph/              # Build dependency graph, topo sort
│   ├── workspace/
│   │   ├── worktree/       # Git worktree management
│   │   ├── patch/          # Patch abstraction and application
│   │   └── fsguard/        # File ownership validation
│   ├── contextpack/        # Prompt context assembly
│   ├── backend/
│   │   ├── core/           # Backend interface, registry, events
│   │   ├── openai/         # OpenAI Responses API
│   │   ├── anthropic/      # Anthropic native SDK
│   │   ├── gemini/         # Google GenAI SDK
│   │   └── cli/            # Local CLI adapters
│   │       ├── claude/     # Claude Code adapter
│   │       ├── codex/      # Codex CLI adapter
│   │       ├── gemini/     # Gemini CLI adapter
│   │       ├── copilot/    # Copilot CLI adapter
│   │       ├── cursor/     # Cursor generic adapter
│   │       └── opencode/   # OpenCode adapter
│   ├── policy/             # Approval profiles and permission engine
│   ├── skills/             # AGENTS.md / SKILL.md loader
│   ├── mcp/                # MCP server registry
│   ├── validate/
│   │   ├── test/           # Test runner abstraction
│   │   ├── coverage/       # Language-specific coverage providers
│   │   └── repair/         # Failure analysis and repair loop
│   ├── takeover/           # Code → spec with round-trip verification
│   ├── receipt/            # Build receipts and audit logs
│   ├── server/             # HTTP daemon (OpenAPI + SSE)
│   └── telemetry/          # pprof hooks and metrics
├── pkg/api/                # Public Go SDK (future)
├── schemas/                # JSON schemas for receipts, events
├── prompts/                # System prompt templates
├── examples/               # Example projects with sample specs
└── docs/                   # Design documents
```

---

## Current Implementation Status

### What Works Today

- **`plaincode init`**: Creates project structure with `plaincode.yaml`, `spec/`, `.plaincode/`
- **`plaincode build --spec <id> --dry-run`**: Parses spec, validates frontmatter strictly, reports parsed result
- **`plaincode parse-spec <file>`**: Debug command to dump parsed spec as JSON
- **Spec parser**: YAML frontmatter with strict unknown field rejection, Markdown body section extraction, SHA-256 hash computation
- **Import resolver**: Resolves spec imports with circular dependency detection
- **Build graph**: Topological sort, dirty detection, cycle detection
- **Ownership validator**: Three-tier model with cross-spec conflict detection
- **Policy engine**: 5 approval profiles with permission matrix
- **Build receipt**: Schema defined with JSON serialization

### What's Coming

See [docs/roadmap.md](docs/roadmap.md) for the full implementation plan.

---

## Design Principles

1. **Specs are the source of truth.** Code is derived. Specs are diffed, versioned, and reviewed.
2. **AI is a pluggable backend.** No provider lock-in. Switch backends with a config change.
3. **Builds are deterministic and auditable.** Every build produces a receipt.
4. **Adopt incrementally.** Mixed mode lets you spec-manage one module at a time.
5. **Measure before optimizing.** Pure Go first. pprof and benchmarks guide optimization.
6. **Policy over flags.** Abstract approval profiles, not raw provider flags.
7. **Verify, don't hope.** Tests, coverage, and repair loops are built into every build.

---

## Technology Choices

| Component | Choice | Why |
|---|---|---|
| Language | Go (pure, no cgo) | Single binary, cross-compile, `os/exec` safety |
| Spec parser | `yaml.v3` + heading extraction | Strict validation with `KnownFields`, pure Go |
| Workspace isolation | `git worktree` | Lightweight, linked to main repo |
| CLI execution | `os/exec` | No shell injection, arg-array safety |
| Profiling | `net/http/pprof` | Built-in, zero-config |
| Remote APIs | Native SDKs per provider | Best feature coverage |
| Policy format | TOML (planned) | Gemini CLI precedent, readable |

---

## Ecosystem Compatibility

PlainCode reads and respects existing agent configuration:
- `AGENTS.md` — project rules (CodeSpeak, Codex, OpenCode compatible)
- `.claude/CLAUDE.md` — Claude Code rules
- `.cursor/rules` — Cursor rules
- `.agents/skills/` — Agent skills
- `SKILL.md` — Skill definitions
- MCP server configurations

---

## FAQ

**Q: Is this a CodeSpeak clone?**
A: No. PlainCode shares the spec-first philosophy but is designed from scratch with multi-backend support, stronger file governance, round-trip takeover verification, and Go-native distribution. It can import from CodeSpeak projects.

**Q: Do I need an AI API key to use Forge?**
A: You need at least one configured backend (API key or local CLI tool) to build specs. Parsing, validation, and project management work without any AI backend.

**Q: Can I use PlainCode with an existing codebase?**
A: Yes. Mixed mode is a core feature. Start with one spec, use `plaincode takeover` to gradually convert existing code, and coexist with manual code indefinitely.

**Q: What languages are supported?**
A: The spec format is language-agnostic. Coverage providers are planned for Go (first), Python, TypeScript, Rust, and Java.

**Q: Is this production-ready?**
A: No. PlainCode is in early development (Phase 1). The spec parser, build graph, and ownership model are functional. Backend integration and full build pipeline are in progress.

**Q: Why Go instead of Rust/Python/TypeScript?**
A: Single binary distribution with zero dependencies. Safe subprocess execution via `os/exec`. Native cross-compilation. Built-in profiling. The target users (developers using AI coding tools) need something that installs in one command and just works.

---

## License

TBD

---

## Documentation

- [Architecture](docs/architecture.md) — System design and layer overview
- [Spec Format](docs/spec-format.md) — Complete spec file reference
- [Roadmap](docs/roadmap.md) — Implementation phases and status
- [Takeover v2](docs/takeover-v2.md) — Round-trip verification design
- [Policy Engine](docs/policy-engine.md) — Approval profiles and permission model
