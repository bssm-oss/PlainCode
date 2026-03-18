package parser

import "testing"

var benchSpec = []byte(`---
id: bench/test-spec
language: go
imports:
  - bench/shared
managed_files:
  owned:
    - internal/bench/main.go
    - internal/bench/main_test.go
  shared:
    - go.mod
  readonly:
    - internal/legacy/**
backend:
  preferred:
    - openai:gpt-4o
    - cli:claude
approval: workspace-auto
tests:
  command: go test ./internal/bench/...
coverage:
  target: 0.90
budget:
  max_turns: 10
  max_cost_usd: 3
---
# Purpose

Benchmark spec for parser performance testing.

## Functional behavior

This module processes data streams and produces aggregated results.
It handles concurrent inputs and maintains consistency guarantees.

## Inputs / outputs

| Function | Input | Output |
|---|---|---|
| Process | DataStream | AggregatedResult, error |
| Flush | none | error |

## Invariants

- All inputs are processed exactly once.
- Output order matches input order.
- No data loss under concurrent access.

## Error cases

- Empty stream: return ErrEmptyStream
- Malformed data: skip and log, continue processing
- Timeout: return partial results with ErrTimeout

## Integration points

- bench/shared: common data types
- storage service: result persistence

## Observability

- Metrics: process_count, error_count, latency_p99
- Logs: structured JSON at info level

## Test oracles

- Process([1,2,3]) == AggregatedResult{Sum: 6, Count: 3}
- Process([]) == ErrEmptyStream
- Concurrent Process calls do not race

## Migration notes

- Replaces legacy batch_processor.go
`)

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Parse(benchSpec)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSplitFrontmatter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := splitFrontmatter(benchSpec)
		if err != nil {
			b.Fatal(err)
		}
	}
}
