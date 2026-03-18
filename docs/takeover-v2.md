# Takeover v2: Round-Trip Verification

## Problem

Adopting a spec-first workflow requires converting existing code into specs. A naive approach (read code → generate spec markdown) produces specs that look reasonable but may not faithfully capture the code's behavior. If the spec is incomplete, rebuilding from it produces different code — silently breaking the system.

## Solution: Round-Trip Verification

Takeover v2 doesn't just generate a spec. It **verifies** that the spec can reproduce equivalent code by running a full round-trip:

```
Existing Code → Spec Draft → Delete Code → Rebuild from Spec → Compare
```

Only if the rebuilt code passes the same tests and preserves the same public API does the takeover succeed.

## Pipeline

### Stage 1: Analysis

1. **Select target**: file, package, or directory
2. **Parse code structure**: Using Tree-sitter (future) or Go AST
   - Public functions, types, interfaces
   - Import graph
   - Constants, errors, enums
   - Comments and doc strings
3. **Collect tests**: Find all test files and fixtures
4. **Baseline metrics**:
   - Run existing tests → pass/fail counts
   - Collect coverage → baseline percentage
   - Snapshot public API surface

### Stage 2: Spec Generation

5. **Generate spec draft**: Send code + analysis to AI backend
6. **Structure frontmatter**:
   - Auto-detect ownership (files that import from this package)
   - Suggest shared files (go.mod, shared types)
   - Mark upstream dependencies as readonly
7. **Populate body sections**: Purpose, behavior, invariants, error cases, test oracles

### Stage 3: Verification

8. **Create isolated worktree**: `git worktree add`
9. **Delete original implementation**: Remove owned files only
10. **Rebuild from spec**: Run normal `plaincode build` pipeline
11. **Compare results**:

| Metric | How | Threshold |
|---|---|---|
| Test pass rate | Run same test command | ≥ baseline |
| Public API delta | Compare exported symbols | 0 differences |
| Coverage delta | Compare coverage reports | ≥ baseline - 5% |
| Behavioral delta | Snapshot output comparison | Within tolerance |

### Stage 4: Scoring

12. **Compute confidence score** (0.0 – 1.0):

```
score = (test_pass_weight * test_ratio)
      + (api_weight * api_match)
      + (coverage_weight * coverage_ratio)
      + (behavior_weight * behavior_match)
```

Default weights: test=0.4, api=0.3, coverage=0.15, behavior=0.15

13. **Decision**:
   - `score ≥ 0.9`: Auto-promote to managed-by-spec
   - `0.7 ≤ score < 0.9`: Promote with warnings, list gaps
   - `score < 0.7`: Reject, show detailed diff report

### Stage 5: Promotion

14. **Register spec**: Add to plaincode.yaml spec registry
15. **Update ownership**: Declare owned/shared/readonly files
16. **Save receipt**: Full takeover receipt with confidence score and deltas

## CLI Interface

```bash
# Basic takeover
plaincode takeover internal/billing/invoice_pdf.go

# Takeover a package
plaincode takeover ./internal/billing/...

# Dry run (analyze + generate spec, don't verify)
plaincode takeover --dry-run internal/billing/

# Set custom confidence threshold
plaincode takeover --threshold 0.85 internal/billing/

# Use specific backend for spec generation
plaincode takeover --backend cli:claude internal/billing/
```

## Output

```
Takeover: internal/billing/invoice_pdf.go
  Analysis:
    Public API:  3 functions, 2 types, 1 interface
    Tests:       12 tests (all passing)
    Coverage:    78%

  Spec Generation:
    Generated:   spec/billing/invoice-pdf.md
    Owned files: internal/billing/invoice_pdf.go
                 internal/billing/invoice_pdf_test.go
    Shared:      go.mod, internal/billing/types.go

  Verification:
    Rebuild:     ✓ successful
    Tests:       12/12 passing (100%)
    API delta:   0 differences
    Coverage:    76% (baseline 78%, within tolerance)

  Confidence:    0.94

  ✓ Promoted to managed-by-spec
```

## Design Decisions

### Why round-trip instead of just generating specs?

A spec that "looks right" but can't reproduce the code is worse than no spec at all. It gives false confidence and will break on the first `plaincode build`. Round-trip verification catches this immediately.

### Why Tree-sitter for code analysis?

Tree-sitter provides:
- Fast, incremental parsing
- Language-agnostic interface
- Reliable AST for any file state (even with syntax errors)
- Go bindings available

For Phase 1, we use Go's `go/ast` package for Go files. Tree-sitter is planned for multi-language support.

### Why a confidence score instead of pass/fail?

Real codebases have edge cases. A strict pass/fail would reject most takeovers. A confidence score lets users:
- Auto-accept high-confidence takeovers
- Review and manually adjust borderline cases
- Understand exactly where the spec falls short
