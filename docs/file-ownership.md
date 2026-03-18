# File Ownership Model

PlainCode enforces a three-tier file ownership model to prevent AI backends from modifying files they shouldn't.

## Three Tiers

### owned
Files exclusively managed by this spec. No other spec or manual edit should touch them.

```yaml
managed_files:
  owned:
    - internal/billing/invoice_pdf.go
    - internal/billing/invoice_pdf_test.go
```

### shared
Files that multiple specs may modify. Extra validation applies (lint, import graph check, conflict detection).

```yaml
managed_files:
  shared:
    - go.mod
    - internal/billing/types.go
```

### readonly
Files the spec can reference but never modify. Useful for protecting stable APIs and legacy code.

```yaml
managed_files:
  readonly:
    - internal/legacy/**
```

## Validation Rules

| Patch Target | owned (this spec) | owned (other spec) | shared | readonly | unmanaged |
|---|---|---|---|---|---|
| **Allowed?** | Yes | **No** | Yes (with checks) | **No** | Yes |

When a backend produces patches, PlainCode validates **every file path** before applying:

1. **Readonly** → Reject immediately
2. **Owned by another spec** → Reject (cross-spec conflict)
3. **Outside project** → Reject
4. **Shared** → Allow, but flag for extra validation
5. **Owned by this spec** → Allow
6. **Unmanaged** → Allow (for new files)

## Extra Validation for Shared Files

- Formatter/linter check
- Import graph sanity (no circular imports introduced)
- API break detection (exported symbols unchanged)
- Conflicting ownership detection (multiple specs claiming same file)

## Internal Classifications

Beyond the user-visible three tiers, the system tracks:

| Class | Meaning |
|---|---|
| `Owned` | This spec owns the file |
| `Shared` | Multiple specs share the file |
| `ReadOnly` | Cannot be modified |
| `OwnedByOtherSpec` | Another spec owns this file |
| `OutsideProject` | Path is outside the project root |
| `Unmanaged` | No spec claims this file |
