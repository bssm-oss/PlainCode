# Development Guide

## Prerequisites

- Go 1.23+
- git

## Build

```bash
go build ./cmd/plaincode/
```

## Test

```bash
go test ./...
```

Current automated test packages include:
- `cmd/plaincode` — init/help/runtime/test CLI surfaces
- `internal/app` — build loop and receipt behavior
- `internal/backend/cli` — adapter parsing and argument mapping
- `internal/backend/mock` — deterministic backend behavior
- `internal/config` — config defaults and round-trip loading
- `internal/execenv` — fallback binary/path resolution
- `internal/graph` — dependency sorting and dirty propagation
- `internal/receipt` — receipt persistence
- `internal/runtime` — process/docker lifecycle and event logging
- `internal/spec/ir` — import resolution and ownership conflicts
- `internal/spec/parser` — frontmatter validation and section extraction
- `internal/validate/coverage` — Go coverage parsing and execution
- `internal/validate/repair` — repair classification/prompt assembly
- `internal/validate/speccheck` — `plaincode test` HTTP oracle execution
- `internal/validate/test` — `tests.command` execution
- `internal/workspace/fsguard` — patch ownership validation

Additional repo-level verification used during development:

```bash
go vet ./...
```

Manual smoke path for a fresh project:

```bash
plaincode init
plaincode build --spec health/server --json
plaincode test --spec health/server --json
plaincode run --spec health/server --mode process
plaincode run --spec health/server --mode docker
plaincode stop --spec health/server
```

## Benchmarks

```bash
go test -bench=. ./internal/spec/parser/
```

## Lint

```bash
go vet ./...
```

## Adding a New CLI Backend

1. Create `internal/backend/cli/<name>/<name>.go`
2. Implement `core.Backend`: `ID()`, `Capabilities()`, `Execute()`, `HealthCheck()`
3. Export `BuildArgs()` for testing
4. Add kind mapping in `internal/app/registry.go`
5. Add test in `internal/backend/cli/adapters_test.go`
6. Add to `plaincode.yaml` providers section

## Adding a New Coverage Provider

1. Create `internal/validate/coverage/<lang>_provider.go`
2. Implement `coverage.Provider`: `Language()`, `RunUnit()`, `RunIntegration()`, `FindGaps()`
3. Register in build pipeline

## Project Conventions

- All CLI invocations use `os/exec` with arg arrays (no shell)
- All public types and packages have godoc comments
- Tests use `t.TempDir()` for isolation
- Runtime/test/coverage subprocesses should use the shared binary/path resolution helpers in `internal/execenv`
- TODO/FIXME markers for unfinished work
- Errors wrap with `fmt.Errorf("context: %w", err)`
