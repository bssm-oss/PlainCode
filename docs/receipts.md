# Build Receipts

Every build produces a receipt — a complete audit trail stored at `.plaincode/builds/<build-id>/receipt.json`.

## Receipt Fields

```json
{
  "build_id": "7e49df33",
  "spec_id": "hello/greeter",
  "spec_hash": "f7204d2e810a3aba74893ef2a847e37a",
  "imported_spec_hashes": {},
  "backend_id": "cli:claude",
  "model": "",
  "cli_version": "",
  "approval_profile": "patch",
  "mcp_servers": [],
  "skills_used": [],
  "changed_files": ["internal/greeter/greeter.go"],
  "tests_run": 5,
  "tests_passed": 5,
  "tests_failed": 0,
  "coverage_before": 0.0,
  "coverage_after": 0.85,
  "retries": 0,
  "input_tokens": 1500,
  "output_tokens": 800,
  "cost_usd": 0.012,
  "duration_ms": 43789,
  "status": "success",
  "error": "",
  "started_at": "2026-03-18T18:11:00Z",
  "completed_at": "2026-03-18T18:11:43Z"
}
```

## Storage Location

```
.plaincode/
  builds/
    7e49df33/
      receipt.json
    cdcd8d01/
      receipt.json
```

## Use Cases

- **Dirty detection**: Compare `spec_hash` to determine if rebuild is needed
- **Audit trail**: Who built what, when, with which model, at what cost
- **Debugging**: Trace back from a code change to the exact spec version and backend
- **Cost tracking**: Aggregate `cost_usd` across builds
- **CI integration**: Parse JSON receipts in CI pipelines

## Inspecting Receipts

```bash
plaincode trace <build-id>
```

## JSON Schema

See `schemas/receipt.schema.json` for the formal schema definition.
