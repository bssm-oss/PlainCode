# PlainCode Roadmap

## Phase 1: Core Foundation (MVP) ← Current

**Goal**: Spec parsing, build graph, ownership validation, receipt skeleton.

| Component | Status | Notes |
|---|---|---|
| CLI entrypoint | Done | `plaincode init`, `plaincode build --spec`, `plaincode parse-spec` |
| Config loader | Done | `plaincode.yaml` with defaults and validation |
| Spec parser | Done | goldmark-free YAML frontmatter + Markdown body |
| Spec AST | Done | Full type definitions with ownership model |
| Spec IR | Skeleton | Normalized representation, needs resolver integration |
| Import resolver | Done | With circular import detection |
| Build graph | Done | Topological sort, dirty detection, cycle detection |
| Ownership validator | Done | 3-tier model: owned/shared/readonly |
| Policy engine | Done | 5 approval profiles with permission matrix |
| Build receipt | Done | Schema defined, JSON serialization |
| Parser tests | Done | Frontmatter validation, section extraction |

**What works today**: You can `plaincode init` a project, write specs, parse them with strict validation, and dump the parsed result as JSON.

## Phase 2: API Backends

**Goal**: Connect to OpenAI, Anthropic, Gemini via native Go SDKs.

| Component | Status | Notes |
|---|---|---|
| OpenAI backend | Not started | Responses API via openai-go |
| Anthropic backend | Not started | Native Go SDK |
| Gemini backend | Not started | Google GenAI SDK |
| Streaming events | Not started | Normalized event model |
| Cost tracking | Not started | Token/cost accounting |
| Structured output | Not started | JSON schema enforcement |

## Phase 3: CLI Backends

**Goal**: Wrap local AI CLI tools as backends.

| Backend | Status | Key Integration Points |
|---|---|---|
| Claude Code | Not started | `--print`, `--output-format json`, `--permission-mode` |
| Codex CLI | Not started | `codex exec`, `--json`, `--full-auto` |
| Gemini CLI | Not started | `-p`, `--approval-mode`, `--output-format` |
| Copilot CLI | Not started | `-p`, `--allow-tool`, `--model` |
| OpenCode | Not started | CLI + server modes |
| Cursor | Not started | Generic CLI adapter |

## Phase 4: Build Pipeline

**Goal**: Full isolated build with git worktree, patch engine, file validation.

| Component | Status |
|---|---|
| Git worktree management | Not started |
| Patch engine (write/delete/rename/diff) | Types defined |
| Context pack assembly | Not started |
| Managed files validation | Logic done, integration pending |
| Mixed mode support | Not started |
| Whitelist management | Not started |

## Phase 5: Validation & Repair

**Goal**: Test execution, coverage analysis, automatic repair loops.

| Component | Status |
|---|---|
| Test runner abstraction | Not started |
| Go coverage provider | Not started |
| Python coverage provider | Not started |
| JS/TS coverage provider | Not started |
| Failure analysis | Not started |
| Repair loop orchestration | Not started |
| Change request separation | Not started |

## Phase 6: Takeover v2

**Goal**: Extract specs from existing code with round-trip verification.

| Component | Status |
|---|---|
| Code structure analysis | Not started |
| Public API extraction | Not started |
| Spec draft generation | Not started |
| Round-trip verification | Not started |
| Confidence scoring | Not started |

## Phase 7: Ecosystem Integration

**Goal**: AGENTS.md, SKILL.md, MCP, hooks, daemon.

| Component | Status |
|---|---|
| AGENTS.md reader | Not started |
| SKILL.md loader | Not started |
| MCP server bridge | Not started |
| Build lifecycle hooks | Not started |
| HTTP daemon | Not started |
| OpenAPI spec | Not started |
| SSE event stream | Not started |

## Phase 8: Optimization

**Goal**: Profile, benchmark, optimize measured hot paths.

| Component | Status |
|---|---|
| pprof integration | Not started |
| Benchmark suite | Not started |
| Incremental builds | Not started |
| Parallel spec builds | Not started |
| Build cache | Not started |

## What Is NOT Planned

- C/C++ rewrite of any component
- GUI / desktop application
- Cloud-hosted build service
- Custom LLM training
- IDE plugin (daemon API enables third-party plugins instead)
