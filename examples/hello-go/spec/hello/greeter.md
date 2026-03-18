---
id: hello/greeter
language: go
imports: []
managed_files:
  owned:
    - internal/greeter/greeter.go
    - internal/greeter/greeter_test.go
  shared:
    - go.mod
  readonly: []
backend:
  preferred:
    - cli:claude
    - openai:gpt-4o
approval: patch
tests:
  command: go test ./internal/greeter/...
coverage:
  target: 0.80
budget:
  max_turns: 5
  max_cost_usd: 2
---
# Purpose

A simple greeter module that generates personalized greeting messages.

## Functional behavior

- `Greet(name string) string` returns "Hello, {name}!" for non-empty names.
- `Greet("")` returns "Hello, World!".
- `GreetFormal(name, title string) string` returns "{title} {name}, welcome."

## Inputs / outputs

| Function | Input | Output |
|---|---|---|
| Greet | name string | greeting string |
| GreetFormal | name, title string | formal greeting string |

## Invariants

- Output is never empty.
- Output always ends with punctuation (! or .).
- Name is trimmed of leading/trailing whitespace before use.

## Error cases

- None. All inputs produce valid output.

## Integration points

- Used by the HTTP handler layer (future spec: hello/server).

## Observability

- No metrics or logging required for this module.

## Test oracles

- `Greet("Alice")` == `"Hello, Alice!"`
- `Greet("")` == `"Hello, World!"`
- `GreetFormal("Smith", "Dr.")` == `"Dr. Smith, welcome."`

## Migration notes

- New module, no migration needed.
