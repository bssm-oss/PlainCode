// Package cli provides shared utilities for all CLI backend adapters.
//
// All CLI adapters use os/exec with arg arrays (no shell) for safety.
// This package provides common helpers for execution, output parsing,
// and binary detection.
package cli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

// ExecResult holds the raw output from a CLI invocation.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecCLI runs a CLI binary with the given args. No shell involved.
func ExecCLI(ctx context.Context, binary string, args []string, workDir string, env map[string]string) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workDir

	if len(env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, fmt.Errorf("executing %s: %w", binary, err)
		}
	}

	return result, nil
}

// CheckBinary verifies a CLI binary is installed by running "<binary> --version".
func CheckBinary(ctx context.Context, binary string) error {
	cmd := exec.CommandContext(ctx, binary, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s not found or not executable: %w\n%s", binary, err, string(output))
	}
	return nil
}

// ParseFileBlocks extracts file patches from CLI text output.
// Expected format:
//
//	--- FILE: path/to/file.go ---
//	<file content>
//	--- END FILE ---
//
// Returns a list of WriteFile PatchOps.
func ParseFileBlocks(output string) []core.PatchOp {
	var ops []core.PatchOp
	lines := strings.Split(output, "\n")

	var currentPath string
	var currentContent []string
	inBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "--- FILE: ") && strings.HasSuffix(line, " ---") {
			// Start of file block
			path := strings.TrimPrefix(line, "--- FILE: ")
			path = strings.TrimSuffix(path, " ---")
			currentPath = strings.TrimSpace(path)
			currentContent = nil
			inBlock = true
		} else if line == "--- END FILE ---" && inBlock {
			// End of file block
			if currentPath != "" {
				ops = append(ops, core.WriteFile{
					FilePath: currentPath,
					Content:  []byte(strings.Join(currentContent, "\n")),
				})
			}
			inBlock = false
			currentPath = ""
		} else if inBlock {
			currentContent = append(currentContent, line)
		}
	}

	return ops
}
