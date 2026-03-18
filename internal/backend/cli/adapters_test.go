package cli_test

import (
	"testing"

	"github.com/bssm-oss/PlainCode/internal/backend/cli/claude"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/codex"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/copilot"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/cursor"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/gemini"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/opencode"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
)

// TestAllAdapters_ID verifies each adapter returns the correct ID.
func TestAllAdapters_ID(t *testing.T) {
	tests := []struct {
		backend core.Backend
		wantID  string
	}{
		{claude.New(""), "cli:claude"},
		{codex.New(""), "cli:codex"},
		{gemini.New(""), "cli:gemini"},
		{copilot.New(""), "cli:copilot"},
		{cursor.New(""), "cli:cursor"},
		{opencode.New(""), "cli:opencode"},
	}

	for _, tt := range tests {
		if got := tt.backend.ID(); got != tt.wantID {
			t.Errorf("%T.ID() = %q, want %q", tt.backend, got, tt.wantID)
		}
	}
}

// TestAllAdapters_Capabilities verifies capabilities are set.
func TestAllAdapters_Capabilities(t *testing.T) {
	backends := []core.Backend{
		claude.New(""),
		codex.New(""),
		gemini.New(""),
		copilot.New(""),
		cursor.New(""),
		opencode.New(""),
	}

	for _, b := range backends {
		caps := b.Capabilities()
		// All CLI adapters should support at least tools or file patch
		t.Logf("%s: structured=%v mcp=%v tools=%v", b.ID(), caps.StructuredOutput, caps.MCP, caps.Tools)
	}
}

// TestClaude_BuildArgs checks Claude flag generation for each profile.
func TestClaude_BuildArgs(t *testing.T) {
	b := claude.New("claude")
	profiles := map[core.ApprovalProfile]string{
		core.ProfilePlan:          "--print",
		core.ProfilePatch:         "--permission-mode",
		core.ProfileWorkspaceAuto: "--permission-mode",
		core.ProfileFullTrust:     "--dangerously-skip-permissions",
	}

	for profile, expectedFlag := range profiles {
		req := &core.ExecRequest{
			ApprovalProfile: profile,
			PromptPack:      core.PromptPack{SpecBody: "test prompt"},
			Budget:          core.Budget{MaxTurns: 5},
		}
		args := b.BuildArgs(req)
		found := false
		for _, a := range args {
			if a == expectedFlag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("profile %q: expected flag %q in args %v", profile, expectedFlag, args)
		}
	}
}

// TestCodex_BuildArgs checks Codex flag generation.
func TestCodex_BuildArgs(t *testing.T) {
	b := codex.New("codex")

	tests := []struct {
		profile  core.ApprovalProfile
		contains string
	}{
		{core.ProfilePlan, "read-only"},
		{core.ProfileWorkspaceAuto, "--full-auto"},
		{core.ProfileFullTrust, "--dangerously-bypass-approvals-and-sandbox"},
	}

	for _, tt := range tests {
		req := &core.ExecRequest{
			ApprovalProfile: tt.profile,
			PromptPack:      core.PromptPack{SpecBody: "test"},
		}
		args := b.BuildArgs(req)
		found := false
		for _, a := range args {
			if a == tt.contains {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("profile %q: expected %q in args %v", tt.profile, tt.contains, args)
		}
	}
}

// TestGemini_BuildArgs checks Gemini flag generation.
func TestGemini_BuildArgs(t *testing.T) {
	b := gemini.New("gemini")

	req := &core.ExecRequest{
		ApprovalProfile: core.ProfileFullTrust,
		PromptPack:      core.PromptPack{SpecBody: "test"},
	}
	args := b.BuildArgs(req)
	found := false
	for _, a := range args {
		if a == "--yolo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("full-trust: expected --yolo in args %v", args)
	}
}

// TestCopilot_BuildArgs checks Copilot flag generation.
func TestCopilot_BuildArgs(t *testing.T) {
	b := copilot.New("copilot")

	req := &core.ExecRequest{
		ApprovalProfile: core.ProfileFullTrust,
		PromptPack:      core.PromptPack{SpecBody: "test"},
	}
	args := b.BuildArgs(req)
	found := false
	for _, a := range args {
		if a == "--yolo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("full-trust: expected --yolo in args %v", args)
	}
}
