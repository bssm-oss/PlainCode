// Package ir also provides the Resolve function that converts
// parsed ASTs into normalized IRs with resolved imports and clean paths.
package ir

import (
	"fmt"
	"path/filepath"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

// Resolve converts a set of parsed specs into normalized SpecIRs.
// It resolves imports, normalizes file paths, and detects cross-spec
// ownership conflicts.
func Resolve(specs map[string]*ast.Spec) (map[string]*SpecIR, error) {
	irs := make(map[string]*SpecIR, len(specs))

	// First pass: create IR nodes
	for id, spec := range specs {
		irs[id] = &SpecIR{
			Spec:              spec,
			ResolvedImports:   make(map[string]*SpecIR),
			NormalizedOwned:   normalizePaths(spec.ManagedFiles.Owned),
			NormalizedShared:  normalizePaths(spec.ManagedFiles.Shared),
			NormalizedReadonly: normalizePaths(spec.ManagedFiles.Readonly),
			DependsOn:         spec.Imports,
		}
	}

	// Second pass: resolve imports
	for id, ir := range irs {
		for _, importID := range ir.Spec.Imports {
			imported, ok := irs[importID]
			if !ok {
				return nil, fmt.Errorf("spec %q imports unknown spec %q", id, importID)
			}
			ir.ResolvedImports[importID] = imported
		}
	}

	// Third pass: detect ownership conflicts
	if err := detectConflicts(irs); err != nil {
		return nil, err
	}

	return irs, nil
}

// normalizePaths cleans and normalizes file paths.
func normalizePaths(paths []string) []string {
	normalized := make([]string, len(paths))
	for i, p := range paths {
		normalized[i] = filepath.Clean(p)
	}
	return normalized
}

// detectConflicts checks for files owned by multiple specs.
func detectConflicts(irs map[string]*SpecIR) error {
	owners := make(map[string]string) // path -> spec ID

	for id, ir := range irs {
		for _, path := range ir.NormalizedOwned {
			if existingOwner, exists := owners[path]; exists {
				return fmt.Errorf("ownership conflict: %q is owned by both %q and %q",
					path, existingOwner, id)
			}
			owners[path] = id
		}
	}

	return nil
}
