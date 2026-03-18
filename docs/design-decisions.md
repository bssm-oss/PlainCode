# Design Decisions

## Why Go?

- **Single binary**: Zero runtime dependencies. `go build` produces one executable.
- **Safe subprocess execution**: `os/exec` uses arg arrays, no shell injection possible.
- **Cross-compilation**: `GOOS=linux GOARCH=amd64 go build` just works (pure Go, no cgo).
- **Built-in profiling**: `net/http/pprof` for measurement-driven optimization.
- **Concurrency**: goroutines for parallel spec builds (future).

## Why not C for the parser?

The bottleneck is **not** parsing. It's:
- LLM API calls: 10-60 seconds
- Test execution: 1-30 seconds
- Repair loops: multiple rounds of the above

Parser benchmark: **18μs/op** on M4 Pro. Rewriting in C would save microseconds in a pipeline measured in minutes.

## Why not Python?

- Python requires a runtime (`python3`, `pip`, virtual environments)
- Installation friction is high
- PlainCode targets developers who want `go install` and done

## Why a policy engine instead of raw flags?

Each AI CLI has different permission flags:
- Claude: `--permission-mode`, `--dangerously-skip-permissions`
- Codex: `--full-auto`, `--dangerously-bypass-approvals-and-sandbox`
- Gemini: `--approval-mode`, `--yolo`
- Copilot: `--allow-all-tools`, `--allow-all`

Exposing all of these creates an unmanageable surface. Instead, 5 abstract profiles (plan/patch/workspace-auto/sandbox-auto/full-trust) cover 95% of use cases, and each adapter translates internally.

## Why three-tier file ownership?

Without ownership control, an AI backend could modify any file. In a monorepo with 50 specs, this is dangerous. The owned/shared/readonly model:
- Prevents cross-spec conflicts at build time
- Protects stable APIs from accidental modification
- Ensures shared files get extra validation

## Why build receipts?

"Why is this code like this?" is unanswerable without receipts. Each receipt records:
- Which spec version was the input
- Which AI model generated the code
- What tests were run
- How much it cost

This is essential for team adoption and debugging.

## Why Markdown specs instead of a DSL?

- Readable by humans without learning a new language
- Editable in any text editor
- Diffable in standard code review tools
- YAML frontmatter provides machine-parseable structure
- Body sections provide natural language context for AI backends

## Why mixed mode?

Requiring spec coverage for the entire codebase is a non-starter for adoption. Mixed mode lets teams:
1. Start with one module
2. Prove value
3. Gradually expand
4. Coexist with manual code indefinitely
