// Package core also provides a backend registry for managing
// and selecting AI backends at runtime.
package core

import (
	"fmt"
	"sync"
)

// Registry manages available backends and selects them for builds.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]Backend
}

// NewRegistry creates an empty backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]Backend),
	}
}

// Register adds a backend to the registry.
func (r *Registry) Register(b Backend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := b.ID()
	if _, exists := r.backends[id]; exists {
		return fmt.Errorf("backend already registered: %s", id)
	}
	r.backends[id] = b
	return nil
}

// Get returns a backend by ID.
func (r *Registry) Get(id string) (Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.backends[id]
	if !ok {
		return nil, fmt.Errorf("unknown backend: %s", id)
	}
	return b, nil
}

// Select picks the first available backend from a preference list.
// Falls back to the default if none of the preferred backends are available.
func (r *Registry) Select(preferred []string, defaultID string) (Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, id := range preferred {
		if b, ok := r.backends[id]; ok {
			return b, nil
		}
	}

	if defaultID != "" {
		if b, ok := r.backends[defaultID]; ok {
			return b, nil
		}
	}

	return nil, fmt.Errorf("no available backend from preferred list %v (default: %s)", preferred, defaultID)
}

// List returns all registered backend IDs.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.backends))
	for id := range r.backends {
		ids = append(ids, id)
	}
	return ids
}

// HealthCheckAll runs health checks on all registered backends
// and returns a map of backend ID -> error (nil = healthy).
func (r *Registry) HealthCheckAll() map[string]error {
	r.mu.RLock()
	backends := make(map[string]Backend, len(r.backends))
	for k, v := range r.backends {
		backends[k] = v
	}
	r.mu.RUnlock()

	results := make(map[string]error, len(backends))
	for id, b := range backends {
		results[id] = b.HealthCheck(nil) // TODO: pass real context
	}
	return results
}
