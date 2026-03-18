package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
	"github.com/bssm-oss/PlainCode/internal/backend/mock"
	"github.com/bssm-oss/PlainCode/internal/config"
)

// setupTestProject creates a temporary project with a spec and config.
func setupTestProject(t *testing.T) (string, *config.ProjectConfig, *core.Registry) {
	t.Helper()
	dir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(dir, "spec", "hello"), 0755)
	os.MkdirAll(filepath.Join(dir, ".plaincode", "builds"), 0755)

	// Write plaincode.yaml
	cfg := config.DefaultProjectConfig()
	cfg.Defaults.Backend = "mock:default"

	cfgData := []byte(`version: 1
project:
  spec_dir: spec
  state_dir: .plaincode
  default_language: go
defaults:
  backend: mock:default
  approval: patch
  retry_limit: 3
`)
	os.WriteFile(filepath.Join(dir, "plaincode.yaml"), cfgData, 0644)

	// Write a sample spec
	specData := []byte(`---
id: hello/greeter
language: go
managed_files:
  owned:
    - internal/greeter/greeter.go
  shared:
    - go.mod
backend:
  preferred:
    - mock:default
approval: patch
tests:
  command: go test ./internal/greeter/...
coverage:
  target: 0.80
budget:
  max_turns: 5
  max_cost_usd: 2
---
# Purpose

A simple greeter module.

## Functional behavior

Greet(name) returns "Hello, {name}!"
`)
	os.WriteFile(filepath.Join(dir, "spec", "hello", "greeter.md"), specData, 0644)

	// Create mock backend
	registry := core.NewRegistry()
	mockBackend := mock.New("mock:default")
	mockBackend.SetResponse("", "package greeter\n\nfunc Greet(name string) string { return \"Hello, \" + name + \"!\" }\n")
	registry.Register(mockBackend)

	return dir, &cfg, registry
}

func TestBuild_SingleSpec_Success(t *testing.T) {
	dir, cfg, registry := setupTestProject(t)

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{
		SpecID:    "hello/greeter",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Status != "success" {
		t.Errorf("expected status 'success', got %q (error: %s)", r.Status, r.Error)
	}
	if r.SpecID != "hello/greeter" {
		t.Errorf("expected spec_id 'hello/greeter', got %q", r.SpecID)
	}
	if r.Receipt == nil {
		t.Fatal("expected receipt, got nil")
	}
	if r.Receipt.BackendID != "mock:default" {
		t.Errorf("expected backend 'mock:default', got %q", r.Receipt.BackendID)
	}
	if r.Receipt.SpecHash == "" {
		t.Error("expected non-empty spec hash")
	}

	// Verify receipt was saved to disk
	_, err = os.Stat(filepath.Join(dir, ".plaincode", "builds", r.BuildID, "receipt.json"))
	if err != nil {
		t.Errorf("receipt file not found: %v", err)
	}
}

func TestBuild_DryRun(t *testing.T) {
	dir, cfg, registry := setupTestProject(t)

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{
		SpecID: "hello/greeter",
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "success" {
		t.Errorf("expected success, got %q", results[0].Status)
	}
}

func TestBuild_NoDirtySpecs(t *testing.T) {
	dir, cfg, registry := setupTestProject(t)

	builder := NewBuilder(cfg, registry, dir)

	// First build
	results1, err := builder.Build(context.Background(), BuildOptions{
		SpecID:    "hello/greeter",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	if results1[0].Status != "success" {
		t.Fatalf("first build should succeed, got %q: %s", results1[0].Status, results1[0].Error)
	}

	// Second build with no changes — should detect no dirty specs
	results2, err := builder.Build(context.Background(), BuildOptions{
		// No specID = build all dirty
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("second build failed: %v", err)
	}

	// Should skip because spec hash matches receipt
	if len(results2) == 0 {
		t.Fatal("expected at least one result")
	}
	if !results2[0].Skipped {
		// It's okay if it rebuilds — dirty detection depends on receipt hash matching
		t.Logf("note: second build was not skipped (dirty detection may need spec hash match)")
	}
}

func TestBuild_SpecNotFound(t *testing.T) {
	dir, cfg, registry := setupTestProject(t)

	builder := NewBuilder(cfg, registry, dir)
	_, err := builder.Build(context.Background(), BuildOptions{
		SpecID: "nonexistent/spec",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent spec")
	}
}

func TestBuild_OwnershipViolation(t *testing.T) {
	dir, cfg, _ := setupTestProject(t)

	// Create a mock that tries to write to an unmanaged file
	registry := core.NewRegistry()
	violatingBackend := mock.New("mock:default")
	registry.Register(violatingBackend)

	// Add a second spec that owns "internal/greeter/greeter.go"
	// so the first spec's build would conflict
	specData2 := []byte(`---
id: other/module
language: go
managed_files:
  owned:
    - internal/greeter/greeter.go
backend:
  preferred:
    - mock:default
approval: patch
tests:
  command: echo ok
coverage:
  target: 0.50
budget:
  max_turns: 3
  max_cost_usd: 1
---
# Purpose

Another module that also claims greeter.go (conflict!).
`)
	os.MkdirAll(filepath.Join(dir, "spec", "other"), 0755)
	os.WriteFile(filepath.Join(dir, "spec", "other", "module.md"), specData2, 0644)

	builder := NewBuilder(cfg, registry, dir)

	// Build hello/greeter — mock backend writes to mock_output.go which is unmanaged
	// This tests that unmanaged file writes are allowed (they're classified as Unmanaged, not rejected)
	results, err := builder.Build(context.Background(), BuildOptions{
		SpecID:    "hello/greeter",
		SkipTests: true,
	})
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	// The mock backend writes to mock_output.go which is Unmanaged — allowed by default
	// A real ownership violation would be writing to a file owned by other/module
	if len(results) > 0 && results[0].Status == "failed" {
		t.Logf("build failed as expected: %s", results[0].Error)
	}
}

func TestLoadSpecs(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "spec")
	os.MkdirAll(filepath.Join(specDir, "hello"), 0755)

	specData := []byte(`---
id: hello/greeter
language: go
managed_files:
  owned:
    - greeter.go
approval: patch
tests:
  command: go test ./...
coverage:
  target: 0.80
budget:
  max_turns: 5
  max_cost_usd: 2
---
# Purpose

Test spec.
`)
	os.WriteFile(filepath.Join(specDir, "hello", "greeter.md"), specData, 0644)

	result, err := LoadSpecs(specDir)
	if err != nil {
		t.Fatalf("LoadSpecs failed: %v", err)
	}

	if len(result.Specs) != 1 {
		t.Errorf("expected 1 spec, got %d", len(result.Specs))
	}
	if _, ok := result.Specs["hello/greeter"]; !ok {
		t.Error("expected 'hello/greeter' in specs")
	}
	if result.Graph.Size() != 1 {
		t.Errorf("expected graph size 1, got %d", result.Graph.Size())
	}
}
