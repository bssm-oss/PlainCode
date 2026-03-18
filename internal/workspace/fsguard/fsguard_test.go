package fsguard

import "testing"

func TestClassify_Owned(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("billing/invoice", []string{"internal/billing/invoice.go"}, nil, nil)

	if class := m.Classify("billing/invoice", "internal/billing/invoice.go"); class != Owned {
		t.Errorf("expected Owned, got %d", class)
	}
}

func TestClassify_OwnedByOther(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("billing/invoice", []string{"internal/billing/invoice.go"}, nil, nil)

	if class := m.Classify("auth/middleware", "internal/billing/invoice.go"); class != OwnedByOtherSpec {
		t.Errorf("expected OwnedByOtherSpec, got %d", class)
	}
}

func TestClassify_Shared(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("billing/invoice", nil, []string{"go.mod"}, nil)

	if class := m.Classify("billing/invoice", "go.mod"); class != Shared {
		t.Errorf("expected Shared, got %d", class)
	}
}

func TestClassify_Readonly(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("billing/invoice", nil, nil, []string{"internal/legacy/old.go"})

	if class := m.Classify("billing/invoice", "internal/legacy/old.go"); class != ReadOnly {
		t.Errorf("expected ReadOnly, got %d", class)
	}
}

func TestClassify_Unmanaged(t *testing.T) {
	m := NewOwnershipMap()

	if class := m.Classify("any", "some/random/file.go"); class != Unmanaged {
		t.Errorf("expected Unmanaged, got %d", class)
	}
}

func TestValidatePatch_ReadonlyRejected(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("spec-a", nil, nil, []string{"readonly.go"})

	err := ValidatePatch("spec-a", []string{"readonly.go"}, m)
	if err == nil {
		t.Fatal("expected error for readonly file")
	}
}

func TestValidatePatch_OwnedByOtherRejected(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("spec-a", []string{"owned.go"}, nil, nil)

	err := ValidatePatch("spec-b", []string{"owned.go"}, m)
	if err == nil {
		t.Fatal("expected error for file owned by other spec")
	}
}

func TestValidatePatch_OwnedAllowed(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("spec-a", []string{"my-file.go"}, nil, nil)

	err := ValidatePatch("spec-a", []string{"my-file.go"}, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePatch_SharedAllowed(t *testing.T) {
	m := NewOwnershipMap()
	m.RegisterSpec("spec-a", nil, []string{"go.mod"}, nil)

	err := ValidatePatch("spec-a", []string{"go.mod"}, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
