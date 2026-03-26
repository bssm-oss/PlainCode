// Package app provides the high-level build orchestration logic.
// It connects the spec parser, build graph, backend registry,
// ownership validator, and receipt store into a working pipeline.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bssm-oss/PlainCode/internal/graph"
	"github.com/bssm-oss/PlainCode/internal/spec/ast"
	"github.com/bssm-oss/PlainCode/internal/spec/parser"
	"github.com/bssm-oss/PlainCode/internal/workspace/fsguard"
)

// LoadResult holds all specs, the build graph, and the ownership map.
type LoadResult struct {
	Specs     map[string]*ast.Spec
	Graph     *graph.BuildGraph
	Ownership *fsguard.OwnershipMap
	Errors    []LoadError
}

// LoadError describes a spec that failed to parse.
type LoadError struct {
	Path string
	Err  error
}

// LoadSpecs scans the spec directory, parses all .md files,
// resolves imports, builds the dependency graph, and constructs
// the ownership map.
func LoadSpecs(specDir string) (*LoadResult, error) {
	result := &LoadResult{
		Specs:     make(map[string]*ast.Spec),
		Graph:     graph.New(),
		Ownership: fsguard.NewOwnershipMap(),
	}

	// Recursively find all .md files in specDir
	err := filepath.Walk(specDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") || shouldIgnoreSpecFile(path) {
			return nil
		}

		spec, parseErr := parser.ParseFile(path)
		if parseErr != nil {
			result.Errors = append(result.Errors, LoadError{Path: path, Err: parseErr})
			return nil // continue scanning
		}

		result.Specs[spec.ID] = spec
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking spec dir %s: %w", specDir, err)
	}

	if len(result.Specs) == 0 && len(result.Errors) == 0 {
		return result, nil // empty project, not an error
	}

	// Build graph nodes
	for _, spec := range result.Specs {
		result.Graph.AddSpec(spec)
	}

	// Add dependency edges from imports
	for _, spec := range result.Specs {
		for _, importID := range spec.Imports {
			if _, exists := result.Specs[importID]; !exists {
				result.Errors = append(result.Errors, LoadError{
					Path: spec.SourcePath,
					Err:  fmt.Errorf("imports unknown spec %q", importID),
				})
				continue
			}
			if err := result.Graph.AddEdge(spec.ID, importID); err != nil {
				result.Errors = append(result.Errors, LoadError{
					Path: spec.SourcePath,
					Err:  fmt.Errorf("adding edge: %w", err),
				})
			}
		}
	}

	// Register ownership
	for _, spec := range result.Specs {
		result.Ownership.RegisterSpec(
			spec.ID,
			spec.ManagedFiles.Owned,
			spec.ManagedFiles.Shared,
			spec.ManagedFiles.Readonly,
		)
	}

	return result, nil
}

// LoadSingleSpec loads just one spec by ID from the spec directory.
func LoadSingleSpec(specDir, specID string) (*ast.Spec, error) {
	path := filepath.Join(specDir, specID+".md")
	return parser.ParseFile(path)
}

func shouldIgnoreSpecFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "_")
}
