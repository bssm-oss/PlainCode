// Package skills loads agent rules and skill definitions from the project.
//
// Supported sources:
//   - AGENTS.md (CodeSpeak, Codex, OpenCode compatible)
//   - .claude/CLAUDE.md (Claude Code rules)
//   - .cursor/rules (Cursor rules)
//   - .agents/skills/<name>/SKILL.md (skill definitions)
//   - SKILL.md (root-level skills)
//
// Skills are separate from specs. A spec describes WHAT to build;
// a skill describes HOW to do specific operations (e.g., "pytest triage",
// "monorepo import fixer", "API regression checker").
package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a loaded skill definition.
type Skill struct {
	Name    string
	Path    string
	Content string
}

// ProjectRules holds all loaded agent rules and skills.
type ProjectRules struct {
	AgentsMD string   // AGENTS.md content
	ClaudeMD string   // .claude/CLAUDE.md content
	CursorRules string // .cursor/rules content
	Skills   []Skill
}

// LoadProjectRules loads all agent rules and skills from the project directory.
func LoadProjectRules(projectDir string) (*ProjectRules, error) {
	rules := &ProjectRules{}

	// Load AGENTS.md
	if data, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md")); err == nil {
		rules.AgentsMD = string(data)
	}

	// Load .claude/CLAUDE.md
	if data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "CLAUDE.md")); err == nil {
		rules.ClaudeMD = string(data)
	}

	// Load .cursor/rules
	if data, err := os.ReadFile(filepath.Join(projectDir, ".cursor", "rules")); err == nil {
		rules.CursorRules = string(data)
	}

	// Load skills from .agents/skills/
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			if data, err := os.ReadFile(skillFile); err == nil {
				rules.Skills = append(rules.Skills, Skill{
					Name:    entry.Name(),
					Path:    skillFile,
					Content: string(data),
				})
			}
		}
	}

	// Load root SKILL.md
	if data, err := os.ReadFile(filepath.Join(projectDir, "SKILL.md")); err == nil {
		rules.Skills = append(rules.Skills, Skill{
			Name:    "root",
			Path:    filepath.Join(projectDir, "SKILL.md"),
			Content: string(data),
		})
	}

	return rules, nil
}

// CombinedRules returns all agent rules as a single string.
func (r *ProjectRules) CombinedRules() string {
	var parts []string
	if r.AgentsMD != "" {
		parts = append(parts, "# AGENTS.md\n\n"+r.AgentsMD)
	}
	if r.ClaudeMD != "" {
		parts = append(parts, "# .claude/CLAUDE.md\n\n"+r.ClaudeMD)
	}
	if r.CursorRules != "" {
		parts = append(parts, "# .cursor/rules\n\n"+r.CursorRules)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// SkillNames returns the names of all loaded skills.
func (r *ProjectRules) SkillNames() []string {
	names := make([]string, len(r.Skills))
	for i, s := range r.Skills {
		names[i] = s.Name
	}
	return names
}
