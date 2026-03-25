package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bssm-oss/PlainCode/internal/config"
)

func TestInitProjectCreatesStarterFiles(t *testing.T) {
	dir := t.TempDir()

	if err := initProject(dir); err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	requiredPaths := []string{
		filepath.Join(dir, "plaincode.yaml"),
		filepath.Join(dir, "spec"),
		filepath.Join(dir, ".plaincode"),
		filepath.Join(dir, ".plaincode", "builds"),
		filepath.Join(dir, initBlueprintPath),
		filepath.Join(dir, initReadmePath),
	}
	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	if cfg.Defaults.Backend != "cli:codex" {
		t.Fatalf("default backend = %q, want %q", cfg.Defaults.Backend, "cli:codex")
	}
	provider, ok := cfg.Providers["cli:codex"]
	if !ok {
		t.Fatal("expected cli:codex provider to be created")
	}
	if provider.Kind != "cli-codex" {
		t.Fatalf("provider kind = %q, want %q", provider.Kind, "cli-codex")
	}

	blueprintData, err := os.ReadFile(filepath.Join(dir, initBlueprintPath))
	if err != nil {
		t.Fatalf("reading blueprint: %v", err)
	}
	blueprint := string(blueprintData)
	if !strings.Contains(blueprint, "id: example/feature") {
		t.Fatalf("blueprint is missing starter spec frontmatter:\n%s", blueprint)
	}
	if !strings.Contains(blueprint, "plaincode build --spec <id>") {
		t.Fatalf("blueprint is missing build instructions:\n%s", blueprint)
	}

	readmeData, err := os.ReadFile(filepath.Join(dir, initReadmePath))
	if err != nil {
		t.Fatalf("reading README: %v", err)
	}
	readme := string(readmeData)
	if !strings.Contains(readme, "PlainCode 시작 가이드") {
		t.Fatalf("README is missing Korean title:\n%s", readme)
	}
	if !strings.Contains(readme, "cp spec/blueprint.md.txt spec/hello.md") {
		t.Fatalf("README is missing quick-start copy command:\n%s", readme)
	}
}

func TestInitProjectFailsWhenConfigAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plaincode.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("writing plaincode.yaml: %v", err)
	}

	err := initProject(dir)
	if err == nil {
		t.Fatal("expected initProject to fail when plaincode.yaml already exists")
	}
	if !strings.Contains(err.Error(), "plaincode.yaml already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}
