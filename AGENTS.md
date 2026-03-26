# AGENTS.md

This file gives repository-specific guidance to coding agents working in `PlainCode`.

Use it as an execution contract for this repository, not as product marketing copy.

## 1. Repository Identity

PlainCode is a spec-first build orchestrator written in Go.

Its current practical scope is:

- parse Markdown specs with YAML frontmatter
- build code through pluggable CLI backends
- enforce file ownership (`owned`, `shared`, `readonly`)
- run `tests.command`
- run Go coverage checks
- retry through a repair loop on ownership / test / coverage failures
- save build receipts and validation artifacts
- manage long-running services with `plaincode run`, `status`, `stop`, and `logs`
- verify implementations against a spec with `plaincode test`

Do not assume every roadmap surface is complete. The codebase contains both active paths and scaffolded future paths.

## 2. What Is Real Today

The current user-facing flow that actually works is:

1. `plaincode init`
2. write a spec in `spec/*.md`
3. `plaincode build --spec <id>`
4. `plaincode test --spec <id>`
5. `plaincode run --spec <id>`
6. `plaincode logs --spec <id>`
7. `plaincode stop --spec <id>`

The main implementation paths are:

- `cmd/plaincode/main.go`
- `cmd/plaincode/init.go`
- `cmd/plaincode/help.go`
- `cmd/plaincode/runtime.go`
- `cmd/plaincode/logs.go`
- `cmd/plaincode/test.go`
- `internal/app/builder.go`
- `internal/runtime/manager.go`
- `internal/runtime/store.go`
- `internal/validate/speccheck/checker.go`
- `internal/validate/test/runner.go`
- `internal/validate/coverage/go_provider.go`
- `internal/receipt/store.go`

## 3. Major Product Constraints

### Build does not start services

`plaincode build` only generates code, runs validation, and writes artifacts.

It does not keep a process or container running.

If the task involves service lifecycle, use:

- `plaincode run`
- `plaincode status`
- `plaincode stop`
- `plaincode logs`

### Spec verification is intentionally narrow

`plaincode test` currently supports:

- `tests.command`
- a focused subset of HTTP oracle sentence patterns in `## Test oracles`

Do not claim full natural-language test oracle support.

If you extend spec oracle parsing, update:

- `internal/validate/speccheck/checker.go`
- `internal/validate/speccheck/checker_test.go`
- `docs/spec-format.md`
- `docs/cli-reference.md`
- `docs/testing.md`

### Go is the strongest supported language today

The spec model is language-agnostic, but the built-in coverage provider and default runtime inference are most complete for Go.

Do not document parity across languages unless the code really implements it.

## 4. Repository Defaults And Config Nuance

There are two different defaults you must not confuse:

- this repository's own `plaincode.yaml` currently defaults to `cli:claude`
- `plaincode init` generates new downstream projects with `cli:codex` as the default backend

When editing docs or examples, distinguish between:

- repo-maintainer defaults in this repo
- generated-project defaults from `plaincode init`

Relevant files:

- `plaincode.yaml`
- `internal/config/config.go`
- `cmd/plaincode/init.go`

## 5. Spec File Rules

Specs live under the configured `spec/` directory and must be Markdown files.

Important behavior:

- only `*.md` files are scanned as specs
- files whose basename starts with `_` are ignored by the loader
- `spec/_blueprint.md` is a template, not a build target

Relevant code:

- `internal/app/loader.go`
- `internal/spec/parser/parser.go`
- `internal/spec/ast/spec.go`

If you change spec format behavior, update:

- parser tests
- `docs/spec-format.md`
- `README.md`
- init scaffolding if the template changed

## 6. Runtime Rules

Runtime behavior is spec-driven via the `runtime` block.

Supported runtime modes:

- `auto`
- `process`
- `docker`

Compatibility note:

- `runtime.default_mode` is still accepted as a legacy alias
- `runtime.mode` is also accepted
- `runtime.process.cwd` is still accepted as a legacy alias of `working_dir`
- top-level `runtime.health_url` is still accepted as a fallback

When changing runtime parsing or behavior, preserve compatibility unless the task explicitly calls for a breaking change.

Relevant code:

- `internal/spec/ast/spec.go`
- `internal/spec/parser/parser.go`
- `internal/runtime/manager.go`

## 7. Subprocess And PATH Rules

PlainCode shells out to:

- backend CLIs
- `go`
- `docker`
- system tools like `ps`

This repository now centralizes fallback binary and PATH handling in:

- `internal/execenv/execenv.go`

If you touch subprocess execution in:

- runtime
- test runner
- coverage
- backend CLI health checks or execution

use the shared `execenv` helpers instead of ad hoc `PATH` or `exec.LookPath` logic.

Do not reintroduce code that assumes the app shell PATH is complete.

This matters especially on macOS desktop environments.

## 8. Build Artifacts And Runtime Artifacts

Build artifacts live under:

- `.plaincode/builds/<build-id>/receipt.json`
- `.plaincode/builds/<build-id>/tests.json`
- `.plaincode/builds/<build-id>/coverage.json`
- `.plaincode/builds/<build-id>/repair.json`

