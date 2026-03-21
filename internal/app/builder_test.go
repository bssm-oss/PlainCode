package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
	"github.com/bssm-oss/PlainCode/internal/config"
)

type sequenceBackend struct {
	id      string
	results []*core.ExecResult
	calls   int
	prompts []string
}

func (b *sequenceBackend) ID() string { return b.id }

func (b *sequenceBackend) Capabilities() core.CapabilitySet {
	return core.CapabilitySet{StructuredOutput: true, FilePatchMode: true}
}

func (b *sequenceBackend) Execute(_ context.Context, req *core.ExecRequest, _ core.EventSink) (*core.ExecResult, error) {
	b.prompts = append(b.prompts, req.PromptPack.SpecBody)
	index := b.calls
	if index >= len(b.results) {
		index = len(b.results) - 1
	}
	b.calls++
	result := *b.results[index]
	return &result, nil
}

func (b *sequenceBackend) HealthCheck(context.Context) error { return nil }

func execResult(files map[string]string) *core.ExecResult {
	var ops []core.PatchOp
	for path, content := range files {
		ops = append(ops, core.WriteFile{
			FilePath: path,
			Content:  []byte(content),
		})
	}
	return &core.ExecResult{
		Patches: ops,
		Usage: core.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			CostUSD:      0.1,
			Turns:        1,
		},
	}
}

func setupProject(t *testing.T, specText string) (string, *config.ProjectConfig, *core.Registry) {
	t.Helper()

	dir := t.TempDir()
	mustMkdirAll(t, filepath.Join(dir, "spec", "hello"))
	mustMkdirAll(t, filepath.Join(dir, ".plaincode", "builds"))

	cfgData := []byte(`version: 1
project:
  spec_dir: spec
  state_dir: .plaincode
  default_language: go
defaults:
  backend: test:backend
  approval: patch
  retry_limit: 1
`)
	mustWriteFile(t, filepath.Join(dir, "plaincode.yaml"), cfgData)
	mustWriteFile(t, filepath.Join(dir, "spec", "hello", "greeter.md"), []byte(specText))

	cfg := config.DefaultProjectConfig()
	cfg.Defaults.Backend = "test:backend"
	cfg.Defaults.RetryLimit = 1

	registry := core.NewRegistry()
	return dir, &cfg, registry
}

func makeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	mustWriteFile(t, path, []byte("#!/bin/sh\nset -eu\n"+body+"\n"))
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
	return path
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func sampleGoSpec(testCommand string, coverageTarget string, owned ...string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("id: hello/greeter\n")
	b.WriteString("language: go\n")
	b.WriteString("managed_files:\n")
	b.WriteString("  owned:\n")
	for _, path := range owned {
		b.WriteString("    - " + path + "\n")
	}
	b.WriteString("backend:\n")
	b.WriteString("  preferred:\n")
	b.WriteString("    - test:backend\n")
	b.WriteString("approval: patch\n")
	b.WriteString("tests:\n")
	b.WriteString("  command: " + testCommand + "\n")
	b.WriteString("coverage:\n")
	b.WriteString("  target: " + coverageTarget + "\n")
	b.WriteString("budget:\n")
	b.WriteString("  max_turns: 3\n")
	b.WriteString("  max_cost_usd: 1\n")
	b.WriteString("---\n")
	b.WriteString("# Purpose\n\nBuild a greeter.\n\n")
	b.WriteString("## Functional behavior\n\nReturn deterministic greetings.\n")
	return b.String()
}

func TestBuild_SingleSpec_Success(t *testing.T) {
	specText := sampleGoSpec("go test ./...", "0.50",
		"go.mod",
		"internal/greeter/greeter.go",
		"internal/greeter/greeter_test.go",
	)
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{
		id: "test:backend",
		results: []*core.ExecResult{execResult(map[string]string{
			"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
			"internal/greeter/greeter.go": `package greeter

func Greet(name string) string {
	if name == "" {
		return "Hello, World!"
	}
	return "Hello, " + name + "!"
}
`,
			"internal/greeter/greeter_test.go": `package greeter

import "testing"

func TestGreet(t *testing.T) {
	if got := Greet("Alice"); got != "Hello, Alice!" {
		t.Fatalf("unexpected greeting: %s", got)
	}
	if got := Greet(""); got != "Hello, World!" {
		t.Fatalf("unexpected empty greeting: %s", got)
	}
}
`,
		})},
	}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{SpecID: "hello/greeter"})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.Status != "success" {
		t.Fatalf("expected success, got %q (%s)", got.Status, got.Error)
	}
	if got.Receipt == nil {
		t.Fatal("expected receipt")
	}
	if got.Receipt.TestsRun != 1 || got.Receipt.TestsPassed != 1 || got.Receipt.TestsFailed != 0 {
		t.Fatalf("unexpected test counts: %+v", got.Receipt)
	}
	if got.Receipt.CoverageAfter < 0.5 {
		t.Fatalf("expected coverage >= 0.5, got %f", got.Receipt.CoverageAfter)
	}

	for _, artifact := range []string{
		filepath.Join(dir, ".plaincode", "builds", got.BuildID, "receipt.json"),
		filepath.Join(dir, ".plaincode", "builds", got.BuildID, "tests.json"),
		filepath.Join(dir, ".plaincode", "builds", got.BuildID, "coverage.json"),
	} {
		if _, err := os.Stat(artifact); err != nil {
			t.Fatalf("expected artifact %s: %v", artifact, err)
		}
	}
}

