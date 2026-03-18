package ir

import (
	"testing"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

func makeSpec(id string, owned []string, imports ...string) *ast.Spec {
	return &ast.Spec{
		ID:       id,
		Language: "go",
		Hash:     "hash-" + id,
		Imports:  imports,
		ManagedFiles: ast.ManagedFiles{
			Owned: owned,
		},
	}
}

func TestResolve_Simple(t *testing.T) {
	specs := map[string]*ast.Spec{
		"a": makeSpec("a", []string{"a.go"}),
		"b": makeSpec("b", []string{"b.go"}, "a"),
	}

	irs, err := Resolve(specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(irs) != 2 {
		t.Fatalf("expected 2 IRs, got %d", len(irs))
	}

	bIR := irs["b"]
	if len(bIR.ResolvedImports) != 1 {
		t.Errorf("expected 1 resolved import, got %d", len(bIR.ResolvedImports))
	}
	if _, ok := bIR.ResolvedImports["a"]; !ok {
		t.Error("expected 'a' in resolved imports")
	}
}

func TestResolve_MissingImport(t *testing.T) {
	specs := map[string]*ast.Spec{
		"a": makeSpec("a", nil, "nonexistent"),
	}

	_, err := Resolve(specs)
	if err == nil {
		t.Fatal("expected error for missing import")
	}
}

func TestResolve_OwnershipConflict(t *testing.T) {
	specs := map[string]*ast.Spec{
		"a": makeSpec("a", []string{"shared.go"}),
		"b": makeSpec("b", []string{"shared.go"}),
	}

	_, err := Resolve(specs)
	if err == nil {
		t.Fatal("expected error for ownership conflict")
	}
}

func TestResolve_NoConflict(t *testing.T) {
	specs := map[string]*ast.Spec{
		"a": makeSpec("a", []string{"a.go"}),
		"b": makeSpec("b", []string{"b.go"}),
	}

	_, err := Resolve(specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
