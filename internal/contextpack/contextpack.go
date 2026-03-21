// Package contextpack assembles the prompt context sent to AI backends.
//
// Instead of sending the entire repository to the backend, the context pack
// contains only what's relevant to the current build:
// - Current spec body
// - Imported spec summaries
// - Agent rules (AGENTS.md, CLAUDE.md, etc.)
// - Skills
// - File ownership map
// - Relevant source file contents
// - Failing test output
// - Coverage gap report
// - Policy and backend capability summary
package contextpack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

// ContextPack holds all context needed for a backend execution.
type ContextPack struct {
	// SpecBody is the full Markdown body of the current spec.
	SpecBody string

	// ImportedSpecs contains summaries of imported specs.
	ImportedSpecs []ImportedSpec

	// AgentRules contains content from AGENTS.md, CLAUDE.md, etc.
	AgentRules string

	// Skills lists available skill names.
	Skills []string

	// OwnedFiles lists files owned by the current spec.
	OwnedFiles []string

	// SharedFiles lists files shared with other specs.
	SharedFiles []string

	// ReadonlyFiles lists files that can be read but not modified.
	ReadonlyFiles []string

	// SourceContents maps file path to file content for relevant source files.
	SourceContents map[string]string

	// FailingTests contains output from recently failed tests.
	FailingTests string

	// CoverageGaps describes areas lacking test coverage.
	CoverageGaps string

	// PolicySummary describes the active approval profile.
	PolicySummary string

	// BackendCapabilities describes what the selected backend supports.
	BackendCapabilities string
}

// ImportedSpec is a summary of an imported spec.
type ImportedSpec struct {
	ID      string
	Purpose string
}

// Assembler builds context packs from specs and project state.
type Assembler struct {
	projectDir string
	specDir    string
}

// NewAssembler creates a context pack assembler.
func NewAssembler(projectDir, specDir string) *Assembler {
	return &Assembler{
		projectDir: projectDir,
		specDir:    specDir,
	}
}

// Assemble builds a ContextPack for the given spec.
func (a *Assembler) Assemble(spec *ast.Spec, importedSpecs map[string]*ast.Spec) (*ContextPack, error) {
	pack := &ContextPack{
		SpecBody:       spec.Body.Raw,
		OwnedFiles:     spec.ManagedFiles.Owned,
		SharedFiles:    spec.ManagedFiles.Shared,
		ReadonlyFiles:  spec.ManagedFiles.Readonly,
		SourceContents: make(map[string]string),
	}

	// Add imported spec summaries
	for id, imp := range importedSpecs {
		pack.ImportedSpecs = append(pack.ImportedSpecs, ImportedSpec{
			ID:      id,
			Purpose: imp.Body.Purpose,
		})
	}

	// Read owned source files
	for _, path := range spec.ManagedFiles.Owned {
		content, err := a.readFile(path)
		if err != nil {
			continue // file may not exist yet (first build)
		}
		pack.SourceContents[path] = content
	}

	// Read shared source files
	for _, path := range spec.ManagedFiles.Shared {
		content, err := a.readFile(path)
		if err != nil {
			continue
		}
		pack.SourceContents[path] = content
	}

	// Load agent rules
	pack.AgentRules = a.loadAgentRules()

	return pack, nil
}

// readFile reads a file relative to the project directory.
func (a *Assembler) readFile(rel string) (string, error) {
	data, err := os.ReadFile(filepath.Join(a.projectDir, rel))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loadAgentRules reads AGENTS.md, .claude/CLAUDE.md, etc.
func (a *Assembler) loadAgentRules() string {
	var parts []string

	candidates := []string{
		"AGENTS.md",
		".claude/CLAUDE.md",
		".cursor/rules",
	}

	for _, name := range candidates {
		content, err := a.readFile(name)
		if err == nil && content != "" {
			parts = append(parts, fmt.Sprintf("--- %s ---\n%s", name, content))
		}
	}

	return strings.Join(parts, "\n\n")
}

// ToPrompt converts the context pack to a formatted prompt string.
// This is a simple text representation; backends may use structured formats.
func (pack *ContextPack) ToPrompt() string {
	var b strings.Builder

	b.WriteString("# Spec\n\n")
	b.WriteString(pack.SpecBody)
	b.WriteString("\n\n")

	if len(pack.ImportedSpecs) > 0 {
		b.WriteString("# Imported Specs\n\n")
		for _, imp := range pack.ImportedSpecs {
			fmt.Fprintf(&b, "- %s: %s\n", imp.ID, imp.Purpose)
		}
		b.WriteString("\n")
	}

	if pack.AgentRules != "" {
		b.WriteString("# Project Rules\n\n")
		b.WriteString(pack.AgentRules)
		b.WriteString("\n\n")
	}

	if len(pack.Skills) > 0 {
		b.WriteString("# Skills\n\n")
		for _, skill := range pack.Skills {
			fmt.Fprintf(&b, "- %s\n", skill)
		}
		b.WriteString("\n")
	}

	if pack.PolicySummary != "" || pack.BackendCapabilities != "" {
		b.WriteString("# Execution Constraints\n\n")
		if pack.PolicySummary != "" {
			fmt.Fprintf(&b, "Policy: %s\n", pack.PolicySummary)
		}
		if pack.BackendCapabilities != "" {
			fmt.Fprintf(&b, "Backend: %s\n", pack.BackendCapabilities)
		}
		b.WriteString("\n")
	}

	b.WriteString("# File Ownership\n\n")
	fmt.Fprintf(&b, "Owned: %v\n", pack.OwnedFiles)
	fmt.Fprintf(&b, "Shared: %v\n", pack.SharedFiles)
	fmt.Fprintf(&b, "Readonly: %v\n\n", pack.ReadonlyFiles)

	if len(pack.SourceContents) > 0 {
		b.WriteString("# Current Source Files\n\n")
		paths := make([]string, 0, len(pack.SourceContents))
		for path := range pack.SourceContents {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			content := pack.SourceContents[path]
			fmt.Fprintf(&b, "## %s\n```\n%s\n```\n\n", path, content)
		}
	}

	if pack.FailingTests != "" {
		b.WriteString("# Failing Tests\n\n")
		b.WriteString(pack.FailingTests)
		b.WriteString("\n\n")
	}

	if pack.CoverageGaps != "" {
		b.WriteString("# Coverage Gaps\n\n")
		b.WriteString(pack.CoverageGaps)
		b.WriteString("\n\n")
	}

	return b.String()
}
