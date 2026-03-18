// Package mock provides a test/development backend that returns
// predefined responses without calling any AI service.
//
// This is useful for:
//   - Running the build pipeline without API keys
//   - Testing the orchestration logic
//   - Understanding the system flow
//   - CI environments without AI access
package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

// Backend is a mock AI backend for testing.
type Backend struct {
	id        string
	responses map[string]string // spec ID -> response content
	delay     time.Duration
}

// New creates a mock backend.
func New(id string) *Backend {
	return &Backend{
		id:        id,
		responses: make(map[string]string),
	}
}

// WithDelay adds artificial latency to simulate real backends.
func (b *Backend) WithDelay(d time.Duration) *Backend {
	b.delay = d
	return b
}

// SetResponse sets a canned response for a spec ID.
func (b *Backend) SetResponse(specID, content string) {
	b.responses[specID] = content
}

// ID returns the backend identifier.
func (b *Backend) ID() string { return b.id }

// Capabilities returns the mock backend's capabilities.
func (b *Backend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{
		StructuredOutput: true,
		StreamingEvents:  false,
		MCP:              false,
		Tools:            false,
		CostTracking:     true,
		FilePatchMode:    true,
		JsonSchemaOutput: true,
	}
}

// Execute runs the mock backend, returning a predefined response.
func (b *Backend) Execute(ctx context.Context, req *core.ExecRequest, sink core.EventSink) (*core.ExecResult, error) {
	if b.delay > 0 {
		select {
		case <-time.After(b.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if sink != nil {
		sink.OnEvent(core.Event{Type: "mock_start", Payload: req.BuildID})
	}

	// Look up canned response or generate a default
	content, ok := b.responses[req.BuildID]
	if !ok {
		content = fmt.Sprintf("// Mock generated code for build %s\n// Spec: %s\npackage mock\n",
			req.BuildID, req.PromptPack.SpecBody[:min(50, len(req.PromptPack.SpecBody))])
	}

	result := &core.ExecResult{
		FinalMessage: "Mock backend completed successfully",
		Patches: []core.PatchOp{
			core.WriteFile{
				FilePath: "mock_output.go",
				Content:  []byte(content),
			},
		},
		Usage: core.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			CostUSD:      0.001,
			DurationMS:   int64(b.delay.Milliseconds()),
			Turns:        1,
		},
		Metadata: map[string]any{
			"backend": "mock",
			"note":    "This is a test/development backend",
		},
	}

	if sink != nil {
		sink.OnEvent(core.Event{Type: "mock_complete", Payload: result})
	}

	return result, nil
}

// HealthCheck always returns nil for mock backend.
func (b *Backend) HealthCheck(_ context.Context) error {
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
