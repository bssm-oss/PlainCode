package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProjectConfig(t *testing.T) {
	cfg := DefaultProjectConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Project.SpecDir != "spec" {
		t.Errorf("expected spec_dir 'spec', got %q", cfg.Project.SpecDir)
	}
	if cfg.Project.StateDir != ".plaincode" {
		t.Errorf("expected state_dir '.plaincode', got %q", cfg.Project.StateDir)
	}
	if cfg.Defaults.Approval != "patch" {
		t.Errorf("expected approval 'patch', got %q", cfg.Defaults.Approval)
	}
	if cfg.Defaults.Backend != "cli:codex" {
		t.Errorf("expected backend 'cli:codex', got %q", cfg.Defaults.Backend)
	}
	if provider, ok := cfg.Providers["cli:codex"]; !ok {
		t.Fatal("expected cli:codex provider in defaults")
	} else if provider.Kind != "cli-codex" {
		t.Errorf("expected cli:codex kind 'cli-codex', got %q", provider.Kind)
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := DefaultProjectConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_BadVersion(t *testing.T) {
	cfg := DefaultProjectConfig()
	cfg.Version = 99
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for bad version")
	}
}

func TestValidate_EmptySpecDir(t *testing.T) {
	cfg := DefaultProjectConfig()
	cfg.Project.SpecDir = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty spec_dir")
	}
}

func TestWriteAndLoad(t *testing.T) {
	dir := t.TempDir()

	if err := WriteDefault(dir); err != nil {
		t.Fatalf("WriteDefault failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, "plaincode.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("plaincode.yaml not created: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("loaded version %d, expected 1", cfg.Version)
	}
	if cfg.Project.SpecDir != "spec" {
		t.Errorf("loaded spec_dir %q, expected 'spec'", cfg.Project.SpecDir)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load should return defaults for missing file, got error: %v", err)
	}
	if cfg.Version != 1 {
		t.Error("expected default config for missing file")
	}
}
