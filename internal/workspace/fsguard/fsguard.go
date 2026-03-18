package fsguard

import (
	"fmt"
	"path/filepath"
)

// FileClass represents the ownership classification of a file.
type FileClass int

const (
	Owned FileClass = iota
	Shared
	ReadOnly
	OwnedByOtherSpec
	OutsideProject
	Unmanaged
)

// OwnershipMap tracks file ownership across all specs.
type OwnershipMap struct {
	owned    map[string]string // path -> spec ID
	shared   map[string]bool
	readonly map[string]bool
}

// NewOwnershipMap creates an empty ownership map.
func NewOwnershipMap() *OwnershipMap {
	return &OwnershipMap{
		owned:    make(map[string]string),
		shared:   make(map[string]bool),
		readonly: make(map[string]bool),
	}
}

// RegisterSpec adds a spec's file declarations to the ownership map.
func (m *OwnershipMap) RegisterSpec(specID string, owned, shared, readonly []string) {
	for _, p := range owned {
		m.owned[filepath.Clean(p)] = specID
	}
	for _, p := range shared {
		m.shared[filepath.Clean(p)] = true
	}
	for _, p := range readonly {
		m.readonly[filepath.Clean(p)] = true
	}
}

// Classify determines the ownership class of a file for a given spec.
func (m *OwnershipMap) Classify(specID, path string) FileClass {
	path = filepath.Clean(path)

	if m.readonly[path] {
		return ReadOnly
	}

	if owner, ok := m.owned[path]; ok {
		if owner == specID {
			return Owned
		}
		return OwnedByOtherSpec
	}

	if m.shared[path] {
		return Shared
	}

	return Unmanaged
}

// ValidatePatch checks that all patch operations respect file ownership.
func ValidatePatch(specID string, paths []string, ownership *OwnershipMap) error {
	for _, path := range paths {
		switch ownership.Classify(specID, path) {
		case ReadOnly:
			return fmt.Errorf("cannot modify readonly file: %s", path)
		case OwnedByOtherSpec:
			return fmt.Errorf("file owned by another spec: %s", path)
		case OutsideProject:
			return fmt.Errorf("file outside project scope: %s", path)
		case Owned, Shared, Unmanaged:
			// allowed
		}
	}
	return nil
}
