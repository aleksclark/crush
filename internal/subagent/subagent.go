// Package subagent implements Claude Code-compatible subagent configuration.
// Subagents are defined in Markdown files with YAML frontmatter and can be
// placed in .crush/agents/ (project) or ~/.crush/agents/ (user) directories.
package subagent

import (
	"time"
)

// Subagent represents a parsed subagent configuration file.
type Subagent struct {
	// Required fields.
	Name        string `yaml:"name" json:"name"`               // Unique identifier (lowercase letters and hyphens).
	Description string `yaml:"description" json:"description"` // Explains when to delegate to this subagent.

	// Optional fields.
	Tools           []string       `yaml:"tools,omitempty" json:"tools,omitempty"`                       // Allowed tools (inherits all if omitted).
	DisallowedTools []string       `yaml:"disallowed_tools,omitempty" json:"disallowed_tools,omitempty"` // Tools to remove from the tool list.
	Model           ModelType      `yaml:"model,omitempty" json:"model,omitempty"`                       // Model to use: sonnet, opus, haiku, inherit.
	PermissionMode  PermissionMode `yaml:"permission_mode,omitempty" json:"permission_mode,omitempty"`   // Permission handling mode.
	Skills          []string       `yaml:"skills,omitempty" json:"skills,omitempty"`                     // Skills to load into context.
	Hooks           *Hooks         `yaml:"hooks,omitempty" json:"hooks,omitempty"`                       // Lifecycle hooks.

	// Parsed from markdown body.
	SystemPrompt string `yaml:"-" json:"system_prompt,omitempty"`

	// Metadata (not from YAML).
	Path     string    `yaml:"-" json:"path,omitempty"`      // Directory containing the agent file.
	FilePath string    `yaml:"-" json:"file_path,omitempty"` // Full path to the agent file.
	Priority int       `yaml:"-" json:"priority,omitempty"`  // Resolution priority (higher = more priority).
	ModTime  time.Time `yaml:"-" json:"mod_time,omitzero"`   // Last modification time for change detection.
}

// Hooks defines lifecycle hooks for subagent execution.
type Hooks struct {
	PreToolExecution  string `yaml:"pre_tool_execution,omitempty" json:"pre_tool_execution,omitempty"`
	PostToolExecution string `yaml:"post_tool_execution,omitempty" json:"post_tool_execution,omitempty"`
}

// PermissionMode controls how permissions are handled for the subagent.
type PermissionMode string

const (
	// PermissionDefault uses standard permission checking.
	PermissionDefault PermissionMode = "default"
	// PermissionAcceptEdits auto-accepts file edits but prompts for bash.
	PermissionAcceptEdits PermissionMode = "acceptEdits"
	// PermissionDontAsk auto-denies all permission prompts.
	PermissionDontAsk PermissionMode = "dontAsk"
	// PermissionBypassPermissions skips all permission checks.
	PermissionBypassPermissions PermissionMode = "bypassPermissions"
	// PermissionYolo is an alias for bypassPermissions (skips all permission checks).
	PermissionYolo PermissionMode = "yolo"
	// PermissionPlan enables read-only exploration mode.
	PermissionPlan PermissionMode = "plan"
)

// IsValid returns true if the permission mode is a known value.
func (p PermissionMode) IsValid() bool {
	switch p {
	case "", PermissionDefault, PermissionAcceptEdits, PermissionDontAsk, PermissionBypassPermissions, PermissionYolo, PermissionPlan:
		return true
	default:
		return false
	}
}

// Normalize returns the canonical form of the permission mode.
// This converts aliases (like "yolo") to their canonical form ("bypassPermissions").
func (p PermissionMode) Normalize() PermissionMode {
	if p == PermissionYolo {
		return PermissionBypassPermissions
	}
	return p
}

// ModelType specifies which model the subagent should use.
type ModelType string

const (
	// ModelSonnet uses the sonnet-class model (typically large).
	ModelSonnet ModelType = "sonnet"
	// ModelOpus uses the opus-class model (typically large, more capable).
	ModelOpus ModelType = "opus"
	// ModelHaiku uses the haiku-class model (typically small, faster).
	ModelHaiku ModelType = "haiku"
	// ModelInherit uses the parent agent's model.
	ModelInherit ModelType = "inherit"
)

// IsValid returns true if the model type is a known value.
func (m ModelType) IsValid() bool {
	switch m {
	case "", ModelSonnet, ModelOpus, ModelHaiku, ModelInherit:
		return true
	default:
		return false
	}
}

// Clone creates a deep copy of the subagent.
func (s *Subagent) Clone() *Subagent {
	if s == nil {
		return nil
	}
	clone := *s

	// Clone slices.
	if s.Tools != nil {
		clone.Tools = make([]string, len(s.Tools))
		copy(clone.Tools, s.Tools)
	}
	if s.DisallowedTools != nil {
		clone.DisallowedTools = make([]string, len(s.DisallowedTools))
		copy(clone.DisallowedTools, s.DisallowedTools)
	}
	if s.Skills != nil {
		clone.Skills = make([]string, len(s.Skills))
		copy(clone.Skills, s.Skills)
	}

	// Clone hooks.
	if s.Hooks != nil {
		hooksCopy := *s.Hooks
		clone.Hooks = &hooksCopy
	}

	return &clone
}
