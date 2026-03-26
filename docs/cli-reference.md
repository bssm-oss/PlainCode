# CLI Reference

## plaincode

Spec-first multi-agent build orchestrator.

### Global Usage

```
plaincode <command> [options]
```

---

## Core Commands

### `plaincode help`

Show CLI usage text in English or Korean.

```bash
plaincode help
plaincode help --lang ko
plaincode help --lang en
plaincode --help
plaincode -h
```

Language resolution order:
- `--lang`
- `PLAINCODE_LANG`
- `LC_ALL`
- `LC_MESSAGES`
- `LANG`

### `plaincode init`

Initialize a new PlainCode project in the current directory.

```bash
plaincode init
```

Creates:
- `plaincode.yaml` — project configuration
- `spec/_blueprint.md` — starter blueprint you can copy into a real spec
- `README.plaincode.ko.md` — Korean quick-start guide for the new project
- `.plaincode/` — state directory (add to .gitignore)
- `.plaincode/runs/` — managed runtime state
- `.plaincode/runs/*.log` / `*.events.jsonl` — stored runtime logs and lifecycle events

---

### `plaincode build`

Build specs into code using the configured AI backend.

`plaincode build` does not keep a server running. Use `plaincode run` to start a managed process or Docker container after a build.

```bash
plaincode build [flags]
```

**Flags:**
| Flag | Description |
|---|---|
| `--spec <id>` | Build a specific spec by ID (e.g., `hello/greeter`) |
| `--dry-run` | Parse and validate only, don't execute backend |
| `--json` | Output results as JSON |
| `--skip-tests` | Skip test execution after build |
| `--skip-coverage` | Skip coverage analysis |

**Examples:**
```bash
# Build a specific spec
plaincode build --spec hello/greeter

# Validate without executing
plaincode build --spec hello/greeter --dry-run

# JSON output for CI integration
plaincode build --spec hello/greeter --json

# Skip tests for faster iteration
plaincode build --spec hello/greeter --skip-tests
```

**Build pipeline steps:**
1. Load and parse specs from `spec/` directory
2. Resolve imports and build dependency graph
3. Detect dirty specs (compare hash vs last receipt)
4. For each dirty spec: assemble prompt/context → select backend → execute → validate ownership → apply owned/shared reconciliation
5. Run `tests.command` unless skipped
6. Run Go coverage when applicable unless skipped
7. Retry with repair context on validation failures up to `retry_limit`
8. Save receipt plus `tests.json` / `coverage.json`

**Notes:**
- `plaincode build` does not start long-running services.
- Build receipts live under `.plaincode/builds/<build-id>/`.
- Successful builds reconcile `managed_files.owned`: omitted owned files are treated as stale and removed.

---

### `plaincode test`

Verify the current implementation against a spec.

`plaincode test` runs `tests.command` and then evaluates parsed HTTP rules from the spec's `## Test oracles` section. If HTTP oracles require a runtime and no matching service is already running, PlainCode starts it automatically and stops it again when verification completes unless `--keep-running` is set.

```bash
plaincode test --spec hello/greeter
plaincode test --spec hello/greeter --json
plaincode test --spec hello/greeter --mode docker
plaincode test --spec hello/greeter --skip-command
```

**Flags:**
| Flag | Description |
|---|---|
| `--spec <id>` | Spec ID to verify |
| `--json` | Output the verification result as JSON |
| `--mode <auto|process|docker>` | Runtime mode used for HTTP oracles |
| `--skip-command` | Skip `tests.command` and only run parsed spec oracles |
| `--keep-running` | Keep the runtime running if `plaincode test` starts it |

**Supported HTTP oracle patterns today:**
- ``GET /health returns status 200.``
- ``GET /health returns status 200 and {"status":"good"}.``
- ``GET /unknown returns status 404.``
- ``GET /api/solve?n=3 has moveCount 7``
- ``GET /api/solve?n=3 where moveCount is 7``
- ``GET /api/solve?n=3 의 moveCount 는 7 이다.``
- ``GET /api/solve?n=3 의 moves 길이는 7 이다.``
- ``GET /api/solve?n=0 은 400 이다.``

Unhandled oracle lines are reported back in the JSON result under `ignored_oracles`.

---

### `plaincode run`

Start a managed service for a spec. This can launch a host process or a Docker container based on the spec's `runtime` block.

`plaincode build` does not start the service automatically. Use `plaincode run` when you want an explicit managed runtime.

```bash
plaincode run --spec hello/greeter --build
plaincode run --spec hello/greeter --mode process
plaincode run --spec hello/greeter --mode docker
```

