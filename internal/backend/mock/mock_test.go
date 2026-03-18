package mock

import (
	"context"
	"testing"
	"time"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

func TestMockBackend_ID(t *testing.T) {
	b := New("mock:test")
	if b.ID() != "mock:test" {
		t.Errorf("expected ID 'mock:test', got %q", b.ID())
	}
}

func TestMockBackend_Capabilities(t *testing.T) {
	b := New("mock:test")
	caps := b.Capabilities()
	if !caps.StructuredOutput {
		t.Error("expected StructuredOutput to be true")
	}
	if !caps.CostTracking {
		t.Error("expected CostTracking to be true")
	}
}

func TestMockBackend_Execute(t *testing.T) {
	b := New("mock:test")
	b.SetResponse("build-1", "package main\n\nfunc Hello() string { return \"hi\" }\n")

	req := &core.ExecRequest{
		BuildID: "build-1",
		PromptPack: core.PromptPack{
			SpecBody: "Generate a hello function",
		},
	}

	result, err := b.Execute(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(result.Patches))
	}

	if result.Usage.Turns != 1 {
		t.Errorf("expected 1 turn, got %d", result.Usage.Turns)
	}
}

func TestMockBackend_HealthCheck(t *testing.T) {
	b := New("mock:test")
	if err := b.HealthCheck(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockBackend_WithDelay(t *testing.T) {
	b := New("mock:test").WithDelay(10 * time.Millisecond)

	req := &core.ExecRequest{
		BuildID:    "build-1",
		PromptPack: core.PromptPack{SpecBody: "test"},
	}

	start := time.Now()
	_, err := b.Execute(context.Background(), req, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected at least 10ms delay, got %v", elapsed)
	}
}

func TestMockBackend_ContextCancel(t *testing.T) {
	b := New("mock:test").WithDelay(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &core.ExecRequest{
		BuildID:    "build-1",
		PromptPack: core.PromptPack{SpecBody: "test"},
	}

	_, err := b.Execute(ctx, req, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
