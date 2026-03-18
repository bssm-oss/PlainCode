# Build Pipeline

The build pipeline is PlainCode's core — a 21-step process that transforms specs into validated code.

## Pipeline Steps

```
 1. Load plaincode.yaml
 2. Scan spec/ directory → parse all .md files
 3. Validate frontmatter (strict, reject unknown fields)
 4. Resolve spec imports (detect circular dependencies)
 5. Build dependency graph (topological sort)
 6. Mark dirty specs (compare hash vs last receipt)
 7. For each dirty spec (in dependency order):
     8.  Assemble context pack (spec + imports + agent rules + source files)
     9.  Select backend from preference list
    10.  Resolve approval profile → policy permissions
    11.  Run pre_exec hooks
    12.  Execute backend (claude/codex/gemini/copilot/cursor/opencode)
    13.  Parse output → extract PatchOps
    14.  Validate patches against ownership map
         - Reject readonly file modifications
         - Reject writes to other spec's owned files
         - Flag shared file modifications for extra validation
    15.  Run pre_apply hooks (lint guard, secret scan)
    16.  Apply patches to workspace
    17.  Run post_apply hooks (auto-format)
    18.  Execute tests (spec.tests.command)
    19.  Run post_test hooks
    20.  If tests fail → repair loop (classify failure, re-prompt backend)
    21.  Collect coverage → check against target
    22.  Save build receipt to .plaincode/builds/<id>/
    23.  Run on_receipt hooks
```

## Skippable Steps

| Flag | Skips |
|---|---|
| `--dry-run` | Steps 8-23 (parse and validate only) |
| `--skip-tests` | Steps 18-19 |
| `--skip-coverage` | Step 21 |

## Dirty Detection

A spec is "dirty" if:
- It has never been built (no receipt exists)
- Its SHA-256 hash differs from the last successful receipt
- Any of its imported specs are dirty (propagation)

## Repair Loop

When tests fail after build:
1. Classify failure: test failure / build error / ownership violation / coverage gap
2. Generate repair prompt with failure context
3. Re-execute backend with repair prompt
4. Re-validate and re-test
5. Repeat up to `retry_limit` (default: 3)

## Receipt Storage

Each build produces `.plaincode/builds/<build-id>/receipt.json` containing:
- Spec hash, backend ID, model, approval profile
- Changed files, test results, coverage delta
- Token usage, cost, duration
- Final status (success/failed/partial)
