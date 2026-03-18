// Package claude implements the Claude Code CLI backend adapter.
//
// This adapter invokes the `claude` CLI tool in non-interactive mode
// and translates Forge's abstract approval profiles into Claude-specific
// flags. It parses Claude's JSON output into normalized ExecResult.
//
// Key Claude CLI flags used:
//   - --print: non-interactive mode, output to stdout
//   - -p "prompt": pass prompt directly
//   - --output-format json: machine-readable output
//   - --permission-mode: ask|auto|bypass
//   - --max-turns N: limit interaction turns
//   - --max-budget-usd N: cost limit
//   - --mcp-config path: MCP server configuration
//   - --dangerously-skip-permissions: full-trust only
package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

// Backend implements the Claude Code CLI adapter.
type Backend struct {
	binary string // path to claude binary, default "claude"
}

// New creates a Claude Code backend with the given binary path.
func New(binary string) *Backend {
	if binary == "" {
		binary = "claude"
	}
	return &Backend{binary: binary}
}

// ID returns the backend identifier.
func (b *Backend) ID() string { return "cli:claude" }

// Capabilities returns Claude Code's capability set.
func (b *Backend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{
		StructuredOutput: true,
		StreamingEvents:  true,
		MCP:              true,
		Tools:            true,
		CostTracking:     true,
		FilePatchMode:    true,
		JsonSchemaOutput: true,
	}
}

// Execute runs the Claude CLI with the given request.
func (b *Backend) Execute(ctx context.Context, req *core.ExecRequest, sink core.EventSink) (*core.ExecResult, error) {
	args := b.BuildArgs(req)

	if sink != nil {
		sink.OnEvent(core.Event{Type: "cli_start", Payload: map[string]any{
			"binary": b.binary,
			"args":   args,
		}})
	}

	// Safe execution via os/exec — no shell involved
	cmd := exec.CommandContext(ctx, b.binary, args...)
	cmd.Dir = req.WorkspaceDir

	// Set environment
	if len(req.Env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range req.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude CLI failed: %w\noutput: %s", err, string(output))
	}

	result, err := parseOutput(output)
	if err != nil {
		// Fallback: treat entire output as the response text
		result = &core.ExecResult{
			FinalMessage: string(output),
		}
	}

	if sink != nil {
		sink.OnEvent(core.Event{Type: "cli_complete", Payload: result})
	}

	return result, nil
}

// HealthCheck verifies the claude CLI is installed and accessible.
func (b *Backend) HealthCheck(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, b.binary, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude not found or not executable: %w\n%s", err, string(output))
	}
	return nil
}

// BuildArgs translates an ExecRequest into Claude CLI arguments.
// Exported for testing.
func (b *Backend) BuildArgs(req *core.ExecRequest) []string {
	args := []string{
		"--print",
		"--output-format", "json",
	}

	// Translate approval profile to Claude CLI permission flags.
	// Valid --permission-mode values: acceptEdits, bypassPermissions, default, dontAsk, plan, auto
	switch req.ApprovalProfile {
	case core.ProfilePlan:
		args = append(args, "--permission-mode", "plan")
	case core.ProfilePatch:
		args = append(args, "--permission-mode", "default")
	case core.ProfileWorkspaceAuto:
		args = append(args, "--permission-mode", "auto")
	case core.ProfileSandboxAuto:
		args = append(args, "--permission-mode", "auto")
	case core.ProfileFullTrust:
		args = append(args, "--dangerously-skip-permissions")
	default:
		args = append(args, "--permission-mode", "default")
	}

	// Budget constraints
	if req.Budget.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", req.Budget.MaxTurns))
	}
	if req.Budget.MaxCostUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", req.Budget.MaxCostUSD))
	}

	// Prompt
	prompt := req.PromptPack.SpecBody
	if prompt != "" {
		args = append(args, "-p", prompt)
	}

	return args
}

// claudeJSONOutput represents Claude's JSON output format.
type claudeJSONOutput struct {
	Result  string `json:"result"`
	Cost    float64 `json:"cost_usd"`
	Turns   int    `json:"num_turns"`
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// parseOutput attempts to parse Claude's JSON output.
func parseOutput(data []byte) (*core.ExecResult, error) {
	// Claude --output-format json produces JSON lines; we want the last one
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty output")
	}

	lastLine := lines[len(lines)-1]
	var out claudeJSONOutput
	if err := json.Unmarshal([]byte(lastLine), &out); err != nil {
		return nil, fmt.Errorf("parsing JSON output: %w", err)
	}

	result := &core.ExecResult{
		FinalMessage: out.Result,
		Usage: core.Usage{
			InputTokens:  out.InputTokens,
			OutputTokens: out.OutputTokens,
			CostUSD:      out.Cost,
			Turns:        out.Turns,
		},
	}

	// TODO: Parse file patches from Claude's structured output
	// Claude can return file modifications in its result;
	// we need to extract them into PatchOps

	return result, nil
}
