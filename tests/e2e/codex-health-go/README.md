# Codex Health Smoke

This fixture is a self-contained end-to-end target for PlainCode using the `cli:codex` backend.

## What It Generates

- `go.mod`
- `go.sum` (empty is acceptable)
- `main.go`
- `main_test.go`
- `Dockerfile`
- `.dockerignore`

The generated service must expose `GET /health` and return `{"status":"good"}`.

## Local Smoke

Run from this fixture directory:

```bash
PLAINCODE_BIN=/path/to/plaincode
/bin/bash scripts/local-smoke.sh
```

The script copies the fixture into a temporary directory first, so the source fixture stays clean.

## Optional Docker Smoke

If Docker is available, run:

```bash
PLAINCODE_BIN=/path/to/plaincode
/bin/bash scripts/docker-smoke.sh
```

If Docker is not installed, the script prints a skip message and exits successfully.
