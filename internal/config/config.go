// Package config loads and validates plaincode.yaml project configuration.
//
// The configuration hierarchy is:
//   1. plaincode.yaml (project root)
//   2. ~/.plaincode/config.yaml (user global)
//   3. Environment variables (PLAINCODE_*)
//   4. CLI flags (highest priority)
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig is the top-level plaincode.yaml structure.
type ProjectConfig struct {
	Version  int             `yaml:"version"`
	Project  ProjectSettings `yaml:"project"`
	Defaults DefaultSettings `yaml:"defaults"`
	Providers map[string]ProviderConfig `yaml:"providers,omitempty"`
}

// ProjectSettings defines project-level paths and defaults.
type ProjectSettings struct {
	SpecDir         string `yaml:"spec_dir"`
	StateDir        string `yaml:"state_dir"`
	DefaultLanguage string `yaml:"default_language"`
}

// DefaultSettings defines default build behavior.
type DefaultSettings struct {
	Backend    string `yaml:"backend"`
	Approval   string `yaml:"approval"`
	RetryLimit int    `yaml:"retry_limit"`
}

// ProviderConfig describes a single backend provider.
type ProviderConfig struct {
	Kind   string `yaml:"kind"`
	Model  string `yaml:"model,omitempty"`
	Binary string `yaml:"binary,omitempty"`
	APIKey string `yaml:"api_key,omitempty"` // or env var reference
}

// DefaultProjectConfig returns sensible defaults for a new project.
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Version: 1,
		Project: ProjectSettings{
			SpecDir:         "spec",
			StateDir:        ".plaincode",
			DefaultLanguage: "go",
		},
		Defaults: DefaultSettings{
			Backend:    "openai:gpt-4o",
			Approval:   "patch",
			RetryLimit: 3,
		},
	}
}

// Load reads plaincode.yaml from the given directory.
// Returns DefaultProjectConfig if the file doesn't exist.
func Load(dir string) (*ProjectConfig, error) {
	path := filepath.Join(dir, "plaincode.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultProjectConfig()
			return &cfg, nil
		}
		return nil, fmt.Errorf("reading plaincode.yaml: %w", err)
	}

	var cfg ProjectConfig
	dec := yaml.NewDecoder(nil)
	_ = dec // NOTE: yaml.v3 doesn't support KnownFields on NewDecoder from bytes directly
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing plaincode.yaml: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plaincode.yaml: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration is internally consistent.
func (c *ProjectConfig) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", c.Version)
	}
	if c.Project.SpecDir == "" {
		return fmt.Errorf("project.spec_dir must not be empty")
	}
	if c.Project.StateDir == "" {
		return fmt.Errorf("project.state_dir must not be empty")
	}
	return nil
}

// WriteDefault creates a plaincode.yaml with default settings.
func WriteDefault(dir string) error {
	cfg := DefaultProjectConfig()
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	path := filepath.Join(dir, "plaincode.yaml")
	return os.WriteFile(path, data, 0644)
}