func TestBuild_DryRun(t *testing.T) {
	specText := sampleGoSpec("/bin/echo ok", "0.0", "owned.txt")
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{id: "test:backend", results: []*core.ExecResult{execResult(map[string]string{
		"owned.txt": "updated\n",
	})}}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{SpecID: "hello/greeter", DryRun: true})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if results[0].Status != "success" || results[0].SkipReason != "dry-run" {
		t.Fatalf("unexpected dry-run result: %+v", results[0])
	}
	if backend.calls != 0 {
		t.Fatalf("expected backend not to run, got %d calls", backend.calls)
	}
}

func TestBuild_NoDirtySpecs(t *testing.T) {
	specText := sampleGoSpec("go test ./...", "0.50",
		"go.mod",
		"internal/greeter/greeter.go",
		"internal/greeter/greeter_test.go",
	)
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{
		id: "test:backend",
		results: []*core.ExecResult{execResult(map[string]string{
			"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
			"internal/greeter/greeter.go": `package greeter
func Greet(name string) string { return "Hello, " + name + "!" }
`,
			"internal/greeter/greeter_test.go": `package greeter
import "testing"
func TestGreet(t *testing.T) { if Greet("Alice") != "Hello, Alice!" { t.Fail() } }
`,
		})},
	}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	first, err := builder.Build(context.Background(), BuildOptions{SpecID: "hello/greeter"})
	if err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	if first[0].Status != "success" {
		t.Fatalf("unexpected first result: %+v", first[0])
	}

	second, err := builder.Build(context.Background(), BuildOptions{})
	if err != nil {
		t.Fatalf("second build failed: %v", err)
	}
	if !second[0].Skipped || second[0].SkipReason != "no dirty specs" {
		t.Fatalf("expected dirty skip, got %+v", second[0])
	}
}

func TestBuild_RetryOnTestFailureAndSuccess(t *testing.T) {
	specText := sampleGoSpec("go test ./...", "0.50",
		"go.mod",
		"internal/greeter/greeter.go",
		"internal/greeter/greeter_test.go",
	)
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{
		id: "test:backend",
		results: []*core.ExecResult{
			execResult(map[string]string{
				"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
				"internal/greeter/greeter.go": `package greeter
func Greet(name string) string { return "wrong" }
`,
				"internal/greeter/greeter_test.go": `package greeter
import "testing"
func TestGreet(t *testing.T) { if Greet("Alice") != "Hello, Alice!" { t.Fatalf("wrong greeting") } }
`,
			}),
			execResult(map[string]string{
				"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
				"internal/greeter/greeter.go": `package greeter
func Greet(name string) string { return "Hello, " + name + "!" }
`,
				"internal/greeter/greeter_test.go": `package greeter
import "testing"
func TestGreet(t *testing.T) { if Greet("Alice") != "Hello, Alice!" { t.Fatalf("wrong greeting") } }
`,
			}),
		},
	}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{SpecID: "hello/greeter", MaxRetries: 1})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if results[0].Status != "success" {
		t.Fatalf("expected success, got %+v", results[0])
	}
	if backend.calls != 2 {
		t.Fatalf("expected 2 backend calls, got %d", backend.calls)
	}
	if results[0].Receipt == nil || results[0].Receipt.Retries != 1 {
		t.Fatalf("expected 1 retry, got %+v", results[0].Receipt)
	}
	if !strings.Contains(backend.prompts[1], "Repair Attempt 2/2") {
		t.Fatalf("expected repair prompt in second attempt, got %q", backend.prompts[1])
	}
	if _, err := os.Stat(filepath.Join(dir, ".plaincode", "builds", results[0].BuildID, "repair.json")); err != nil {
		t.Fatalf("expected repair artifact: %v", err)
	}
}

