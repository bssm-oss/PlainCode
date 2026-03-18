# Policy Engine

## Purpose

The policy engine provides a unified abstraction over the diverse permission models of AI backends. Instead of exposing raw provider flags (which differ across Claude, Codex, Gemini, Copilot, etc.), PlainCode defines abstract approval profiles that each backend driver translates to provider-specific options.

## The Problem

Every AI CLI tool has its own permission system:

| Tool | Flags |
|---|---|
| Claude Code | `--permission-mode`, `--allowedTools`, `--dangerously-skip-permissions` |
| Codex CLI | `--full-auto`, `--sandbox`, `--dangerously-bypass-approvals-and-sandbox` |
| Gemini CLI | `--approval-mode`, policy files (TOML) |
| Copilot CLI | `--allow-tool`, `--allow-all-tools`, `--allow-all`, `--yolo` |

Exposing these directly would create:
- An unmanageable API surface
- Security confusion (what does "yolo" mean for file safety?)
- Inconsistent behavior across backends

## Solution: Abstract Approval Profiles

Five profiles, from most restrictive to most permissive:

### `plan` — Read-only, no modifications

| Resource | Permission |
|---|---|
| File write | deny |
| Shell execution | deny |
| Network access | deny |
| Tool usage | deny |
| MCP access | deny |
| Dangerous flags | deny |

**Use case**: Dry-run, plan generation, spec validation.

### `patch` — Controlled file modifications

| Resource | Permission |
|---|---|
| File write | owned-only |
| Shell execution | ask |
| Network access | deny |
| Tool usage | ask |
| MCP access | deny |
| Dangerous flags | deny |

**Use case**: Safe code generation. The backend can only write to files declared as owned by the current spec.

### `workspace-auto` — Automated within workspace

| Resource | Permission |
|---|---|
| File write | owned-or-shared |
| Shell execution | allow-limited |
| Network access | allow-allowlist |
| Tool usage | allow |
| MCP access | allow |
| Dangerous flags | deny |

**Use case**: CI automation. The backend can modify owned and shared files, run limited shell commands, and access allowlisted network endpoints.

### `sandbox-auto` — Full automation in sandbox

| Resource | Permission |
|---|---|
| File write | allow |
| Shell execution | allow |
| Network access | allow |
| Tool usage | allow |
| MCP access | allow |
| Dangerous flags | deny |

**Use case**: Isolated environments (containers, VMs). Near-full autonomy but without truly dangerous operations.

### `full-trust` — Unrestricted (explicit opt-in)

| Resource | Permission |
|---|---|
| File write | allow |
| Shell execution | allow |
| Network access | allow |
| Tool usage | allow |
| MCP access | allow |
| Dangerous flags | allow |

**Use case**: When you explicitly trust the backend and environment. Maps to `--dangerously-skip-permissions` (Claude), `--yolo` (Codex/Copilot). Must be explicitly requested.

## Backend Translation

Each backend driver translates the abstract profile to concrete flags:

### Claude Code

| Profile | Flags |
|---|---|
| plan | `--print --output-format json` (no execution) |
| patch | `--permission-mode ask --allowedTools write_file` |
| workspace-auto | `--permission-mode auto` |
| full-trust | `--dangerously-skip-permissions` |

### Codex CLI

| Profile | Flags |
|---|---|
| plan | `codex exec --sandbox read-only` |
| patch | `codex exec --sandbox write-owned` |
| workspace-auto | `codex exec --full-auto` |
| full-trust | `codex exec --dangerously-bypass-approvals-and-sandbox` |

### Gemini CLI

| Profile | Flags |
|---|---|
| plan | `--approval-mode deny-all` |
| patch | `--approval-mode ask` |
| workspace-auto | `--approval-mode auto` |
| full-trust | `--approval-mode yolo` |

## Configuration

### Per-spec override

```yaml
# In spec frontmatter
approval: workspace-auto
```

### Project default

```yaml
# In plaincode.yaml
defaults:
  approval: patch
```

### Policy file (advanced)

```toml
# policies/custom.toml
[profiles.ci-build]
tools = "allow"
file_write = "owned-or-shared"
shell = "allow-limited"
network = "allow-allowlist"
dangerous_flags = false

[profiles.ci-build.network_allowlist]
domains = ["registry.npmjs.org", "proxy.golang.org"]
```

## File Ownership Enforcement

The policy engine works in conjunction with the file ownership model:

1. Backend produces patches
2. Policy engine checks each patch against the ownership map
3. Violations are rejected before any file is modified

```
Patch: write internal/billing/invoice_pdf.go
  → Ownership check: owned by current spec ✓
  → Policy check: file_write = owned-only ✓ → ALLOW

Patch: write internal/auth/middleware.go
  → Ownership check: owned by auth/middleware spec
  → Policy check: file_write = owned-only ✗ → REJECT

Patch: write go.mod
  → Ownership check: shared file
  → Policy check: file_write = owned-only ✗ → REJECT (would pass under owned-or-shared)
```

## Design Decisions

### Why not just expose raw flags?

1. **Security**: Users shouldn't need to understand each tool's permission model.
2. **Portability**: Switching backends shouldn't change security posture.
3. **Auditability**: Receipts record the abstract profile, not raw flags.
4. **Evolution**: New backends can be added without changing the user-facing model.

### Why TOML for policy files?

Gemini CLI already uses TOML for its policy engine. TOML is well-suited for configuration (readable, typed, hierarchical) and familiar to the target audience.

### Why 5 profiles instead of fine-grained per-permission?

Most users need one of 5 common patterns. Fine-grained control is available via custom policy files for advanced users, but the defaults cover 95% of use cases.
