package cli

import (
	"testing"

	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

func TestParseFileBlocks(t *testing.T) {
	output := `Some preamble text.

--- FILE: internal/greeter/greeter.go ---
package greeter

func Greet(name string) string {
	return "Hello, " + name + "!"
}
--- END FILE ---

Some middle text.

--- FILE: internal/greeter/greeter_test.go ---
package greeter

import "testing"

func TestGreet(t *testing.T) {
	if Greet("Alice") != "Hello, Alice!" {
		t.Fail()
	}
}
--- END FILE ---
`

	ops := ParseFileBlocks(output)
	if len(ops) != 2 {
		t.Fatalf("expected 2 patches, got %d", len(ops))
	}

	p1, ok := ops[0].(core.WriteFile)
	if !ok {
		t.Fatalf("expected WriteFile, got %T", ops[0])
	}
	if p1.FilePath != "internal/greeter/greeter.go" {
		t.Errorf("expected path 'internal/greeter/greeter.go', got %q", p1.FilePath)
	}
	if len(p1.Content) == 0 {
		t.Error("expected non-empty content")
	}

	p2, ok := ops[1].(core.WriteFile)
	if !ok {
		t.Fatalf("expected WriteFile, got %T", ops[1])
	}
	if p2.FilePath != "internal/greeter/greeter_test.go" {
		t.Errorf("expected path 'internal/greeter/greeter_test.go', got %q", p2.FilePath)
	}
}

func TestParseFileBlocks_Empty(t *testing.T) {
	ops := ParseFileBlocks("just some text with no file blocks")
	if len(ops) != 0 {
		t.Errorf("expected 0 patches, got %d", len(ops))
	}
}
