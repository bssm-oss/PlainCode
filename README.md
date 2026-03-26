# PlainCode

**A spec-first build orchestrator that compiles Markdown specs into real code using pluggable AI backends, validates and repairs the output, and enforces deterministic file ownership and build receipts.**

## Install

```bash
git clone https://github.com/bssm-oss/PlainCode.git && cd PlainCode && ./install.sh
```

Or with `go install` (requires `~/go/bin` in PATH):
```bash
go install github.com/bssm-oss/PlainCode/cmd/plaincode@latest
```

Works immediately:
```bash
plaincode version
plaincode init
plaincode help --lang ko
plaincode build --spec my-feature
plaincode test --spec my-feature
plaincode run --spec my-feature --build
plaincode providers list
```

---

PlainCode is not a code generator. It is not a prompt wrapper. It is not a Claude wrapper.

PlainCode is an orchestrator that treats **specifications as the source of truth**, **AI as a swappable backend**, and **build outputs as reproducible, auditable artifacts**. It enables teams to adopt spec-driven development incrementally — without rewriting their entire codebase.

---

## Why PlainCode?

### The Problem

Current AI code generation tools share common weaknesses:

1. **Provider lock-in**: Tied to a single AI provider (Claude, GPT, etc.)
2. **No file governance**: AI can modify any file in your repository
3. **No build trail**: No record of what was generated, by which model, or why
4. **All-or-nothing adoption**: You either generate the whole project or nothing
5. **No verification loop**: Code is generated and hoped to work
6. **Installation friction**: Python runtimes, remote build services, heavy dependencies

### PlainCode's Answer

| Problem | PlainCode's Solution |
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

`plaincode build` only produces code and receipts. `plaincode test` verifies the current implementation against the spec, and long-running services are managed explicitly with `plaincode run`, `plaincode status`, and `plaincode stop`.
Runtime logs and lifecycle events are persisted under `.plaincode/runs/` so failed starts and unexpected exits are debuggable after the fact.

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

**Remote API Backends** (direct model access, scaffolded):
| Backend | SDK | Status |
|---|---|---|
| OpenAI | Responses API via `openai-go` | Planned |
| Anthropic | Native Go SDK | Planned |
| Gemini | Google GenAI SDK | Planned |

