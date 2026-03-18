// Package mcp provides Model Context Protocol server registration and bridging.
//
// MCP is a first-class citizen in Forge. Backends that support MCP
// (Claude Code, Codex, Gemini CLI, Copilot, OpenCode) can connect to
// MCP servers configured in plaincode.yaml.
//
// This package manages:
//   - MCP server registration from config
//   - MCP config generation for CLI backends that accept --mcp-config
//   - MCP capability discovery
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerConfig describes an MCP server configuration.
type ServerConfig struct {
	// Name is the server identifier.
	Name string `json:"name" yaml:"name"`

	// Command is the command to start the server.
	Command string `json:"command" yaml:"command"`

	// Args are command-line arguments for the server.
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// Env contains environment variables for the server.
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

// Registry manages MCP server configurations.
type Registry struct {
	servers map[string]ServerConfig
}

// NewRegistry creates an empty MCP registry.
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]ServerConfig),
	}
}

// Register adds an MCP server configuration.
func (r *Registry) Register(cfg ServerConfig) {
	r.servers[cfg.Name] = cfg
}

// Get returns a server config by name.
func (r *Registry) Get(name string) (ServerConfig, bool) {
	cfg, ok := r.servers[name]
	return cfg, ok
}

// List returns all registered server names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	return names
}

// GenerateConfigFile writes an MCP config file in the format expected
// by Claude Code's --mcp-config flag.
// Format: {"mcpServers": {"name": {"command": "...", "args": [...]}}}
func (r *Registry) GenerateConfigFile(dir string) (string, error) {
	type mcpEntry struct {
		Command string            `json:"command"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
	}

	config := struct {
		MCPServers map[string]mcpEntry `json:"mcpServers"`
	}{
		MCPServers: make(map[string]mcpEntry),
	}

	for name, srv := range r.servers {
		config.MCPServers[name] = mcpEntry{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling MCP config: %w", err)
	}

	path := filepath.Join(dir, ".plaincode", "mcp-config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	return path, nil
}