Runtime artifacts live under:

- `.plaincode/runs/<escaped-spec>.json`
- `.plaincode/runs/<escaped-spec>.log`
- `.plaincode/runs/<escaped-spec>.events.jsonl`

Coverage output may also appear under:

- `.plaincode/coverage/`

If you change artifact names, paths, or meaning, update:

- `README.md`
- `docs/config-reference.md`
- `docs/cli-reference.md`
- `docs/testing.md`

## 9. Preferred Verification Commands

For most code changes, use the narrowest sufficient verification first, then the broader repo check.

High-signal targeted checks:

```bash
go test ./cmd/plaincode ./internal/app ./internal/runtime ./internal/validate/speccheck ./internal/validate/test ./internal/validate/coverage ./internal/spec/parser
```

Full regression:

```bash
go test ./...
go vet ./...
```

If runtime behavior changed, also verify one or more of:

```bash
plaincode test --spec health/server --json
plaincode run --spec health/server --mode process
plaincode run --spec health/server --mode docker
plaincode logs --spec health/server --events
plaincode stop --spec health/server
```

If you need a real backend smoke path, use:

- `tests/e2e/codex-health-go`

Do not claim real-backend verification unless you actually ran it or you explicitly say you did not.

## 10. Desktop E2E Expectations

This repo has already been validated through a fresh desktop project flow.

A representative real-user flow is:

```bash
plaincode init
plaincode build --spec health/server --json
go test ./...
plaincode test --spec health/server --json
plaincode run --spec health/server --mode process
plaincode run --spec health/server --mode docker
plaincode stop --spec health/server
```

If you modify:

- init scaffolding
- runtime management
- spec verification
- CLI help or docs

prefer to re-run at least part of that flow in a clean external directory.

## 11. Documentation Sync Rules

When behavior changes, docs must move with code in the same change.

Usually relevant:

- `README.md`
- `docs/README.md`
- `docs/cli-reference.md`
- `docs/config-reference.md`
- `docs/spec-format.md`
- `docs/testing.md`
- `docs/project-structure.md`
- `docs/development.md`
- `docs/install.md`

Keep docs aligned with implemented behavior, not roadmap aspirations.

Specifically:

- do not describe daemon endpoints as implemented if they still return `not yet implemented`
- do not describe nonexistent receipt files
- do not describe unsupported oracle patterns as supported
- do not document init output that the command does not create

## 12. Server / Daemon Caveat

The HTTP daemon exists, but only part of its surface is implemented.

As of the current code:

- working: `/health`, `/providers`, `/policies`, `/openapi.json`
- scaffolded but not complete: `/build`, `/builds/:id`, `/events`

Relevant file:

- `internal/server/server.go`

Do not overstate daemon completeness in docs, comments, or PR summaries.

## 13. Safety Rules For Edits

When making changes in this repository:

- keep changes tightly scoped to the user request
- prefer extending existing code paths over introducing parallel implementations
- preserve backward compatibility for current spec/runtime aliases unless explicitly changing them
- do not silently change artifact names or CLI behavior without updating docs and tests
- do not edit `.plaincode/` state as source code
- do not commit temporary downstream projects, desktop smoke directories, or generated fixture output

## 14. If You Add A New CLI Backend

Expected work:

1. add adapter under `internal/backend/cli/<name>/`
2. implement `core.Backend`
3. register it in `internal/app/registry.go`
4. add tests in `internal/backend/cli/`
5. document config and capability expectations

Also verify that subprocess execution still uses safe arg-array calls and shared PATH resolution where appropriate.

## 15. If You Add A New Coverage Provider

Expected work:

1. add provider under `internal/validate/coverage/`
2. wire it into the build path
3. add unit tests for parsing and execution
4. update docs to reflect which languages now have first-class coverage support

Do not document a provider as supported just because a stub file exists.

## 16. Commit / PR Expectations

Prefer small, reviewable commits.

Good split examples in this repo:

- feature code
- docs refresh
- test/verification additions
- follow-up doc correction

If you update behavior and docs in one pass, small thematic commits are preferred over one giant commit.

## 17. Quick Checklist Before Finishing

Before closing work, check:

- code builds or tests cleanly
- docs match the new behavior
- CLI examples still reflect the actual commands
- runtime changes write the expected state/log/event artifacts
- no temporary generated project directories were added to the repo

If the task touched spec verification or runtime behavior, also note whether you validated:

- process mode
- docker mode
- receipt artifacts
- runtime event logs

## 18. Source Of Truth

When in doubt, trust the code over prose.

The highest-signal files for current behavior are:

- `cmd/plaincode/*.go`
- `internal/app/builder.go`
- `internal/runtime/*.go`
- `internal/validate/speccheck/checker.go`
- `internal/spec/parser/parser.go`
- `internal/spec/ast/spec.go`
- `internal/receipt/store.go`

If you find a mismatch between docs and code, fix the docs unless the task explicitly requires changing behavior.