**Local CLI Backends** (agent-based execution, usable today):
| Backend | Integration | Status |
|---|---|---|
| Claude Code | CLI adapter | Implemented |
| Codex CLI | CLI adapter | Implemented |
| Gemini CLI | CLI adapter | Implemented |
| Copilot CLI | CLI adapter | Implemented |
| OpenCode | CLI adapter | Implemented |
| Cursor | CLI adapter | Implemented |

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
- `receipt.json` — Full build metadata (spec hash, backend, tests, coverage, retries, duration)
- `tests.json` — Captured `tests.command` result when tests ran
- `coverage.json` — Coverage report when coverage ran
- `repair.json` — Repair attempt summary when retries were needed

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
go install github.com/bssm-oss/PlainCode/cmd/plaincode@latest
```

### Initialize a Project

```bash
cd my-project
plaincode init
```

This creates:
- `plaincode.yaml` — project configuration
- `spec/_blueprint.md` — starter blueprint you can copy into a real spec
- `README.plaincode.ko.md` — Korean quick-start guide
- `.plaincode/` — state directory (add to `.gitignore`)
- `.plaincode/runs/` — managed runtime state
- `.plaincode/runs/*.log` / `*.events.jsonl` — stored runtime logs and lifecycle timeline

You can view CLI help immediately:

```bash
plaincode help --lang ko
plaincode help --lang en
```

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
    - cli:codex
approval: patch
tests:
  command: go test ./internal/greeter/...
coverage:
  target: 0.80
budget:
  max_turns: 5
  max_cost_usd: 2
runtime:
  default_mode: process
  process:
    command: go run .
    working_dir: .
    healthcheck_url: http://127.0.0.1:8080/health
  docker:
    dockerfile: Dockerfile
    context: .
    ports:
      - 8080:8080
    healthcheck_url: http://127.0.0.1:8080/health
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

# Full build
plaincode build --spec hello/greeter

# Verify the implementation against the spec
plaincode test --spec hello/greeter
```

### Run And Stop A Service

```bash
# Build first, then start the managed service
plaincode run --spec hello/greeter --build

# Inspect runtime status
plaincode status --spec hello/greeter

# Stop the managed service
plaincode stop --spec hello/greeter

# Inspect stored logs
plaincode logs --spec hello/greeter
plaincode logs --spec hello/greeter --events
```

`plaincode build` does not auto-start long-running services. Build generates code and receipts; `run`, `status`, and `stop` manage the runtime lifecycle explicitly.

### Verified Flow

The current CLI/runtime flow has been validated on a clean Desktop project with a small Go health service spec:

```bash
plaincode init
plaincode build --spec health/server --json
plaincode test --spec health/server --json
plaincode run --spec health/server --mode process
plaincode run --spec health/server --mode docker
plaincode stop --spec health/server
```

That flow produced owned files, a successful build receipt, stored runtime logs/events, and working `/health` responses in both process and Docker modes.

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
  backend: cli:codex
  approval: patch
  retry_limit: 3

providers:
  cli:codex:
    kind: cli-codex
    binary: codex
```

See all configuration options in [docs/config-reference.md](docs/config-reference.md).

---

## Commands

### Core

| Command | Description |
|---|---|
| `plaincode init` | Initialize a new PlainCode project |
| `plaincode help [--lang ko|en]` | Show CLI usage in Korean or English |
| `plaincode build [--spec <id>]` | Build one or all dirty specs |
| `plaincode build --dry-run` | Parse and validate only |
| `plaincode test --spec <id>` | Verify implementation against spec tests and test oracles |
| `plaincode run --spec <id> [--build]` | Start a managed service |
| `plaincode stop --spec <id>` | Stop a managed service |
| `plaincode status [--spec <id>]` | Show managed service status |
| `plaincode logs --spec <id>` | Show stored runtime logs or events |
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
plaincode/
├── cmd/
│   ├── plaincode/          # CLI entrypoint
│   └── plaincoded/         # Daemon entrypoint
├── internal/
│   ├── app/                # Command wiring, DI
│   ├── config/             # plaincode.yaml loader and validation
│   ├── execenv/            # Common binary/path resolution for subprocesses
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
│   ├── runtime/            # Managed process / Docker lifecycle and logs
│   ├── validate/
│   │   ├── test/           # Test runner abstraction
│   │   ├── coverage/       # Language-specific coverage providers
│   │   ├── speccheck/      # Spec oracle runner (plaincode test)
│   │   └── repair/         # Failure analysis and repair loop
│   ├── takeover/           # Code → spec with round-trip verification
│   ├── receipt/            # Build receipts and audit logs
│   ├── server/             # HTTP daemon (OpenAPI + SSE)
│   └── telemetry/          # pprof hooks and metrics
├── pkg/                    # Public packages
├── schemas/                # JSON schemas for receipts, events
├── prompts/                # System prompt templates
├── examples/               # Example projects with sample specs
└── docs/                   # Design documents
```

---

## Current Implementation Status

### What Works Today

- **`plaincode init`**: Creates `plaincode.yaml`, `spec/_blueprint.md`, `README.plaincode.ko.md`, and `.plaincode/`
- **`plaincode help --lang ko|en`**: Prints localized CLI help in Korean or English
- **`plaincode build`**: Runs backend generation, ownership validation, tests, Go coverage, repair retries, and receipt saving
- **`plaincode test`**: Runs `tests.command` plus parsed HTTP spec oracles and can auto-start/stop the declared runtime
- **`plaincode run / status / stop / logs`**: Manages process and Docker runtimes with persisted state, logs, and lifecycle events
- **Spec parser**: YAML frontmatter with strict unknown field rejection, Markdown body section extraction, SHA-256 hash computation
- **Import resolver / build graph**: Import resolution, dirty detection, dependency ordering, and conflict checks
- **Ownership validator**: Three-tier owned/shared/readonly model with patch validation
- **Build receipt**: Receipt, test, coverage, and runtime artifacts under `.plaincode/`

### Current Limits

- `plaincode test` currently understands a focused subset of HTTP oracle sentence patterns, not arbitrary natural language
- Go is the only coverage provider wired into the build loop today
- The HTTP daemon and some takeover/server surfaces are still incomplete compared with the long-term roadmap

See [docs/roadmap.md](docs/roadmap.md) for the broader implementation plan.

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
| Workspace safety | snapshot + rollback, optional `git worktree` helpers | Keeps failed repair attempts from polluting the workspace |
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

**Q: Do I need an AI API key to use PlainCode?**
A: You need at least one configured backend (API key or local CLI tool) to build specs. Parsing, validation, and project management work without any AI backend.

**Q: Can I use PlainCode with an existing codebase?**
A: Yes. Mixed mode is a core feature. Start with one spec, use `plaincode takeover` to gradually convert existing code, and coexist with manual code indefinitely.

**Q: What languages are supported?**
A: The spec format is language-agnostic. The current validation and coverage path is strongest for Go, and the built-in coverage provider is Go-only today.

**Q: Is this production-ready?**
A: Not yet. The core CLI flow is usable and has been verified end-to-end, but the broader product is still early and some surfaces remain partial.

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
