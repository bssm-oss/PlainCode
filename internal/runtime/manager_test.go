package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bssm-oss/PlainCode/internal/spec/ast"
)

func TestStoreSaveLoadListDelete(t *testing.T) {
	store := NewStore(t.TempDir())
	state := &State{
		SpecID:    "hello/service",
		Mode:      "process",
		PID:       123,
		Command:   []string{"sleep", "30"},
		StartedAt: time.Now(),
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.AppendEvent(state.SpecID, Event{Kind: "start_requested", Message: "requested"}); err != nil {
		t.Fatalf("append event: %v", err)
	}

	loaded, err := store.Load(state.SpecID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SpecID != state.SpecID || loaded.PID != state.PID {
		t.Fatalf("unexpected loaded state: %+v", loaded)
	}

	states, err := store.ListAll()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(states) != 1 || states[0].SpecID != state.SpecID {
		t.Fatalf("unexpected list output: %+v", states)
	}
	events, err := store.ReadEvents(state.SpecID, 10)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 || events[0].Kind != "start_requested" {
		t.Fatalf("unexpected events: %+v", events)
	}

	if err := store.Delete(state.SpecID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Load(state.SpecID); err == nil {
		t.Fatalf("expected state file to be removed, got %v", err)
	}
}

func TestManagerStartStatusStopProcess(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, ".plaincode")
	manager := NewManager(projectDir, stateDir)
	scriptPath := filepath.Join(projectDir, "run-process.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho process-ready\nsleep 30\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	spec := &ast.Spec{
		ID:       "runtime/process",
		Language: "go",
		Runtime: ast.RuntimeConfig{
			DefaultMode: "process",
			Process: ast.ProcessRuntime{
				Command: scriptPath,
			},
		},
	}

	status, err := manager.Start(context.Background(), spec, StartOptions{
		HealthTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if status.Mode != "process" || status.Status != StatusRunning {
		t.Fatalf("unexpected start status: %+v", status)
	}
	if status.PID == 0 {
		t.Fatalf("expected pid to be populated: %+v", status)
	}
	if _, err := os.Stat(status.LogPath); err != nil {
		t.Fatalf("expected log path to exist: %v", err)
	}
	var logText string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		logData, err := os.ReadFile(status.LogPath)
		if err != nil {
			t.Fatalf("read runtime log: %v", err)
		}
		logText = string(logData)
		if strings.Contains(logText, "process-ready") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(logText, "process-ready") {
		t.Fatalf("expected process output in log, got:\n%s", logText)
	}

	statuses, err := manager.Status(context.Background(), spec.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if statuses.Status != StatusRunning {
		t.Fatalf("expected running status, got %+v", statuses)
	}

	stopped, err := manager.Stop(context.Background(), spec.ID)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if processExists(status.PID) {
		t.Fatalf("expected pid %d to be stopped", status.PID)
	}
	if stopped.Status != StatusStopped {
		t.Fatalf("expected stopped status, got %+v", stopped)
	}
	events, err := manager.store.ReadEvents(spec.ID, 20)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	var kinds []string
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}
	for _, want := range []string{"start_requested", "process_spawned", "start_succeeded", "stop_requested", "stop_succeeded"} {
		if !containsString(kinds, want) {
			t.Fatalf("expected event %q in %v", want, kinds)
		}
	}
}

func TestManagerStartFailsWhenProcessExitsImmediately(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, ".plaincode")
	manager := NewManager(projectDir, stateDir)
	spec := &ast.Spec{
		ID:       "runtime/fail-fast",
		Language: "go",
		Runtime: ast.RuntimeConfig{
			DefaultMode: "process",
			Process: ast.ProcessRuntime{
				Command: "false",
			},
		},
	}

	_, err := manager.Start(context.Background(), spec, StartOptions{
		HealthTimeout: 1 * time.Second,
	})
	if err == nil {
		t.Fatal("expected start to fail when process exits immediately")
	}
	events, readErr := manager.store.ReadEvents(spec.ID, 20)
	if readErr != nil {
		t.Fatalf("read events: %v", readErr)
	}
	var kinds []string
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}
	if !containsString(kinds, "start_failed") {
		t.Fatalf("expected start_failed event, got %v", kinds)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
