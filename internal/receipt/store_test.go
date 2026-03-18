package receipt

import (
	"testing"
	"time"
)

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	r := &Receipt{
		BuildID:         "build-001",
		SpecID:          "hello/greeter",
		SpecHash:        "abc123",
		BackendID:       "mock:test",
		ApprovalProfile: "patch",
		ChangedFiles:    []string{"internal/greeter/greeter.go"},
		TestsRun:        5,
		TestsPassed:     5,
		TestsFailed:     0,
		CoverageBefore:  0.0,
		CoverageAfter:   0.85,
		Status:          "success",
		StartedAt:       time.Now().Add(-10 * time.Second),
		CompletedAt:     time.Now(),
	}

	if err := store.Save(r); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load("build-001")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.SpecID != "hello/greeter" {
		t.Errorf("expected spec_id 'hello/greeter', got %q", loaded.SpecID)
	}
	if loaded.Status != "success" {
		t.Errorf("expected status 'success', got %q", loaded.Status)
	}
	if loaded.TestsPassed != 5 {
		t.Errorf("expected 5 tests passed, got %d", loaded.TestsPassed)
	}
}

func TestStore_SpecHashes(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Save two receipts for different specs
	store.Save(&Receipt{
		BuildID:     "build-001",
		SpecID:      "hello/greeter",
		SpecHash:    "hash-1",
		Status:      "success",
		CompletedAt: time.Now().Add(-1 * time.Hour),
	})
	store.Save(&Receipt{
		BuildID:     "build-002",
		SpecID:      "billing/invoice",
		SpecHash:    "hash-2",
		Status:      "success",
		CompletedAt: time.Now(),
	})

	hashes, err := store.SpecHashes()
	if err != nil {
		t.Fatalf("SpecHashes failed: %v", err)
	}

	if hashes["hello/greeter"] != "hash-1" {
		t.Errorf("expected hash-1 for hello/greeter, got %q", hashes["hello/greeter"])
	}
	if hashes["billing/invoice"] != "hash-2" {
		t.Errorf("expected hash-2 for billing/invoice, got %q", hashes["billing/invoice"])
	}
}

func TestStore_LoadMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing receipt")
	}
}
