// Package imports resolves spec import references to actual spec files.
//
// When a spec declares `imports: [billing/shared/money]`, the resolver
// locates the corresponding spec file in the spec directory, verifies
// it exists and parses successfully, and returns the resolved spec.
//
// Circular imports are detected and rejected.
package imports

import (
	"fmt"
	"path/filepath"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
	"github.com/bssm-oss/PlainCode/internal/spec/parser"
)

// Resolver resolves spec imports to parsed specs.
type Resolver struct {
	specDir string
	cache   map[string]*ast.Spec
	loading map[string]bool // circular import detection
}

// NewResolver creates a resolver rooted at the given spec directory.
func NewResolver(specDir string) *Resolver {
	return &Resolver{
		specDir: specDir,
		cache:   make(map[string]*ast.Spec),
		loading: make(map[string]bool),
	}
}

// Resolve resolves a single spec ID to its parsed spec.
// It caches results and detects circular imports.
func (r *Resolver) Resolve(id string) (*ast.Spec, error) {
	if spec, ok := r.cache[id]; ok {
		return spec, nil
	}

	if r.loading[id] {
		return nil, fmt.Errorf("circular import detected: %s", id)
	}

	r.loading[id] = true
	defer func() { delete(r.loading, id) }()

	path := r.specPath(id)
	spec, err := parser.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("resolving import %s: %w", id, err)
	}

	// Recursively resolve this spec's imports
	for _, importID := range spec.Imports {
		if _, err := r.Resolve(importID); err != nil {
			return nil, fmt.Errorf("resolving transitive import %s (from %s): %w", importID, id, err)
		}
	}

	r.cache[id] = spec
	return spec, nil
}

// ResolveAll resolves all imports for a given spec, returning
// a map of import ID -> parsed spec.
func (r *Resolver) ResolveAll(spec *ast.Spec) (map[string]*ast.Spec, error) {
	result := make(map[string]*ast.Spec)
	for _, id := range spec.Imports {
		resolved, err := r.Resolve(id)
		if err != nil {
			return nil, err
		}
		result[id] = resolved
	}
	return result, nil
}

// specPath converts a spec ID to a filesystem path.
// Example: "billing/shared/money" -> "<specDir>/billing/shared/money.md"
func (r *Resolver) specPath(id string) string {
	return filepath.Join(r.specDir, id+".md")
}

// AllResolved returns all specs that have been resolved so far.
func (r *Resolver) AllResolved() map[string]*ast.Spec {
	result := make(map[string]*ast.Spec, len(r.cache))
	for k, v := range r.cache {
		result[k] = v
	}
	return result
}
