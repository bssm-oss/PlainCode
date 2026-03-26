// Package ast defines the Spec Abstract Syntax Tree — the primary
// representation of a parsed spec file.
//
// A Spec consists of:
//   - Structured YAML frontmatter (id, language, imports, ownership, backend, etc.)
//   - Markdown body with known sections (Purpose, Functional behavior, etc.)
//   - Computed metadata (hash, source path)
//
// This package is intentionally free of I/O. Parsing is done by spec/parser.
package ast

// Spec represents a fully parsed spec file with frontmatter and body.
type Spec struct {
	// --- Frontmatter fields ---

	// ID is the unique identifier for this spec, e.g. "billing/invoice-pdf".
	ID string `yaml:"id"`

	// Language is the target programming language, e.g. "go", "python", "typescript".
	Language string `yaml:"language"`

	// Imports lists other spec IDs that this spec depends on.
	Imports []string `yaml:"imports,omitempty"`

	// ManagedFiles defines the three-tier file ownership model.
	ManagedFiles ManagedFiles `yaml:"managed_files"`

	// Backend specifies preferred AI backends in priority order.
	Backend BackendPref `yaml:"backend"`

	// Approval is the autonomy profile name: plan, patch, workspace-auto, etc.
	Approval string `yaml:"approval"`

	// Tests specifies how to run tests for this spec's owned files.
	Tests TestConfig `yaml:"tests"`

	// Coverage specifies the target coverage percentage.
	Coverage CoverageCfg `yaml:"coverage"`

	// Budget constrains resource usage for a single build.
	Budget BudgetCfg `yaml:"budget"`

	// Runtime specifies how a built service should be started and managed.
	Runtime RuntimeConfig `yaml:"runtime,omitempty"`

	// --- Computed fields (not from YAML) ---

	// Hash is a SHA-256 truncated hash of the raw spec file content.
	// Used for dirty detection and receipt tracking.
	Hash string `yaml:"-"`

	// SourcePath is the filesystem path where this spec was loaded from.
	SourcePath string `yaml:"-"`

	// Body contains the parsed Markdown body sections.
	Body SpecBody `yaml:"-"`
}

// ManagedFiles defines the three-tier file ownership model.
// Each spec declares which files it owns, shares, and reads.
type ManagedFiles struct {
	// Owned files are exclusively managed by this spec.
	// No other spec may modify them.
	Owned []string `yaml:"owned,omitempty"`

	// Shared files may be modified by multiple specs.
	// Extra validation (lint, import check, conflict detection) applies.
	Shared []string `yaml:"shared,omitempty"`

	// Readonly files can be referenced but never modified.
	Readonly []string `yaml:"readonly,omitempty"`
}

// BackendPref lists preferred backends in priority order.
type BackendPref struct {
	Preferred []string `yaml:"preferred,omitempty"`
}

// TestConfig specifies how to run tests for this spec.
type TestConfig struct {
	Command string `yaml:"command"`
}

// CoverageCfg specifies the target coverage percentage (0.0–1.0).
type CoverageCfg struct {
	Target float64 `yaml:"target"`
}

// RuntimeConfig specifies how a built service should be managed locally.
type RuntimeConfig struct {
	// Mode is the preferred runtime launcher: auto, process, or docker.
	Mode string `yaml:"mode,omitempty"`

	// DefaultMode is the legacy alias kept for compatibility.
	DefaultMode string         `yaml:"default_mode,omitempty"`

	// HealthURL is a top-level service healthcheck URL.
	HealthURL string `yaml:"health_url,omitempty"`

	Process ProcessRuntime `yaml:"process,omitempty"`
	Docker  DockerRuntime  `yaml:"docker,omitempty"`
}

// ProcessRuntime defines direct process execution settings.
type ProcessRuntime struct {
	Command        string            `yaml:"command,omitempty"`
	Cwd            string            `yaml:"cwd,omitempty"`
	WorkingDir     string            `yaml:"working_dir,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	HealthcheckURL string            `yaml:"healthcheck_url,omitempty"`
}

// DockerRuntime defines Docker-based execution settings.
type DockerRuntime struct {
	Context        string            `yaml:"context,omitempty"`
	Dockerfile     string            `yaml:"dockerfile,omitempty"`
	Image          string            `yaml:"image,omitempty"`
	ContainerName  string            `yaml:"container_name,omitempty"`
	Ports          []string          `yaml:"ports,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	HealthcheckURL string            `yaml:"healthcheck_url,omitempty"`
}

// BudgetCfg constrains resource usage for a single build.
type BudgetCfg struct {
	MaxTurns   int     `yaml:"max_turns"`
	MaxCostUSD float64 `yaml:"max_cost_usd"`
}

// SpecBody holds the parsed Markdown body sections.
// Sections are extracted by heading: "## Purpose", "## Functional behavior", etc.
type SpecBody struct {
	Purpose            string
	FunctionalBehavior string
	InputsOutputs      string
	Invariants         string
	ErrorCases         string
	IntegrationPoints  string
	Observability      string
	TestOracles        string
	MigrationNotes     string
	Raw                string // full body markdown
}

// AllOwnedFiles returns the combined list of owned files.
func (s *Spec) AllOwnedFiles() []string {
	return s.ManagedFiles.Owned
}

// AllManagedPaths returns every path declared by this spec (owned + shared + readonly).
func (s *Spec) AllManagedPaths() []string {
	var paths []string
	paths = append(paths, s.ManagedFiles.Owned...)
	paths = append(paths, s.ManagedFiles.Shared...)
	paths = append(paths, s.ManagedFiles.Readonly...)
	return paths
}