func TestBuild_RestoresWorkspaceAfterExhaustedRetries(t *testing.T) {
	specText := sampleGoSpec("", "0.0", "owned.txt")
	dir, cfg, registry := setupProject(t, specText)

	failScript := makeScript(t, dir, "fail-tests.sh", "exit 1")
	specText = sampleGoSpec(failScript, "0.0", "owned.txt")
	mustWriteFile(t, filepath.Join(dir, "spec", "hello", "greeter.md"), []byte(specText))
	mustWriteFile(t, filepath.Join(dir, "owned.txt"), []byte("baseline\n"))

	backend := &sequenceBackend{
		id:      "test:backend",
		results: []*core.ExecResult{execResult(map[string]string{"owned.txt": "changed\n"})},
	}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{SpecID: "hello/greeter", MaxRetries: 1})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if results[0].Status != "failed" {
		t.Fatalf("expected failure, got %+v", results[0])
	}

	data, err := os.ReadFile(filepath.Join(dir, "owned.txt"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != "baseline\n" {
		t.Fatalf("expected baseline content, got %q", string(data))
	}
}

func TestBuild_ReconcilesOwnedFiles(t *testing.T) {
	specText := sampleGoSpec("", "0.0", "go.mod", "main.go", "obsolete.txt")
	dir, cfg, registry := setupProject(t, specText)
	mustWriteFile(t, filepath.Join(dir, "obsolete.txt"), []byte("stale\n"))

	backend := &sequenceBackend{
		id: "test:backend",
		results: []*core.ExecResult{execResult(map[string]string{
			"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
			"main.go": `package main
func main() {}
`,
		})},
	}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{
		SpecID:       "hello/greeter",
		SkipTests:    true,
		SkipCoverage: true,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if results[0].Status != "success" {
		t.Fatalf("expected success, got %+v", results[0])
	}
	if _, err := os.Stat(filepath.Join(dir, "obsolete.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected obsolete.txt to be deleted, err=%v", err)
	}
}

func TestBuild_RetryOnCoverageGap(t *testing.T) {
	specText := sampleGoSpec("go test ./...", "1.0",
		"go.mod",
		"main.go",
		"main_test.go",
	)
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{
		id: "test:backend",
		results: []*core.ExecResult{
			execResult(map[string]string{
				"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
				"main.go": `package main

func used() int { return 1 }
func unused() int { return 2 }
`,
				"main_test.go": `package main

import "testing"

func TestUsed(t *testing.T) {
	if used() != 1 {
		t.Fatal("bad value")
	}
}
`,
			}),
			execResult(map[string]string{
				"go.mod": "module example.com/plaincode-test\n\ngo 1.23\n",
				"main.go": `package main

func used() int { return 1 }
`,
				"main_test.go": `package main

import "testing"

func TestUsed(t *testing.T) {
	if used() != 1 {
		t.Fatal("bad value")
	}
}
`,
			}),
		},
	}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{SpecID: "hello/greeter", MaxRetries: 1})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if results[0].Status != "success" {
		t.Fatalf("expected success, got %+v", results[0])
	}
	if backend.calls != 2 {
		t.Fatalf("expected 2 backend calls, got %d", backend.calls)
	}
	if results[0].Receipt == nil || results[0].Receipt.CoverageAfter != 1.0 {
		t.Fatalf("expected full coverage receipt, got %+v", results[0].Receipt)
	}
}

func TestBuild_SkipTestsAndCoverage(t *testing.T) {
	specText := sampleGoSpec("/path/that/does/not/exist", "1.0", "owned.txt")
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{id: "test:backend", results: []*core.ExecResult{execResult(map[string]string{
		"owned.txt": "updated\n",
	})}}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	results, err := builder.Build(context.Background(), BuildOptions{
		SpecID:       "hello/greeter",
		SkipTests:    true,
		SkipCoverage: true,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if results[0].Status != "success" {
		t.Fatalf("expected success, got %+v", results[0])
	}
}

func TestBuild_SpecNotFound(t *testing.T) {
	specText := sampleGoSpec("/bin/echo ok", "0.0", "owned.txt")
	dir, cfg, registry := setupProject(t, specText)

	backend := &sequenceBackend{id: "test:backend", results: []*core.ExecResult{execResult(map[string]string{
		"owned.txt": "updated\n",
	})}}
	if err := registry.Register(backend); err != nil {
		t.Fatalf("register backend: %v", err)
	}

	builder := NewBuilder(cfg, registry, dir)
	if _, err := builder.Build(context.Background(), BuildOptions{SpecID: "missing/spec"}); err == nil {
		t.Fatal("expected missing spec error")
	}
}

func TestLoadSpecs(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "spec")
	mustMkdirAll(t, filepath.Join(specDir, "hello"))

	specData := []byte(sampleGoSpec("/bin/echo ok", "0.0", "owned.txt"))
	mustWriteFile(t, filepath.Join(specDir, "hello", "greeter.md"), specData)

	result, err := LoadSpecs(specDir)
	if err != nil {
		t.Fatalf("LoadSpecs failed: %v", err)
	}
	if len(result.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(result.Specs))
	}
	if result.Graph.Size() != 1 {
		t.Fatalf("expected graph size 1, got %d", result.Graph.Size())
	}
}
