package execenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveBinaryFromFallbackDir(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "demo-tool")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	resolved := resolveBinary("demo-tool", []string{dir})
	if resolved != binary {
		t.Fatalf("resolved = %q, want %q", resolved, binary)
	}
}

func TestResolveBinaryKeepsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "demo-tool")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	resolved := resolveBinary(binary, nil)
	if resolved != binary {
		t.Fatalf("resolved = %q, want %q", resolved, binary)
	}
}

func TestEnsurePathDeduplicatesAndAppendsFallbacks(t *testing.T) {
	dir := t.TempDir()
	pathValue := ensurePath(dir+string(os.PathListSeparator)+dir, []string{dir, "/definitely/missing"})
	parts := strings.Split(pathValue, string(os.PathListSeparator))
	if len(parts) != 1 || parts[0] != dir {
		t.Fatalf("parts = %v, want [%q]", parts, dir)
	}
}
