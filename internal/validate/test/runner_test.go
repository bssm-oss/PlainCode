package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerRun_Success(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "success.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	result, err := NewRunner().Run(context.Background(), dir, script)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !result.Passed || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Stdout != "ok\n" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestRunnerRun_Failure(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "failure.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho boom >&2\nexit 7\n"), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	result, err := NewRunner().Run(context.Background(), dir, script)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Passed || result.ExitCode != 7 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Stderr != "boom\n" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}
