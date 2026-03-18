// Package copilot implements the GitHub Copilot CLI backend adapter.
//
// Invocation: copilot -p "<prompt>" [flags]
// Key flags:
//   -p: pass prompt
//   --allow-tool <tool>: allow specific tool
//   --allow-all-tools: allow all tools
//   --allow-all / --yolo: full autonomy
//   --no-ask-user: non-interactive mode
//   --model <model>: model selection
//   -s: minimize session metadata
package copilot

import (
	"context"

	"github.com/bssm-oss/PlainCode/internal/backend/cli"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

type Backend struct {
	binary string
	model  string
}

func New(binary string) *Backend {
	if binary == "" {
		binary = "copilot"
	}
	return &Backend{binary: binary}
}

// WithModel sets the model for Copilot CLI.
func (b *Backend) WithModel(model string) *Backend {
	b.model = model
	return b
}

func (b *Backend) ID() string { return "cli:copilot" }

func (b *Backend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{
		StructuredOutput: false, // text output, parsed via file blocks
		StreamingEvents:  false,
		MCP:              true,
		Tools:            true,
		CostTracking:     false,
		FilePatchMode:    false,
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

	patches := cli.ParseFileBlocks(result.Stdout)

	return &core.ExecResult{
		FinalMessage: result.Stdout,
		Patches:      patches,
		Usage:        core.Usage{Turns: 1},
	}, nil
}

func (b *Backend) HealthCheck(ctx context.Context) error {
	return cli.CheckBinary(ctx, b.binary)
}

func (b *Backend) BuildArgs(req *core.ExecRequest) []string {
	args := []string{"-p", req.PromptPack.SpecBody}

	if b.model != "" {
		args = append(args, "--model", b.model)
	}

	switch req.ApprovalProfile {
	case core.ProfilePlan:
		// default mode — no auto-approve
	case core.ProfilePatch:
		args = append(args, "--allow-tool", "read_file", "--allow-tool", "write_file")
	case core.ProfileWorkspaceAuto:
		args = append(args, "--allow-all-tools", "--no-ask-user")
	case core.ProfileSandboxAuto:
		args = append(args, "--allow-all-tools", "--no-ask-user")
	case core.ProfileFullTrust:
		args = append(args, "--allow-all", "--yolo")
	}

	// Minimize session metadata for automation
	args = append(args, "-s")

	return args
}
