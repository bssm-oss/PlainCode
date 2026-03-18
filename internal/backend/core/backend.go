package core

import "context"

// Backend is the core abstraction over all AI providers (API and CLI).
type Backend interface {
	// ID returns a unique identifier like "openai:gpt-5" or "cli:claude".
	ID() string

	// Capabilities reports what this backend supports.
	Capabilities() CapabilitySet

	// Execute runs a code generation request and streams events to sink.
	Execute(ctx context.Context, req *ExecRequest, sink EventSink) (*ExecResult, error)

	// HealthCheck verifies the backend is reachable and configured.
	HealthCheck(ctx context.Context) error
}

// CapabilitySet describes what a backend can do.
type CapabilitySet struct {
	StructuredOutput bool
	StreamingEvents  bool
	MCP              bool
	Tools            bool
	CostTracking     bool
	FilePatchMode    bool
	JsonSchemaOutput bool
}

// ApprovalProfile controls the autonomy level of a backend execution.
type ApprovalProfile string

const (
	ProfilePlan         ApprovalProfile = "plan"
	ProfilePatch        ApprovalProfile = "patch"
	ProfileWorkspaceAuto ApprovalProfile = "workspace-auto"
	ProfileSandboxAuto  ApprovalProfile = "sandbox-auto"
	ProfileFullTrust    ApprovalProfile = "full-trust"
)

// Budget constrains a single backend execution.
type Budget struct {
	MaxTurns  int
	MaxCostUSD float64
}

// ExecRequest is the input to Backend.Execute.
type ExecRequest struct {
	BuildID         string
	PromptPack      PromptPack
	WorkspaceDir    string
	ApprovalProfile ApprovalProfile
	Budget          Budget
	OutputSchema    []byte // optional JSON schema for structured output
	Env             map[string]string
}

// PromptPack holds the assembled context for the backend.
type PromptPack struct {
	SpecBody         string
	ImportedSpecs    []string
	AgentsRules      string
	Skills           []string
	OwnedFiles       []string
	SharedFiles      []string
	ReadonlyFiles    []string
	SourceSummaries  map[string]string
	FailingTests     []string
	CoverageGaps     []string
	PolicySummary    string
	BackendCapSummary string
}

// ExecResult is the output from Backend.Execute.
type ExecResult struct {
	FinalMessage string
	Patches      []PatchOp
	Files        []GeneratedFile
	Usage        Usage
	Metadata     map[string]any
}

// GeneratedFile is a complete file produced by the backend.
type GeneratedFile struct {
	Path    string
	Content []byte
}

// Usage tracks resource consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	DurationMS   int64
	Turns        int
}

// EventSink receives streaming events during execution.
type EventSink interface {
	OnEvent(event Event)
}

// Event is a streaming event from a backend.
type Event struct {
	Type    string
	Payload any
}
