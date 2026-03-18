// Package cursor implements the Cursor CLI backend adapter.
//
// Invocation: cursor-cli generate [flags] "<prompt>"
// Key flags:
//   --auto-run: automatically execute generated code
//
// NOTE: Cursor CLI structured output is not yet stable.
// This adapter uses text parsing with file block extraction.
package cursor

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
		binary = "cursor-cli"
	}
	return &Backend{binary: binary}
}

func (b *Backend) ID() string { return "cli:cursor" }

func (b *Backend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{
		StructuredOutput: false,
		StreamingEvents:  false,
		MCP:              false,
		Tools:            false,
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
	args := []string{"generate"}

	switch req.ApprovalProfile {
	case core.ProfilePlan:
		// no --auto-run
	case core.ProfilePatch:
		// no --auto-run
	case core.ProfileWorkspaceAuto, core.ProfileSandboxAuto, core.ProfileFullTrust:
		args = append(args, "--auto-run")
	}

	args = append(args, req.PromptPack.SpecBody)
	return args
}
