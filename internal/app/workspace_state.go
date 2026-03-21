package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

type fileState struct {
	Exists bool
	Mode   os.FileMode
	Data   []byte
}

type workspaceState struct {
	root  string
	files map[string]fileState
}

func captureWorkspaceState(root string, paths []string) (*workspaceState, error) {
	state := &workspaceState{
		root:  root,
		files: make(map[string]fileState),
	}

	for _, path := range uniquePaths(paths) {
		abs := filepath.Join(root, path)
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				state.files[path] = fileState{}
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if info.IsDir() {
			continue
		}

		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		state.files[path] = fileState{
			Exists: true,
			Mode:   info.Mode(),
			Data:   data,
		}
	}

	return state, nil
}

func (s *workspaceState) Restore() error {
	for _, path := range uniquePaths(mapKeys(s.files)) {
		snapshot := s.files[path]
		abs := filepath.Join(s.root, path)

		if !snapshot.Exists {
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", path, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
		}
		mode := snapshot.Mode
		if mode == 0 {
			mode = 0644
		}
		if err := os.WriteFile(abs, snapshot.Data, mode); err != nil {
			return fmt.Errorf("restore %s: %w", path, err)
		}
	}

	return nil
}

func managedPaths(specPaths ...[]string) []string {
	var paths []string
	for _, group := range specPaths {
		paths = append(paths, group...)
	}
	return uniquePaths(paths)
}

func patchPathsForSnapshot(ops []core.PatchOp) []string {
	var paths []string
	for _, op := range ops {
		switch patch := op.(type) {
		case core.RenameFile:
			paths = append(paths, patch.From, patch.To)
		default:
			paths = append(paths, op.Path())
		}
	}
	return uniquePaths(paths)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var out []string
	for _, path := range paths {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
