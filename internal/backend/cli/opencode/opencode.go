// Package opencode implements the OpenCode CLI backend adapter.
//
// Invocation: opencode generate [flags] "<prompt>"
// Key flags:
//   --auto: automated mode
//   --dry-run: plan only, no modifications
//
// OpenCode also supports a server mode (OpenAPI 3.1 + SSE) which
// can be used as a provider aggregation backend. This adapter
// implements the CLI driver only; server driver is planned.
package opencode

import (
	"context"

	"github.com/bssm-oss/PlainCode/internal/backend/cli"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

type Backend struct {
	binary string
}

func New(binary string) *Backend {
	if binary == "" {
		binary = "opencode"
	}
	return &Backend{binary: binary}
}

func (b *Backend) ID() string { return "cli:opencode" }

func (b *Backend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{
		StructuredOutput: true,
		StreamingEvents:  false,
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
	args := []string{"generate"}

	switch req.ApprovalProfile {
	case core.ProfilePlan:
		args = append(args, "--dry-run")
	case core.ProfilePatch:
		// default interactive mode
	case core.ProfileWorkspaceAuto, core.ProfileSandboxAuto, core.ProfileFullTrust:
		args = append(args, "--auto")
	}

	args = append(args, req.PromptPack.SpecBody)
	return args
}
