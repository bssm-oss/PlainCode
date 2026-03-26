# Testing

## Overview

PlainCode has two complementary testing layers:

- automated Go test suites for parser, builder, runtime, receipts, validation, and CLI behavior
- manual end-to-end smoke flows that exercise a real backend, a generated service, and runtime management

For repository regression checks, start with:

```bash
go test ./...
go vet ./...
```

## Automated Test Coverage

These packages carry the highest-signal automated checks today:

| Package | What it verifies |
|---|---|
| `cmd/plaincode` | CLI help, init scaffolding, command wiring |
| `internal/app` | build orchestration, receipts, retries, owned-file reconciliation |
| `internal/runtime` | process and Docker runtime lifecycle, state persistence, event logging |
| `internal/validate/test` | `tests.command` execution and exit handling |
| `internal/validate/coverage` | Go coverage collection and parsing |
| `internal/validate/speccheck` | `plaincode test` HTTP oracle parsing and runtime-assisted verification |
| `internal/spec/parser` | strict frontmatter validation and Markdown section extraction |
| `internal/workspace/fsguard` | ownership enforcement for owned/shared/readonly files |
| `internal/backend/cli` | CLI adapter argument generation and file-block parsing |

Useful targeted commands:

```bash
go test ./internal/app ./internal/runtime ./internal/validate/test ./internal/validate/coverage ./internal/validate/speccheck
go test -run TestManagerStartStatusStopProcess ./internal/runtime
go test -run TestBuilder ./internal/app
```

## Spec-Driven Verification

`plaincode test` is the main user-facing verification command.

It performs up to two layers of checking:

1. Run `tests.command` from the spec.
2. Parse supported HTTP sentences from `## Test oracles` and execute them against a managed runtime.

Example:

```bash
plaincode test --spec health/server
plaincode test --spec health/server --json
plaincode test --spec health/server --mode docker
plaincode test --spec health/server --skip-command
```

Current HTTP oracle support is intentionally narrow. It handles a focused subset of English and Korean sentence forms such as:

- ``GET /health returns status 200.``
- ``GET /health returns status 200 and {"status":"good"}.``
- ``GET /unknown returns status 404.``
- ``GET /api/solve?n=3 has moveCount 7``
- ``GET /api/solve?n=3 where moveCount is 7``
- ``GET /api/solve?n=3 의 moveCount 는 7 이다.``
- ``GET /api/solve?n=3 의 moves 길이는 7 이다.``

Anything outside that subset is left as documentation and reported under `ignored_oracles` in JSON output.

## Runtime Verification

Runtime management is tested both automatically and manually through:

- `plaincode run --spec <id> [--mode process|docker]`
- `plaincode status --spec <id>`
- `plaincode logs --spec <id>`
- `plaincode logs --spec <id> --events`
- `plaincode stop --spec <id>`

Runtime artifacts are stored in `.plaincode/runs/`:

- `<spec>.json` — latest runtime state
- `<spec>.log` — captured process output or Docker build/runtime log
- `<spec>.events.jsonl` — lifecycle timeline

This makes failed startup, healthcheck timeout, and unexpected exits debuggable after the process is gone.

## Real-Backend Smoke

The repository includes a manual real-backend fixture at:

- `tests/e2e/codex-health-go`

That fixture is intended to answer the question: "Can PlainCode generate, test, run, and stop a real service with Codex?"

Typical flow:

```bash
cd tests/e2e/codex-health-go
plaincode build --spec health/server
go test ./...
plaincode test --spec health/server
plaincode run --spec health/server --mode process
plaincode run --spec health/server --mode docker
plaincode stop --spec health/server
```

The generated service is a minimal Go `net/http` health endpoint with both process and Docker runtime definitions.

## Desktop / Local Shell Notes

On desktop shells, PATH can differ between your terminal and the app host. PlainCode now compensates for this by resolving common developer tools from standard install locations such as:

- `/usr/local/go/bin`
- `/usr/local/bin`
- `/opt/homebrew/bin`
- Docker Desktop's bundled binary directory

That behavior is used by:

- `plaincode test` when it runs `tests.command`
- Go coverage collection
- runtime process launch
- Docker build/run/stop helpers

## Recommended Release Checks

Before cutting a release or merging a substantial runtime/build change:

```bash
go test ./...
go vet ./...
plaincode help --lang ko
plaincode init
cd tests/e2e/codex-health-go && plaincode build --spec health/server
```

If the change touches runtime management, also verify:

```bash
plaincode test --spec health/server
plaincode run --spec health/server --mode process
plaincode logs --spec health/server --events
plaincode stop --spec health/server
```
