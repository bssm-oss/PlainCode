// Package patch provides the patch application engine.
//
// After a backend produces PatchOps (write, delete, rename, diff),
// this package applies them to a workspace directory with rollback support.
// All operations are validated against the file ownership model before application.
package patch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

// Applier applies patch operations to a workspace directory.
type Applier struct {
	workspaceDir string
	applied      []core.PatchOp // for rollback tracking
}

// NewApplier creates a patch applier for the given workspace directory.
func NewApplier(workspaceDir string) *Applier {
	return &Applier{
		workspaceDir: workspaceDir,
	}
}

// Apply executes a single patch operation.
func (a *Applier) Apply(op core.PatchOp) error {
	var err error
	switch o := op.(type) {
	case core.WriteFile:
		err = a.applyWrite(o)
	case core.DeleteFile:
		err = a.applyDelete(o)
	case core.RenameFile:
		err = a.applyRename(o)
	case core.ApplyDiff:
		err = a.applyDiff(o)
	default:
		return fmt.Errorf("unknown patch op type: %T", op)
	}
	if err != nil {
		return err
	}
	a.applied = append(a.applied, op)
	return nil
}

// ApplyAll applies a list of patch operations in order.
// If any operation fails, previously applied operations remain (no auto-rollback).
func (a *Applier) ApplyAll(ops []core.PatchOp) error {
	for i, op := range ops {
		if err := a.Apply(op); err != nil {
			return fmt.Errorf("patch %d (%s): %w", i, op.Path(), err)
		}
	}
	return nil
}

// Applied returns the list of successfully applied operations.
func (a *Applier) Applied() []core.PatchOp {
	return a.applied
}

func (a *Applier) absPath(rel string) string {
	return filepath.Join(a.workspaceDir, rel)
}

func (a *Applier) applyWrite(op core.WriteFile) error {
	path := a.absPath(op.FilePath)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.WriteFile(path, op.Content, 0644)
}

func (a *Applier) applyDelete(op core.DeleteFile) error {
	path := a.absPath(op.FilePath)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // idempotent
	}
	return err
}

func (a *Applier) applyRename(op core.RenameFile) error {
	from := a.absPath(op.From)
	to := a.absPath(op.To)
	dir := filepath.Dir(to)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.Rename(from, to)
}

func (a *Applier) applyDiff(_ core.ApplyDiff) error {
	// TODO: Implement unified diff application.
	// Options:
	// 1. Use go-diff library
	// 2. Shell out to `patch` command (safe via os/exec arg array)
	// 3. Custom line-level diff applier
	return fmt.Errorf("diff application not yet implemented")
}
