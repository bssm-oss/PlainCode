// Package parser implements a spec file parser that reads Markdown files
// with YAML frontmatter and produces a structured Spec AST.
//
// Spec files use the format:
//
//	---
//	id: billing/invoice-pdf
//	language: go
//	...frontmatter fields...
//	---
//	# Purpose
//	...markdown body...
//
// The parser performs strict frontmatter validation, rejecting unknown fields,
// and extracts structured body sections by heading.
package parser

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
	"gopkg.in/yaml.v3"
)

// Parse reads a spec file from raw bytes and returns a Spec AST.
// It validates frontmatter strictly and extracts body sections.
func Parse(data []byte) (*ast.Spec, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("splitting frontmatter: %w", err)
	}

	spec, err := parseFrontmatter(fm)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	spec.Body = parseBody(body)
	spec.Body.Raw = body

	// Compute spec hash for deterministic tracking
	h := sha256.Sum256(data)
	spec.Hash = fmt.Sprintf("%x", h[:16])

	if err := validate(spec); err != nil {
		return nil, fmt.Errorf("validating spec: %w", err)
	}

	return spec, nil
}

// splitFrontmatter separates YAML frontmatter from Markdown body.
// Frontmatter must be delimited by "---" lines.
func splitFrontmatter(data []byte) (frontmatter []byte, body string, err error) {
	content := string(data)

	if !strings.HasPrefix(content, "---") {
		return nil, "", fmt.Errorf("spec must start with '---' frontmatter delimiter")
	}

	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, "", fmt.Errorf("no closing '---' frontmatter delimiter found")
	}

	fm := rest[:idx]
	body = strings.TrimLeft(rest[idx+4:], "\n")

	return []byte(fm), body, nil
}

// parseFrontmatter decodes YAML frontmatter into Spec struct with strict validation.
func parseFrontmatter(data []byte) (*ast.Spec, error) {
	var spec ast.Spec

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // Reject unknown fields

	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("yaml decode (unknown fields will be rejected): %w", err)
	}

	return &spec, nil
}

// parseBody extracts known sections from markdown body by heading.
func parseBody(body string) ast.SpecBody {
	sections := extractSections(body)

	return ast.SpecBody{
		Purpose:            sections["purpose"],
		FunctionalBehavior: sections["functional behavior"],
		InputsOutputs:      sections["inputs / outputs"],
		Invariants:         sections["invariants"],
		ErrorCases:         sections["error cases"],
		IntegrationPoints:  sections["integration points"],
		Observability:      sections["observability"],
		TestOracles:        sections["test oracles"],
		MigrationNotes:     sections["migration notes"],
	}
}

// extractSections splits markdown by ## headings into a map keyed by lowercase heading text.
func extractSections(body string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(body, "\n")

	var currentKey string
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
			if currentKey != "" {
				sections[currentKey] = strings.TrimSpace(strings.Join(currentLines, "\n"))
			}
			heading := trimmed
			heading = strings.TrimPrefix(heading, "## ")
			heading = strings.TrimPrefix(heading, "# ")
			currentKey = strings.ToLower(strings.TrimSpace(heading))
			currentLines = nil
		} else {
			currentLines = append(currentLines, line)
		}
	}

	if currentKey != "" {
		sections[currentKey] = strings.TrimSpace(strings.Join(currentLines, "\n"))
	}

	return sections
}

// validate performs semantic validation on a parsed spec.
func validate(spec *ast.Spec) error {
	if spec.ID == "" {
		return fmt.Errorf("spec id is required")
	}
	if spec.Language == "" {
		return fmt.Errorf("spec language is required")
	}
	mode := spec.Runtime.Mode
	if mode == "" {
		mode = spec.Runtime.DefaultMode
	}
	switch mode {
	case "", "auto", "process", "docker":
	default:
		return fmt.Errorf("runtime.mode/default_mode must be one of auto, process, docker")
	}
	if mode == "process" && spec.Runtime.Process.Command == "" && spec.Language != "go" {
		return fmt.Errorf("runtime.process.command is required for non-go process runtimes")
	}
	return nil
}

// ParseFile reads and parses a spec file from the filesystem.
func ParseFile(path string) (*ast.Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file %s: %w", path, err)
	}
	spec, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing spec file %s: %w", path, err)
	}
	spec.SourcePath = path
	return spec, nil
}
