package parser

import (
	"testing"
)

const validSpec = `---
id: billing/invoice-pdf
language: go
imports:
  - billing/shared/money
managed_files:
  owned:
    - internal/billing/invoice_pdf.go
  shared:
    - go.mod
  readonly:
    - internal/legacy/**
backend:
  preferred:
    - openai:gpt-5
approval: workspace-auto
tests:
  command: go test ./...
coverage:
  target: 0.85
budget:
  max_turns: 12
  max_cost_usd: 5
runtime:
  default_mode: process
  process:
    command: go run .
    working_dir: .
    healthcheck_url: http://127.0.0.1:8080/health
---
# Purpose

Generate PDF invoices for paid orders.

## Functional behavior

Converts order data into formatted PDF documents.

## Invariants

All invoices must have a non-zero total.
`

func TestParse_ValidSpec(t *testing.T) {
	spec, err := Parse([]byte(validSpec))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.ID != "billing/invoice-pdf" {
		t.Errorf("expected id 'billing/invoice-pdf', got %q", spec.ID)
	}
	if spec.Language != "go" {
		t.Errorf("expected language 'go', got %q", spec.Language)
	}
	if len(spec.Imports) != 1 || spec.Imports[0] != "billing/shared/money" {
		t.Errorf("unexpected imports: %v", spec.Imports)
	}
	if len(spec.ManagedFiles.Owned) != 1 {
		t.Errorf("expected 1 owned file, got %d", len(spec.ManagedFiles.Owned))
	}
	if spec.Approval != "workspace-auto" {
		t.Errorf("expected approval 'workspace-auto', got %q", spec.Approval)
	}
	if spec.Coverage.Target != 0.85 {
		t.Errorf("expected coverage target 0.85, got %f", spec.Coverage.Target)
	}
	if spec.Budget.MaxTurns != 12 {
		t.Errorf("expected max_turns 12, got %d", spec.Budget.MaxTurns)
	}
	if spec.Runtime.DefaultMode != "process" {
		t.Errorf("expected runtime default mode process, got %q", spec.Runtime.DefaultMode)
	}
	if spec.Runtime.Process.Command != "go run ." {
		t.Errorf("expected runtime command, got %q", spec.Runtime.Process.Command)
	}
	if spec.Runtime.Process.WorkingDir != "." {
		t.Errorf("expected runtime working dir '.', got %q", spec.Runtime.Process.WorkingDir)
	}
	if spec.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if spec.Body.Purpose == "" {
		t.Error("expected non-empty Purpose section")
	}
	if spec.Body.FunctionalBehavior == "" {
		t.Error("expected non-empty Functional behavior section")
	}
}

func TestParse_MissingID(t *testing.T) {
	data := []byte(`---
language: go
---
# Purpose
Test
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestParse_MissingFrontmatter(t *testing.T) {
	data := []byte(`# Just markdown
No frontmatter here.
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParse_UnknownField(t *testing.T) {
	data := []byte(`---
id: test/spec
language: go
unknown_field: bad
---
# Purpose
Test
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for unknown frontmatter field")
	}
}

func TestParse_InvalidRuntimeMode(t *testing.T) {
	data := []byte(`---
id: test/spec
language: go
runtime:
  default_mode: invalid
---
# Purpose
Test
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid runtime mode")
	}
}

func TestSplitFrontmatter(t *testing.T) {
	data := []byte("---\nid: test\n---\n# Body\nContent")
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(fm) != "\nid: test" {
		t.Errorf("unexpected frontmatter: %q", string(fm))
	}
	if body != "# Body\nContent" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestExtractSections(t *testing.T) {
	body := "## Purpose\nDo things.\n\n## Error cases\nNone.\n"
	sections := extractSections(body)

	if sections["purpose"] != "Do things." {
		t.Errorf("unexpected purpose: %q", sections["purpose"])
	}
	if sections["error cases"] != "None." {
		t.Errorf("unexpected error cases: %q", sections["error cases"])
	}
}
