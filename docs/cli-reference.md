# CLI Reference

## plaincode

Spec-first multi-agent build orchestrator.

### Global Usage

```
plaincode <command> [options]
```

---

## Core Commands

### `plaincode init`

Initialize a new PlainCode project in the current directory.

```bash
plaincode init
```

Creates:
- `plaincode.yaml` — project configuration
- `spec/blueprint.md.txt` — starter blueprint you can copy into a real spec
- `README.plaincode.ko.md` — Korean quick-start guide for the new project
- `.plaincode/` — state directory (add to .gitignore)

---

### `plaincode build`

Build specs into code using the configured AI backend.

```bash
plaincode build [flags]
```

**Flags:**
| Flag | Description |
|---|---|
| `--spec <id>` | Build a specific spec by ID (e.g., `hello/greeter`) |
| `--dry-run` | Parse and validate only, don't execute backend |
| `--json` | Output results as JSON |
| `--skip-tests` | Skip test execution after build |
| `--skip-coverage` | Skip coverage analysis |

**Examples:**
```bash
# Build a specific spec
plaincode build --spec hello/greeter

# Validate without executing
plaincode build --spec hello/greeter --dry-run

# JSON output for CI integration
plaincode build --spec hello/greeter --json

# Skip tests for faster iteration
plaincode build --spec hello/greeter --skip-tests
```

**Build pipeline steps:**
1. Load and parse specs from `spec/` directory
2. Resolve imports and build dependency graph
3. Detect dirty specs (compare hash vs last receipt)
4. For each dirty spec: assemble context → select backend → execute → validate patches → apply → test → save receipt

---

### `plaincode change`

Submit an implementation change request. Use this when the **implementation is wrong** but the **spec is correct**.

```bash
plaincode change -m "fix: invoice total calculation off by one"
```

This is distinct from editing the spec. If the spec itself is wrong, edit the spec file directly and run `plaincode build`.

---

### `plaincode takeover`

Extract a spec from existing code with round-trip verification.

```bash
plaincode takeover <file|package>
```

**Examples:**
```bash
plaincode takeover internal/billing/invoice_pdf.go
plaincode takeover ./internal/billing/...
```

See [takeover-v2.md](takeover-v2.md) for the full verification pipeline.

---

### `plaincode coverage`

Run coverage analysis and identify gaps.

```bash
plaincode coverage
```

---

## Inspection Commands

### `plaincode providers list`

List all configured AI backends with their capabilities.

```bash
plaincode providers list
```

Output:
```
Registered backends (6):
  cli:claude           structured=true mcp=true tools=true
  cli:codex            structured=true mcp=true tools=true
  cli:gemini           structured=true mcp=true tools=true
  cli:copilot          structured=false mcp=true tools=true
  cli:cursor           structured=false mcp=false tools=false
  cli:opencode         structured=true mcp=true tools=true

Default: cli:claude
```

### `plaincode providers doctor`

Run health checks on all configured backends.

```bash
plaincode providers doctor
```

Output:
```
Health check:
  cli:claude           OK
  cli:codex            FAIL: codex not found
  cli:gemini           FAIL: gemini not found
```

---

### `plaincode trace`

Inspect a build receipt and its full audit trail.

```bash
plaincode trace <build-id>
```

---

### `plaincode explain`

Show spec dependencies, file ownership, and backend preferences.

```bash
plaincode explain <spec-id>
```

---

### `plaincode agents list`

List discovered AGENTS.md, SKILL.md, and agent rules.

```bash
plaincode agents list
```

---

## Platform Commands

### `plaincode serve`

Start the HTTP daemon for IDE/CI integration.

```bash
plaincode serve
```

Endpoints: `/build`, `/health`, `/providers`, `/policies`, `/openapi.json`

---

## Development Commands

### `plaincode parse-spec`

Parse a spec file and dump the result as JSON. Useful for debugging.

```bash
plaincode parse-spec spec/hello/greeter.md
```

### `plaincode version`

Print version.

```bash
plaincode version
# plaincode 0.1.0-dev
```

### `plaincode help`

Show CLI usage in the requested language.

```bash
plaincode help --lang ko
plaincode help --lang en
plaincode --help
plaincode -h
```
