package app

import (
	"context"

	"github.com/bssm-oss/PlainCode/internal/backend/cli/claude"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/codex"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/copilot"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/cursor"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/gemini"
	"github.com/bssm-oss/PlainCode/internal/backend/cli/opencode"
	"github.com/bssm-oss/PlainCode/internal/backend/core"
	"github.com/bssm-oss/PlainCode/internal/backend/mock"
	"github.com/bssm-oss/PlainCode/internal/config"
)

// BuildRegistry creates a backend registry from the project config.
// It maps provider kinds to their concrete adapter implementations.
func BuildRegistry(cfg *config.ProjectConfig) *core.Registry {
	registry := core.NewRegistry()

	for id, p := range cfg.Providers {
		var backend core.Backend

		switch p.Kind {
		case "cli-claude":
			backend = claude.New(p.Binary)
		case "cli-codex":
			backend = codex.New(p.Binary)
		case "cli-gemini":
			backend = gemini.New(p.Binary)
		case "cli-copilot":
			backend = copilot.New(p.Binary)
		case "cli-cursor":
			backend = cursor.New(p.Binary)
		case "cli-opencode":
			backend = opencode.New(p.Binary)
		case "mock":
			backend = mock.New(id)
		default:
			// TODO: Add API backends (openai, anthropic, gemini-api)
			continue
		}

		// Wrap to use the config key as ID (e.g., "cli:claude")
		registry.Register(&idOverride{backend: backend, id: id})
	}

	// Always register a mock fallback if default backend isn't available
	if _, err := registry.Get(cfg.Defaults.Backend); err != nil {
		fallback := mock.New(cfg.Defaults.Backend)
		registry.Register(fallback)
	}

	return registry
}

// idOverride wraps a backend to override its ID with the config key.
type idOverride struct {
	backend core.Backend
	id      string
}

func (o *idOverride) ID() string                       { return o.id }
func (o *idOverride) Capabilities() core.CapabilitySet { return o.backend.Capabilities() }
func (o *idOverride) Execute(ctx context.Context, req *core.ExecRequest, sink core.EventSink) (*core.ExecResult, error) {
	return o.backend.Execute(ctx, req, sink)
}
func (o *idOverride) HealthCheck(ctx context.Context) error { return o.backend.HealthCheck(ctx) }
