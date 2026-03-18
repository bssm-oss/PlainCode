package receipt

import "time"

// Receipt captures the full audit trail of a single build.
type Receipt struct {
	BuildID          string            `json:"build_id"`
	SpecID           string            `json:"spec_id"`
	SpecHash         string            `json:"spec_hash"`
	ImportedSpecHashes map[string]string `json:"imported_spec_hashes,omitempty"`

	BackendID        string            `json:"backend_id"`
	Model            string            `json:"model,omitempty"`
	CLIVersion       string            `json:"cli_version,omitempty"`
	ApprovalProfile  string            `json:"approval_profile"`

	MCPServers       []string          `json:"mcp_servers,omitempty"`
	SkillsUsed       []string          `json:"skills_used,omitempty"`

	ChangedFiles     []string          `json:"changed_files"`
	TestsRun         int               `json:"tests_run"`
	TestsPassed      int               `json:"tests_passed"`
	TestsFailed      int               `json:"tests_failed"`

	CoverageBefore   float64           `json:"coverage_before"`
	CoverageAfter    float64           `json:"coverage_after"`

	Retries          int               `json:"retries"`
	InputTokens      int               `json:"input_tokens"`
	OutputTokens     int               `json:"output_tokens"`
	CostUSD          float64           `json:"cost_usd"`
	DurationMS       int64             `json:"duration_ms"`

	Status           string            `json:"status"` // success, failed, partial
	Error            string            `json:"error,omitempty"`

	StartedAt        time.Time         `json:"started_at"`
	CompletedAt      time.Time         `json:"completed_at"`
}
