package speccheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	pruntime "github.com/bssm-oss/PlainCode/internal/runtime"
	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

func TestParseHTTPOracles_KoreanPatterns(t *testing.T) {
	text := `
- go test ./... 가 통과한다.
- GET /api/health 는 200 과 {"status":"good"} 를 반환한다.
- GET /api/solve?n=3 의 moveCount 는 7 이다.
- GET /api/solve?n=3 의 moves 길이는 7 이다.
- GET /api/solve?n=0 은 400 이다.
`

	oracles, ignored := ParseHTTPOracles(text)
	if len(oracles) != 4 {
		t.Fatalf("expected 4 parsed oracles, got %d", len(oracles))
	}
	if len(ignored) != 1 {
		t.Fatalf("expected 1 ignored line, got %d", len(ignored))
	}
	if oracles[0].Kind != "http_status_json" {
		t.Fatalf("unexpected first oracle: %+v", oracles[0])
	}
	if oracles[1].Kind != "http_json_field_value" {
		t.Fatalf("unexpected second oracle: %+v", oracles[1])
	}
	if oracles[2].Kind != "http_json_field_length" {
		t.Fatalf("unexpected third oracle: %+v", oracles[2])
	}
	if oracles[3].ExpectStatus != 400 {
		t.Fatalf("unexpected fourth oracle: %+v", oracles[3])
	}
}

func TestChecker_Run_UsesCommandAndHTTPOracles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "good"})
		case "/api/solve":
			if r.URL.Query().Get("n") == "0" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"moveCount": 7,
				"moves":     []int{1, 2, 3, 4, 5, 6, 7},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	projectDir := t.TempDir()
	testScriptPath := filepath.Join(projectDir, "test-ok.sh")
	if err := os.WriteFile(testScriptPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write test script: %v", err)
	}
	runtimeScriptPath := filepath.Join(projectDir, "run-ok.sh")
	if err := os.WriteFile(runtimeScriptPath, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatalf("write runtime script: %v", err)
	}

	spec := &ast.Spec{
		ID:       "demo/spec",
		Language: "go",
		Tests: ast.TestConfig{
			Command: testScriptPath,
		},
		Runtime: ast.RuntimeConfig{
			DefaultMode: "process",
			Process: ast.ProcessRuntime{
				Command:        runtimeScriptPath,
				HealthcheckURL: server.URL + "/api/health",
			},
		},
		Body: ast.SpecBody{
			TestOracles: `
- GET /api/health 는 200 과 {"status":"good"} 를 반환한다.
- GET /api/solve?n=3 의 moveCount 는 7 이다.
- GET /api/solve?n=3 의 moves 길이는 7 이다.
- GET /api/solve?n=0 은 400 이다.
`,
		},
	}

	runtimeManager := pruntime.NewManager(projectDir, filepath.Join(projectDir, ".plaincode"))
	checker := New(runtimeManager)
	result, err := checker.Run(context.Background(), spec, projectDir, Options{
		SkipCommand: false,
	})
	if err != nil {
		t.Fatalf("checker run: %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected result to pass, got %+v", result)
	}
	if result.CommandResult == nil || !result.CommandResult.Passed {
		t.Fatalf("expected command result to pass, got %+v", result.CommandResult)
	}
	if len(result.Oracles) != 4 {
		t.Fatalf("expected 4 oracle results, got %d", len(result.Oracles))
	}
}