**Flags:**
| Flag | Description |
|---|---|
| `--spec <id>` | Spec ID to start |
| `--build` | Build the spec before starting it |
| `--mode <auto|process|docker>` | Override runtime selection (otherwise the spec's runtime mode is used) |
| `--wait <duration>` | Wait for the runtime health check before returning |
| `--json` | Output runtime state as JSON |

**Notes:**
- Process mode resolves commands like `go` using common developer install paths on macOS/Homebrew/Linux if PATH is minimal.
- Docker mode stores both build output and runtime logs in `.plaincode/runs/<spec>.log`.

---

### `plaincode stop`

Stop a managed service for a spec.

```bash
plaincode stop --spec hello/greeter
plaincode stop --spec hello/greeter --json
```

---

### `plaincode status`

Show runtime state for one managed service or all tracked services.

```bash
plaincode status
plaincode status --spec hello/greeter
plaincode status --json
```

Shows:
- runtime mode
- status and health
- process PID or Docker container name
- healthcheck URL
- stored log and event paths

---

### `plaincode logs`

Show stored runtime logs or lifecycle events for a managed service.

```bash
plaincode logs --spec hello/greeter
plaincode logs --spec hello/greeter --events
plaincode logs --spec hello/greeter --tail 50
plaincode logs --spec hello/greeter --events --json
```

**Flags:**
| Flag | Description |
|---|---|
| `--spec <id>` | Spec ID to inspect |
| `--events` | Show lifecycle events instead of the main log |
| `--tail <n>` | Limit output to the last N lines/events |
| `--json` | Output events as JSON |
| `--path` | Print the artifact path only |

Artifacts written by runtime commands:
- `.plaincode/runs/<escaped-spec>.json`
- `.plaincode/runs/<escaped-spec>.log`
- `.plaincode/runs/<escaped-spec>.events.jsonl`

---

### `plaincode change`

Submit an implementation change request. Use this when the **implementation is wrong** but the **spec is correct**.

```bash
plaincode change -m "fix: invoice total calculation off by one"
```

This is distinct from editing the spec. If the spec itself is wrong, edit the spec file directly and run `plaincode build`.

---

### `plaincode takeover`

Extract a spec from existing code with round-trip verification.

```bash
plaincode takeover <file|package>
```

**Examples:**
```bash
plaincode takeover internal/billing/invoice_pdf.go
plaincode takeover ./internal/billing/...
```

See [takeover-v2.md](takeover-v2.md) for the full verification pipeline.

---

### `plaincode coverage`

Run coverage analysis and identify gaps.

```bash
plaincode coverage
```

---

## Inspection Commands

### `plaincode providers list`

List all configured AI backends with their capabilities.

```bash
plaincode providers list
```

Output:
```
Registered backends (6):
  cli:claude           structured=true mcp=true tools=true
  cli:codex            structured=true mcp=true tools=true
  cli:gemini           structured=true mcp=true tools=true
  cli:copilot          structured=false mcp=true tools=true
  cli:cursor           structured=false mcp=false tools=false
  cli:opencode         structured=true mcp=true tools=true

Default: cli:codex
```

In a freshly initialized project the default backend is `cli:codex`.

### `plaincode providers doctor`

Run health checks on all configured backends.

```bash
plaincode providers doctor
```

Output:
```
Health check:
  cli:claude           OK
  cli:codex            FAIL: codex not found
  cli:gemini           FAIL: gemini not found
```

---

### `plaincode trace`

Inspect a build receipt and its full audit trail.

```bash
plaincode trace <build-id>
```

---

### `plaincode explain`

Show spec dependencies, file ownership, and backend preferences.

```bash
plaincode explain <spec-id>
```

---

### `plaincode agents list`

List discovered AGENTS.md, SKILL.md, and agent rules.

```bash
plaincode agents list
```

---

## Platform Commands

### `plaincode serve`

Start the HTTP daemon for IDE/CI integration.

```bash
plaincode serve
```

Endpoints: `/build`, `/health`, `/providers`, `/policies`, `/openapi.json`

Current server surface:
- working now: `/health`, `/providers`, `/policies`, `/openapi.json`
- scaffolded but not implemented yet: `/build`, `/builds/:id`, `/events`

---

## Development Commands

### `plaincode parse-spec`

Parse a spec file and dump the result as JSON. Useful for debugging.

```bash
plaincode parse-spec spec/hello/greeter.md
```

### `plaincode version`

Print version.

```bash
plaincode version
# plaincode 0.1.0-dev
```

### `plaincode help`

Show CLI usage in the requested language.

```bash
plaincode help --lang ko
plaincode help --lang en
plaincode --help
plaincode -h
```
