---
id: health/server
language: go
managed_files:
  owned:
    - go.mod
    - go.sum
    - main.go
    - main_test.go
    - Dockerfile
    - .dockerignore
backend:
  preferred:
    - cli:codex
approval: workspace-auto
tests:
  command: go test ./...
coverage:
  target: 0.50
budget:
  max_turns: 8
  max_cost_usd: 5
---
# Purpose

Build a minimal Go HTTP service that exposes a single health endpoint for Codex smoke testing.

## Functional behavior

- The service listens on `PORT` and defaults to `8080` when `PORT` is unset.
- `GET /health` returns HTTP `200`.
- `GET /health` returns the exact JSON body `{"status":"good"}`.
- Any non-health path returns `404`.
- The service is implemented with `net/http` and runs as a `main` package.

## Inputs / outputs

- Input: process environment, especially `PORT`.
- Output: a small HTTP server binary and a test suite that validates the health response.

## Invariants

- Response content type for `/health` is JSON.
- The health payload is stable and does not include timestamps or random data.
- `go test ./...` must pass after generation.
- `go.sum` must exist in the final owned snapshot even if it is empty.

## Error cases

- If `PORT` is invalid, startup should fail rather than silently binding an unexpected port.

## Integration points

- `Dockerfile` should build the service and run it on port `8080`.
- `.dockerignore` should keep build context small and exclude generated or local state files.

## Observability

- Keep the service intentionally small and deterministic so smoke checks can compare the exact `/health` response.

## Test oracles

- `go test ./...` passes.
- `GET /health` returns status `200`.
- `GET /health` returns `{"status":"good"}` exactly after whitespace normalization.
