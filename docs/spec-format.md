# Spec File Format

## Overview

A PlainCode spec is a Markdown file with structured YAML frontmatter. The spec is both human-readable documentation and machine-parseable build input.

Specs live in the `spec/` directory (configurable via `plaincode.yaml`). The file path determines the spec ID: `spec/billing/invoice-pdf.md` has ID `billing/invoice-pdf`.

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
