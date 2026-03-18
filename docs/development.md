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

Current test suites (9):
- `internal/spec/parser` — Spec parsing, frontmatter validation, section extraction
- `internal/graph` — Topological sort, cycle detection, dirty propagation
- `internal/workspace/fsguard` — Ownership validation
- `internal/config` — Config load/save/validate
- `internal/receipt` — Receipt store save/load/query
- `internal/spec/ir` — IR resolution, ownership conflict detection
- `internal/backend/mock` — Mock backend execution
- `internal/backend/cli` — ParseFileBlocks, adapter ID/Capabilities, BuildArgs
- `internal/app` — E2E build pipeline with mock backend

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
- TODO/FIXME markers for unfinished work
- Errors wrap with `fmt.Errorf("context: %w", err)`
