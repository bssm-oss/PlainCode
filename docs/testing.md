# Testing Documentation

## Overview

PlainCode currently uses two verification layers:

1. **Repository automation**
   - `go test ./...`
   - `go vet ./...`
2. **Manual end-to-end smoke**
   - fresh project initialization
   - real backend build
   - spec-driven verification
   - managed runtime start/stop
   - process and Docker health checks

The goal is not just parser correctness. The project verifies the full spec lifecycle:

- parse spec
- detect dirty work
- generate files
- validate ownership
- run tests
- collect Go coverage
- save receipts
- start/stop managed runtimes
- preserve logs and lifecycle events for debugging

## Repository Verification

Run the standard suite from the repo root:

```bash
go test ./...
go vet ./...
```

The automated coverage includes these areas:

- `cmd/plaincode`
  - `init`, localized help, runtime/log/test CLI commands
- `internal/app`
  - build orchestration and receipt behavior
- `internal/backend/cli`
  - CLI adapter parsing and argument generation
- `internal/backend/mock`
  - deterministic backend execution
- `internal/config`
  - config defaults, validation, and round-trip loading
- `internal/execenv`
  - fallback binary/path resolution
- `internal/graph`
  - dependency graph ordering and dirty propagation
- `internal/receipt`
  - receipt persistence and hash lookup
- `internal/runtime`
  - process/docker lifecycle, state refresh, event logging
- `internal/spec/ir`
  - import resolution and ownership conflicts
- `internal/spec/parser`
  - strict frontmatter validation and section extraction
- `internal/validate/coverage`
  - Go coverage execution and parsing
- `internal/validate/repair`
  - repair prompt assembly and failure classification
- `internal/validate/speccheck`
  - `plaincode test` HTTP oracles
- `internal/validate/test`
  - `tests.command` execution
- `internal/workspace/fsguard`
  - owned/shared/readonly validation

## Desktop End-to-End Smoke

The practical smoke flow is:

```bash
plaincode init
plaincode build --spec health/server --json
plaincode test --spec health/server --json
plaincode run --spec health/server --mode process
plaincode status --spec health/server
plaincode logs --spec health/server --events
plaincode stop --spec health/server
plaincode run --spec health/server --mode docker
plaincode stop --spec health/server
```

Expected outcomes for the minimal health-service spec:

- `plaincode build` writes owned files:
  - `go.mod`
  - `go.sum`
  - `main.go`
  - `main_test.go`
  - `Dockerfile`
  - `.dockerignore`
- `go test ./...` passes
- `plaincode test` passes
- process runtime responds at `/health` with `200` and `{"status":"good"}`
- unknown path returns `404`
- Docker runtime responds the same way
- `.plaincode/builds/<build-id>/receipt.json` records tests and coverage
- `.plaincode/runs/<escaped-spec>.log` and `.events.jsonl` are written

## Runtime Debug Artifacts

Managed runtime commands write debugging artifacts under `.plaincode/runs/`:

- `<escaped-spec>.json`
  - last known runtime state
- `<escaped-spec>.log`
  - process stdout/stderr or Docker build/runtime logs
- `<escaped-spec>.events.jsonl`
  - lifecycle timeline such as:
    - `start_requested`
    - `process_spawned`
    - `docker_build_started`
    - `start_succeeded`
    - `stop_requested`
    - `stop_succeeded`
    - `state_changed`

These files are the first place to look when a service starts locally but a spec check fails.

## Spec Test Oracles

`plaincode test` currently supports a focused subset of HTTP oracle lines from `## Test oracles`:

- status checks
- status + JSON checks
- top-level JSON field value checks
- top-level JSON field length checks

Lines outside that subset are returned in `ignored_oracles` so gaps are explicit rather than silent.

## Tool Path Notes

Desktop smoke surfaced a practical issue: subprocesses like `go` and `docker` cannot be assumed to be on PATH in every shell. The runtime/test/coverage path now resolves common locations such as:

- `/usr/local/go/bin`
- `/opt/homebrew/bin`
- `/Applications/Docker.app/Contents/Resources/bin`

This behavior is especially important for macOS/Homebrew setups and Desktop-triggered runs.
