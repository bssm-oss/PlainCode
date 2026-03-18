// Package graph builds and manages the spec dependency graph.
//
// The build graph determines the order in which specs must be built,
// detects cycles, identifies dirty specs that need rebuilding, and
// enables incremental builds by tracking spec hashes against receipts.
package graph

import (
	"fmt"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

// Node represents a single spec in the build graph.
type Node struct {
	Spec     *ast.Spec
	Deps     []*Node
	IsDirty  bool
	LastHash string // from last build receipt, empty if never built
}

// BuildGraph represents the complete spec dependency graph.
type BuildGraph struct {
	nodes map[string]*Node
	order []string // topological order
}

// New creates an empty build graph.
func New() *BuildGraph {
	return &BuildGraph{
		nodes: make(map[string]*Node),
	}
}

// AddSpec adds a spec to the graph.
func (g *BuildGraph) AddSpec(spec *ast.Spec) {
	if _, exists := g.nodes[spec.ID]; exists {
		return
	}
	g.nodes[spec.ID] = &Node{
		Spec: spec,
	}
}

// AddEdge declares that `from` depends on `to`.
func (g *BuildGraph) AddEdge(fromID, toID string) error {
	from, ok := g.nodes[fromID]
	if !ok {
		return fmt.Errorf("unknown spec: %s", fromID)
	}
	to, ok := g.nodes[toID]
	if !ok {
		return fmt.Errorf("unknown spec: %s", toID)
	}
	from.Deps = append(from.Deps, to)
	return nil
}

// TopologicalSort returns specs in dependency order (dependencies first).
// Returns an error if cycles are detected.
func (g *BuildGraph) TopologicalSort() ([]*Node, error) {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var result []*Node

	var visit func(id string) error
	visit = func(id string) error {
		if inStack[id] {
			return fmt.Errorf("cycle detected involving spec: %s", id)
		}
		if visited[id] {
			return nil
		}

		inStack[id] = true
		node := g.nodes[id]

		for _, dep := range node.Deps {
			if err := visit(dep.Spec.ID); err != nil {
				return err
			}
		}

		inStack[id] = false
		visited[id] = true
		result = append(result, node)
		return nil
	}

	for id := range g.nodes {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	g.order = make([]string, len(result))
	for i, n := range result {
		g.order[i] = n.Spec.ID
	}

	return result, nil
}

// MarkDirty compares current spec hashes against known receipt hashes
// and marks nodes that need rebuilding.
func (g *BuildGraph) MarkDirty(receiptHashes map[string]string) {
	for id, node := range g.nodes {
		lastHash, exists := receiptHashes[id]
		if !exists || lastHash != node.Spec.Hash {
			node.IsDirty = true
		}
	}

	// Propagate: if a dependency is dirty, dependents are also dirty
	changed := true
	for changed {
		changed = false
		for _, node := range g.nodes {
			if node.IsDirty {
				continue
			}
			for _, dep := range node.Deps {
				if dep.IsDirty {
					node.IsDirty = true
					changed = true
					break
				}
			}
		}
	}
}

// DirtyNodes returns only nodes that need rebuilding, in topological order.
func (g *BuildGraph) DirtyNodes() ([]*Node, error) {
	all, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}
	var dirty []*Node
	for _, n := range all {
		if n.IsDirty {
			dirty = append(dirty, n)
		}
	}
	return dirty, nil
}

// Node returns the graph node for a spec ID, or nil if not found.
func (g *BuildGraph) Node(id string) *Node {
	return g.nodes[id]
}

// Size returns the number of specs in the graph.
func (g *BuildGraph) Size() int {
	return len(g.nodes)
}
