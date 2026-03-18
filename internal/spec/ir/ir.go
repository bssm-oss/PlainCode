// Package ir defines the normalized Spec Intermediate Representation.
//
// While the AST is a direct parse result, the IR is a resolved, validated
// representation ready for the build graph. Import references are resolved
// to actual spec pointers, file paths are normalized, and cross-spec
// ownership conflicts are detected.
package ir

import "github.com/bssm-oss/PlainCode/internal/spec/ast"

// SpecIR is the normalized, resolved representation of a spec.
type SpecIR struct {
	// Spec is the underlying parsed AST.
	Spec *ast.Spec

	// ResolvedImports maps import ID to the resolved SpecIR.
	ResolvedImports map[string]*SpecIR

	// NormalizedOwned contains cleaned, absolute-relative paths for owned files.
	NormalizedOwned []string

	// NormalizedShared contains cleaned paths for shared files.
	NormalizedShared []string

	// NormalizedReadonly contains cleaned paths (may include globs) for readonly files.
	NormalizedReadonly []string

	// DependsOn lists spec IDs that must be built before this one.
	DependsOn []string

	// IsDirty indicates whether this spec needs rebuilding.
	// Determined by comparing current hash vs last receipt hash.
	IsDirty bool
}

// ID returns the spec identifier.
func (s *SpecIR) ID() string {
	return s.Spec.ID
}

// Hash returns the spec content hash.
func (s *SpecIR) Hash() string {
	return s.Spec.Hash
}

// TODO: Add Resolve() function that takes a map[string]*ast.Spec
// and produces a []*SpecIR with all imports resolved and paths normalized.
