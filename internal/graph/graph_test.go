package graph

import (
	"testing"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

func makeSpec(id string, imports ...string) *ast.Spec {
	return &ast.Spec{
		ID:       id,
		Language: "go",
		Hash:     "hash-" + id,
		Imports:  imports,
	}
}

func TestTopologicalSort_Simple(t *testing.T) {
	g := New()
	g.AddSpec(makeSpec("a"))
	g.AddSpec(makeSpec("b"))
	g.AddSpec(makeSpec("c"))

	if err := g.AddEdge("c", "b"); err != nil {
		t.Fatal(err)
	}
	if err := g.AddEdge("b", "a"); err != nil {
		t.Fatal(err)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	// a must come before b, b before c
	idx := make(map[string]int)
	for i, n := range order {
		idx[n.Spec.ID] = i
	}
	if idx["a"] > idx["b"] || idx["b"] > idx["c"] {
		t.Errorf("wrong order: a=%d b=%d c=%d", idx["a"], idx["b"], idx["c"])
	}
}

func TestTopologicalSort_CycleDetection(t *testing.T) {
	g := New()
	g.AddSpec(makeSpec("a"))
	g.AddSpec(makeSpec("b"))

	_ = g.AddEdge("a", "b")
	_ = g.AddEdge("b", "a")

	_, err := g.TopologicalSort()
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestMarkDirty_NewSpec(t *testing.T) {
	g := New()
	g.AddSpec(makeSpec("a"))

	g.MarkDirty(map[string]string{}) // no receipts
	if !g.Node("a").IsDirty {
		t.Error("new spec should be dirty")
	}
}

func TestMarkDirty_UnchangedSpec(t *testing.T) {
	g := New()
	g.AddSpec(makeSpec("a"))

	g.MarkDirty(map[string]string{"a": "hash-a"})
	if g.Node("a").IsDirty {
		t.Error("unchanged spec should not be dirty")
	}
}

func TestMarkDirty_Propagation(t *testing.T) {
	g := New()
	g.AddSpec(makeSpec("a"))
	g.AddSpec(makeSpec("b"))
	_ = g.AddEdge("b", "a")

	// a changed, b unchanged
	g.MarkDirty(map[string]string{"a": "old-hash", "b": "hash-b"})

	if !g.Node("a").IsDirty {
		t.Error("changed spec should be dirty")
	}
	if !g.Node("b").IsDirty {
		t.Error("dependent of dirty spec should be dirty")
	}
}

func TestDirtyNodes_Order(t *testing.T) {
	g := New()
	g.AddSpec(makeSpec("a"))
	g.AddSpec(makeSpec("b"))
	g.AddSpec(makeSpec("c"))
	_ = g.AddEdge("b", "a")
	_ = g.AddEdge("c", "b")

	g.MarkDirty(map[string]string{}) // all new
	dirty, err := g.DirtyNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirty) != 3 {
		t.Fatalf("expected 3 dirty, got %d", len(dirty))
	}
}
