// Package codex implements the OpenAI Codex CLI backend adapter.
//
// Invocation: codex exec [flags] "<prompt>"
// Key flags:
//   --full-auto: non-interactive automation
//   --output-last-message <file>: save last response to file
//   --json: JSON event stream output
//   --sandbox: enable sandboxed execution
//   --dangerously-bypass-approvals-and-sandbox: full-trust only
package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bssm-oss/PlainCode/internal/backend/cli"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

// Backend implements the Codex CLI adapter.
type Backend struct {
	binary string
}

// New creates a Codex CLI backend.
func New(binary string) *Backend {
	if binary == "" {
		binary = "codex"
	}
	return &Backend{binary: binary}
}

func (b *Backend) ID() string { return "cli:codex" }

func (b *Backend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{
		StructuredOutput: true,
		StreamingEvents:  true,
		MCP:              true,
		Tools:            true,
		CostTracking:     false,
		FilePatchMode:    true,
		JsonSchemaOutput: false,
	}
}

func (b *Backend) Execute(ctx context.Context, req *core.ExecRequest, sink core.EventSink) (*core.ExecResult, error) {
	args := b.BuildArgs(req)

	if sink != nil {
		sink.OnEvent(core.Event{Type: "cli_start", Payload: map[string]any{"binary": b.binary, "args": args}})
	}

	// Use temp file for output capture
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("forge-codex-%s.txt", req.BuildID))
	args = append(args, "--output-last-message", tmpFile)

	result, err := cli.ExecCLI(ctx, b.binary, args, req.WorkspaceDir, req.Env)
	if err != nil {
		return nil, fmt.Errorf("codex exec failed: %w", err)
	}

	// Read output from temp file
	output := result.Stdout
	if data, err := os.ReadFile(tmpFile); err == nil {
		output = string(data)
		os.Remove(tmpFile)
	}

	// Parse file blocks from output
	patches := cli.ParseFileBlocks(output)

	return &core.ExecResult{
		FinalMessage: output,
		Patches:      patches,
		Usage: core.Usage{
			Turns: 1,
		},
	}, nil
}

func (b *Backend) HealthCheck(ctx context.Context) error {
	return cli.CheckBinary(ctx, b.binary)
}

// buildArgs translates policy profile to Codex CLI flags.
func (b *Backend) BuildArgs(req *core.ExecRequest) []string {
	args := []string{"exec"}

	switch req.ApprovalProfile {
	case core.ProfilePlan:
		args = append(args, "--sandbox", "read-only")
	case core.ProfilePatch:
		// default mode — interactive approval
	case core.ProfileWorkspaceAuto:
		args = append(args, "--full-auto")
	case core.ProfileSandboxAuto:
		args = append(args, "--full-auto", "--sandbox")
	case core.ProfileFullTrust:
		args = append(args, "--full-auto", "--dangerously-bypass-approvals-and-sandbox")
	default:
		// safe default
	}

	prompt := req.PromptPack.SpecBody
	if prompt != "" {
		args = append(args, prompt)
	}

	return args
}
