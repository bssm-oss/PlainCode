// Package test provides a test runner abstraction for executing
// spec-defined test commands and collecting results.
//
// The test runner:
// 1. Executes the test command defined in the spec's tests.command field
// 2. Captures stdout/stderr
// 3. Determines pass/fail status from exit code
// 4. Parses test count if possible
package test

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result holds the outcome of a test run.
type Result struct {
	Passed   bool
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration

	// Parsed fields (best-effort)
	TotalTests  int
	PassedTests int
	FailedTests int
}

// Runner executes test commands.
type Runner struct{}

// NewRunner creates a test runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run executes a test command in the given directory.
// The command is split by spaces and executed via os/exec (no shell).
func (r *Runner) Run(ctx context.Context, workDir, command string) (*Result, error) {
	if command == "" {
		return nil, fmt.Errorf("no test command specified")
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty test command")
	}

	// Safe execution via os/exec — no shell involved
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = workDir

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &Result{
		Stdout:   string(output),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Passed = false
		} else {
			return nil, fmt.Errorf("executing test command: %w", err)
		}
	} else {
		result.Passed = true
		result.ExitCode = 0
	}

	// TODO: Parse test counts from output based on language
	// Go: "ok" / "FAIL" lines, "--- PASS:" / "--- FAIL:" counts
	// Python: pytest summary line
	// JS/TS: Jest/Vitest summary

	return result, nil
}
