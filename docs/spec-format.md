# Spec File Format

## Overview

A PlainCode spec is a Markdown file with structured YAML frontmatter. The spec is both human-readable documentation and machine-parseable build input.

Specs live in the `spec/` directory (configurable via `plaincode.yaml`). The file path determines the spec ID: `spec/billing/invoice-pdf.md` has ID `billing/invoice-pdf`.

Files whose basename starts with `_` are ignored by the loader. This is how `plaincode init` can ship `spec/_blueprint.md` as a reusable template without treating it as a build target.

Files whose basename starts with `_` are ignored by the loader. This lets `plaincode init` ship templates such as `spec/_blueprint.md` without treating them as buildable specs.

## Structure

```markdown
---
<YAML frontmatter>
---
<Markdown body with known sections>
```

## Frontmatter Fields

### Required Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique spec identifier (e.g., `billing/invoice-pdf`) |
| `language` | string | Target language: `go`, `python`, `typescript`, `rust`, `java` |

### File Ownership

| Field | Type | Description |
|---|---|---|
| `managed_files.owned` | string[] | Files exclusively owned by this spec |
| `managed_files.shared` | string[] | Files shared with other specs (extra validation) |
| `managed_files.readonly` | string[] | Files that can be read but not modified (supports globs) |

### Build Configuration

| Field | Type | Description |
|---|---|---|
| `imports` | string[] | Other spec IDs this spec depends on |
| `backend.preferred` | string[] | Preferred backends in priority order |
| `approval` | string | Autonomy profile: `plan`, `patch`, `workspace-auto`, `sandbox-auto`, `full-trust` |
| `tests.command` | string | Shell command to run tests |
| `coverage.target` | float | Target coverage percentage (0.0–1.0) |
| `budget.max_turns` | int | Maximum LLM interaction turns |
| `budget.max_cost_usd` | float | Maximum cost in USD |

### Runtime Configuration

`plaincode build` only generates code. `plaincode run`, `plaincode test`, `plaincode status`, and `plaincode stop` use the optional `runtime` block to manage long-running services.

| Field | Type | Description |
|---|---|---|
| `runtime.mode` | string | Preferred runtime launcher: `auto`, `process`, or `docker` |
| `runtime.default_mode` | string | Legacy alias for `runtime.mode` kept for compatibility |
| `runtime.health_url` | string | Legacy top-level healthcheck URL used when a mode-specific one is absent |
| `runtime.process.command` | string | Command to launch a host process |
| `runtime.process.working_dir` | string | Working directory relative to the project root |
| `runtime.process.cwd` | string | Legacy alias retained for compatibility |
| `runtime.process.env` | map | Environment variables for the process runtime |
| `runtime.process.healthcheck_url` | string | URL used to determine when the process is ready |
| `runtime.docker.context` | string | Docker build context |
| `runtime.docker.dockerfile` | string | Dockerfile path |
| `runtime.docker.image` | string | Explicit image name override |
| `runtime.docker.container_name` | string | Explicit container name override |
| `runtime.docker.ports` | string[] | Port mappings passed as `-p` |
| `runtime.docker.env` | map | Environment variables passed to `docker run` |
| `runtime.docker.healthcheck_url` | string | URL used to determine when the container is ready |

### Runtime Configuration

| Field | Type | Description |
|---|---|---|
| `runtime.default_mode` | string | Default runtime mode: `auto`, `process`, `docker` |
| `runtime.process.command` | string | Command used by `plaincode run --mode process` |
| `runtime.process.working_dir` | string | Working directory relative to project root |
| `runtime.process.env` | map | Environment overrides for the process runtime |
| `runtime.process.healthcheck_url` | string | URL checked before the runtime is considered ready |
| `runtime.docker.dockerfile` | string | Dockerfile path relative to project root |
| `runtime.docker.context` | string | Docker build context |
| `runtime.docker.image` | string | Optional explicit image name |
| `runtime.docker.container_name` | string | Optional explicit container name |
| `runtime.docker.ports` | string[] | Port mappings such as `18081:8080` |
| `runtime.docker.env` | map | Environment variables passed to `docker run` |
| `runtime.docker.healthcheck_url` | string | URL checked before Docker runtime is considered ready |

Compatibility aliases currently accepted by the parser:
- `runtime.mode`
- `runtime.health_url`
- `runtime.process.cwd`

