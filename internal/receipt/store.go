// Package receipt also provides storage for build receipts.
// Receipts are stored as JSON files under .plaincode/builds/<build-id>/.
package receipt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Store manages build receipt persistence.
type Store struct {
	stateDir string // e.g., .plaincode
}

// NewStore creates a receipt store rooted at the given state directory.
func NewStore(stateDir string) *Store {
	return &Store{stateDir: stateDir}
}

// Save writes a receipt to disk.
func (s *Store) Save(r *Receipt) error {
	dir := s.buildDir(r.BuildID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating build directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling receipt: %w", err)
	}

	path := filepath.Join(dir, "receipt.json")
	return os.WriteFile(path, data, 0644)
}

// Load reads a receipt from disk by build ID.
func (s *Store) Load(buildID string) (*Receipt, error) {
	path := filepath.Join(s.buildDir(buildID), "receipt.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading receipt: %w", err)
	}

	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing receipt: %w", err)
	}

	return &r, nil
}

// LatestForSpec returns the most recent receipt for a spec ID.
func (s *Store) LatestForSpec(specID string) (*Receipt, error) {
	builds, err := s.ListAll()
	if err != nil {
		return nil, err
	}

	var latest *Receipt
	for _, r := range builds {
		if r.SpecID == specID {
			if latest == nil || r.CompletedAt.After(latest.CompletedAt) {
				latest = r
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no receipts found for spec: %s", specID)
	}
	return latest, nil
}

// ListAll returns all stored receipts, sorted by completion time (newest first).
func (s *Store) ListAll() ([]*Receipt, error) {
	buildsDir := filepath.Join(s.stateDir, "builds")
	entries, err := os.ReadDir(buildsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var receipts []*Receipt
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		r, err := s.Load(entry.Name())
		if err != nil {
			continue // skip corrupt receipts
		}
		receipts = append(receipts, r)
	}

	sort.Slice(receipts, func(i, j int) bool {
		return receipts[i].CompletedAt.After(receipts[j].CompletedAt)
	})

	return receipts, nil
}

// SpecHashes returns a map of spec ID → last known hash from receipts.
// Used by the build graph for dirty detection.
func (s *Store) SpecHashes() (map[string]string, error) {
	receipts, err := s.ListAll()
	if err != nil {
		return nil, err
	}

	hashes := make(map[string]string)
	for _, r := range receipts {
		if r.Status == "success" {
			if _, exists := hashes[r.SpecID]; !exists {
				hashes[r.SpecID] = r.SpecHash
			}
		}
	}

	return hashes, nil
}

// SaveArtifact writes a named artifact to the build directory.
func (s *Store) SaveArtifact(buildID, name string, data []byte) error {
	dir := s.buildDir(buildID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0644)
}

func (s *Store) buildDir(buildID string) string {
	return filepath.Join(s.stateDir, "builds", buildID)
}
