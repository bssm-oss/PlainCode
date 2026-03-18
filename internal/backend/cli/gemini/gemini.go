// Package gemini implements the Google Gemini CLI backend adapter.
//
// Invocation: gemini -p "<prompt>" [flags]
// Key flags:
//   -p / --prompt: pass prompt directly
//   --output-format json|text|stream-json
//   --approval-mode ask|auto
//   --yolo: full-trust mode (auto-approve everything)
package gemini

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/backend/cli"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

type Backend struct {
	binary string
}

func New(binary string) *Backend {
	if binary == "" {
		binary = "gemini"
	}
	return &Backend{binary: binary}
}

func (b *Backend) ID() string { return "cli:gemini" }

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

	result, err := cli.ExecCLI(ctx, b.binary, args, req.WorkspaceDir, req.Env)
	if err != nil {
		return nil, err
	}

	output := result.Stdout
	patches := cli.ParseFileBlocks(output)

	// Try JSON parse for structured output
	var parsed struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(lastLine(output)), &parsed); err == nil && parsed.Result != "" {
		output = parsed.Result
	}

	return &core.ExecResult{
		FinalMessage: output,
		Patches:      patches,
		Usage:        core.Usage{Turns: 1},
	}, nil
}

func (b *Backend) HealthCheck(ctx context.Context) error {
	return cli.CheckBinary(ctx, b.binary)
}

func (b *Backend) BuildArgs(req *core.ExecRequest) []string {
	args := []string{"-p", req.PromptPack.SpecBody, "--output-format", "json"}

	switch req.ApprovalProfile {
	case core.ProfilePlan:
		// read-only, no approval flags needed
	case core.ProfilePatch:
		args = append(args, "--approval-mode", "ask")
	case core.ProfileWorkspaceAuto, core.ProfileSandboxAuto:
		args = append(args, "--approval-mode", "auto")
	case core.ProfileFullTrust:
		args = append(args, "--yolo")
	}

	return args
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}
