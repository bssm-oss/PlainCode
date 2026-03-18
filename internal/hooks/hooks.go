// Package hooks implements build lifecycle hooks.
//
// Hooks allow users to run custom commands at specific points in the
// build pipeline. They can be used for:
//   - Pre-apply: lint guard, secret scan
//   - Post-apply: auto-formatter
//   - Pre-test: environment setup
//   - Post-test: coverage upload
//   - On-failure: alert, summarize
//   - On-receipt: PR body generation
//
// Hooks are defined in plaincode.yaml or in the hooks/ directory.
// Each hook is a shell command executed via os/exec (safe arg array).
package hooks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Event identifies a point in the build lifecycle.
type Event string

const (
	PrePlan   Event = "pre_plan"
	PreExec   Event = "pre_exec"
	PreApply  Event = "pre_apply"
	PostApply Event = "post_apply"
	PreTest   Event = "pre_test"
	PostTest  Event = "post_test"
	OnReceipt Event = "on_receipt"
	OnError   Event = "on_error"
)

// AllEvents returns all defined hook events.
func AllEvents() []Event {
	return []Event{PrePlan, PreExec, PreApply, PostApply, PreTest, PostTest, OnReceipt, OnError}
}

// Hook represents a single lifecycle hook.
type Hook struct {
	// Event is the lifecycle point this hook runs at.
	Event Event

	// Name is a human-readable name for this hook.
	Name string

	// Command is the shell command to execute.
	// Parsed into args by splitting on spaces (no shell expansion).
	Command string

	// Timeout is the maximum duration for this hook (0 = no limit).
	Timeout time.Duration

	// ContinueOnError controls whether build continues if hook fails.
	ContinueOnError bool
}

// HookResult holds the outcome of a hook execution.
type HookResult struct {
	Hook     Hook
	Passed   bool
	ExitCode int
	Output   string
	Duration time.Duration
	Error    error
}

// Runner executes hooks.
type Runner struct {
	hooks      map[Event][]Hook
	workingDir string
}

// NewRunner creates a hook runner with the given working directory.
func NewRunner(workingDir string) *Runner {
	return &Runner{
		hooks:      make(map[Event][]Hook),
		workingDir: workingDir,
	}
}

// Register adds a hook for the given event.
func (r *Runner) Register(h Hook) {
	r.hooks[h.Event] = append(r.hooks[h.Event], h)
}

// Run executes all hooks for the given event.
// Returns results for each hook. If a non-continuable hook fails,
// remaining hooks are skipped and the error is returned.
func (r *Runner) Run(ctx context.Context, event Event, env map[string]string) ([]HookResult, error) {
	hooks := r.hooks[event]
	if len(hooks) == 0 {
		return nil, nil
	}

	var results []HookResult
	for _, h := range hooks {
		result := r.executeHook(ctx, h, env)
		results = append(results, result)

		if !result.Passed && !h.ContinueOnError {
			return results, fmt.Errorf("hook %q failed: %s", h.Name, result.Output)
		}
	}

	return results, nil
}

// executeHook runs a single hook command.
func (r *Runner) executeHook(ctx context.Context, h Hook, env map[string]string) HookResult {
	if h.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.Timeout)
		defer cancel()
	}

	parts := strings.Fields(h.Command)
	if len(parts) == 0 {
		return HookResult{
			Hook:  h,
			Error: fmt.Errorf("empty hook command"),
		}
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = r.workingDir

	// Set environment variables
	if len(env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := HookResult{
		Hook:     h,
		Output:   string(output),
		Duration: duration,
	}

	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	} else {
		result.Passed = true
	}

	return result
}

// HasHooks returns true if any hooks are registered for the event.
func (r *Runner) HasHooks(event Event) bool {
	return len(r.hooks[event]) > 0
}