### Strict Validation

The parser uses `yaml.v3` with `KnownFields(true)`. Unknown frontmatter fields are rejected at parse time. This prevents typos from silently becoming dead configuration.

## Body Sections

The Markdown body should contain these sections (by `##` heading):

| Section | Purpose |
|---|---|
| `# Purpose` | What this module does and why it exists |
| `## Functional behavior` | Detailed behavior description |
| `## Inputs / outputs` | Function signatures, data shapes |
| `## Invariants` | Properties that must always hold |
| `## Error cases` | How errors are handled |
| `## Integration points` | Dependencies on other modules/services |
| `## Observability` | Logging, metrics, tracing requirements |
| `## Test oracles` | Specific test cases with expected results |
| `## Migration notes` | Notes for migrating from existing code |

Sections are optional but recommended. The parser extracts them by heading for structured use in context packs.

`plaincode test` currently turns a focused subset of `## Test oracles` into executable HTTP checks. Supported patterns include both Korean and English sentence forms such as:

- ``GET /health returns status 200.``
- ``GET /health returns status 200 and {"status":"good"}.``
- ``GET /unknown returns status 404.``
- ``GET /api/solve?n=3 has moveCount 7``
- ``GET /api/solve?n=3 where moveCount is 7``
- ``GET /api/solve?n=3 의 moves 길이는 7 이다``

Other sentences are preserved as documentation and reported as ignored oracles by `plaincode test`.

## Example

```markdown
---
id: billing/invoice-pdf
language: go
imports:
  - billing/shared/money
managed_files:
  owned:
    - internal/billing/invoice_pdf.go
    - internal/billing/invoice_pdf_test.go
  shared:
    - go.mod
    - internal/billing/types.go
  readonly:
    - internal/legacy/**
backend:
  preferred:
    - openai:gpt-5
    - anthropic:claude-sonnet
    - cli:codex
approval: workspace-auto
tests:
  command: go test ./internal/billing/... -coverprofile=.plaincode/coverage/unit.out
coverage:
  target: 0.85
budget:
  max_turns: 12
  max_cost_usd: 5
runtime:
  mode: process
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

Generate PDF invoices for paid orders.

## Functional behavior

The invoice generator takes an Order struct and produces a PDF byte stream.
It formats line items, calculates totals using the shared money package,
and applies tax rules based on jurisdiction.

## Inputs / outputs

| Function | Input | Output |
|---|---|---|
| GenerateInvoice | Order, TaxConfig | []byte (PDF), error |
| FormatLineItem | LineItem | string |

## Invariants

- Total always equals sum of line items + tax.
- PDF output is never empty for valid orders.
- All monetary values use the shared money.Amount type.

## Error cases

- Empty order: return ErrEmptyOrder
- Negative amounts: return ErrInvalidAmount
- Unknown tax jurisdiction: return ErrUnknownJurisdiction

## Integration points

- billing/shared/money: Amount type and arithmetic
- storage service: saving generated PDFs (future)

## Test oracles

- Order with 2 items at $10 each, 10% tax → total $22.00
- Empty order → ErrEmptyOrder
- PDF output starts with %PDF magic bytes

## Migration notes

Replaces legacy invoice_gen.go. The old function signature changes from
GenerateInvoicePDF(orderID int64) to GenerateInvoice(Order, TaxConfig).
```

## Spec Hash

Every spec file gets a SHA-256 truncated hash computed from its raw content. This hash is used for:
- **Dirty detection**: comparing current hash vs. last build receipt hash
- **Receipt tracking**: recording exactly which spec version was built
- **Cache invalidation**: knowing when to rebuild

## Schema Evolution

The frontmatter schema is versioned implicitly via the `plaincode.yaml` `version` field. Future additions to the frontmatter schema will be backward-compatible (new optional fields only). Breaking changes will require a version bump.

## Runnable Oracle Subset

`plaincode test` currently parses a focused subset of HTTP oracle sentence patterns from `## Test oracles`. The supported shapes are:

- status checks: `GET /health returns status 200.`
- status + JSON checks: `GET /health returns status 200 and {"status":"good"}.`
- top-level field value checks: `GET /api/solve?n=3 where moveCount is 7`
- top-level field length checks: `GET /api/solve?n=3 의 moves 길이는 7 이다.`

Lines outside that subset are left in `ignored_oracles` so the command result stays transparent.
