// Package worktree manages git worktree creation and cleanup for isolated builds.
//
// Each spec build runs in an isolated git worktree to prevent interference
// between concurrent builds and to protect the main working tree from
// partial or failed builds.
//
// Lifecycle:
//  1. Create worktree: `git worktree add <path> --detach`
//  2. Run build pipeline in worktree
//  3. If successful, copy patches back to main tree
//  4. Remove worktree: `git worktree remove <path>`
package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles git worktree lifecycle.
type Manager struct {
	repoDir  string
	stateDir string // e.g., .plaincode/worktrees
}

// NewManager creates a worktree manager for the given repository.
func NewManager(repoDir, stateDir string) *Manager {
	return &Manager{
		repoDir:  repoDir,
		stateDir: filepath.Join(stateDir, "worktrees"),
	}
}

// Create creates a new git worktree for a build.
// Returns the path to the worktree directory.
func (m *Manager) Create(buildID string) (string, error) {
	wtPath := filepath.Join(m.stateDir, buildID)

	if err := os.MkdirAll(filepath.Dir(wtPath), 0755); err != nil {
		return "", fmt.Errorf("creating worktree parent: %w", err)
	}

	// git worktree add --detach <path>
	// --detach creates the worktree at HEAD without a branch
	cmd := exec.Command("git", "worktree", "add", "--detach", wtPath)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	return wtPath, nil
}

// Remove removes a git worktree.
func (m *Manager) Remove(buildID string) error {
	wtPath := filepath.Join(m.stateDir, buildID)

	cmd := exec.Command("git", "worktree", "remove", "--force", wtPath)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: try manual cleanup
		if rmErr := os.RemoveAll(wtPath); rmErr != nil {
			return fmt.Errorf("git worktree remove failed (%w: %s), manual cleanup also failed: %v",
				err, strings.TrimSpace(string(output)), rmErr)
		}
		// Prune stale worktree references
		pruneCmd := exec.Command("git", "worktree", "prune")
		pruneCmd.Dir = m.repoDir
		_ = pruneCmd.Run()
	}

	return nil
}

// List returns all active worktree paths managed by forge.
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			paths = append(paths, filepath.Join(m.stateDir, e.Name()))
		}
	}
	return paths, nil
}

// Cleanup removes all forge-managed worktrees.
func (m *Manager) Cleanup() error {
	worktrees, err := m.List()
	if err != nil {
		return err
	}
	for _, wt := range worktrees {
		buildID := filepath.Base(wt)
		if err := m.Remove(buildID); err != nil {
			return fmt.Errorf("removing worktree %s: %w", buildID, err)
		}
	}
	return nil
}
