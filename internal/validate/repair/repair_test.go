package repair

import (
	"strings"
	"testing"
)

func TestClassifyAndPrompt(t *testing.T) {
	failures := Classify("test output", false, 0.4, 0.8, []string{"readonly file"})
	if len(failures) != 2 {
		t.Fatalf("expected 2 failures, got %d", len(failures))
	}
	if failures[0].Kind != OwnershipViolation || failures[1].Kind != TestFailure {
		t.Fatalf("unexpected failures: %+v", failures)
	}

	prompt := BuildRepairPrompt(RepairContext{
		OriginalPrompt: "original",
		Failures:       failures,
		Attempt:        1,
		MaxAttempts:    3,
	})
	if !strings.Contains(prompt, "Repair Attempt 2/3") {
		t.Fatalf("unexpected prompt header: %q", prompt)
	}
	if !strings.Contains(prompt, "readonly file") || !strings.Contains(prompt, "test output") {
		t.Fatalf("expected failure details in prompt: %q", prompt)
	}
}
