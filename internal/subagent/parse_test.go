package subagent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		check       func(t *testing.T, s *Subagent)
	}{
		{
			name: "valid basic subagent",
			content: `---
name: code-reviewer
description: Reviews code for quality and best practices
---
Review the code carefully.
`,
			check: func(t *testing.T, s *Subagent) {
				require.Equal(t, "code-reviewer", s.Name)
				require.Equal(t, "Reviews code for quality and best practices", s.Description)
				require.Equal(t, "Review the code carefully.", s.SystemPrompt)
			},
		},
		{
			name: "valid subagent with all fields",
			content: `---
name: security-auditor
description: Audits code for security vulnerabilities
tools:
  - Glob
  - Grep
  - View
disallowed_tools:
  - Bash
model: sonnet
permission_mode: plan
skills:
  - security-patterns
---
You are a security auditor. Review the code for vulnerabilities.
`,
			check: func(t *testing.T, s *Subagent) {
				require.Equal(t, "security-auditor", s.Name)
				require.Equal(t, "Audits code for security vulnerabilities", s.Description)
				require.Equal(t, []string{"Glob", "Grep", "View"}, s.Tools)
				require.Equal(t, []string{"Bash"}, s.DisallowedTools)
				require.Equal(t, ModelSonnet, s.Model)
				require.Equal(t, PermissionPlan, s.PermissionMode)
				require.Equal(t, []string{"security-patterns"}, s.Skills)
				require.Contains(t, s.SystemPrompt, "security auditor")
			},
		},
		{
			name: "valid subagent with hooks",
			content: `---
name: test-runner
description: Runs tests
hooks:
  pre_tool_execution: echo "starting"
  post_tool_execution: echo "done"
---
Run the tests.
`,
			check: func(t *testing.T, s *Subagent) {
				require.NotNil(t, s.Hooks)
				require.Equal(t, `echo "starting"`, s.Hooks.PreToolExecution)
				require.Equal(t, `echo "done"`, s.Hooks.PostToolExecution)
			},
		},
		{
			name:        "missing frontmatter",
			content:     "No frontmatter here",
			wantErr:     true,
			errContains: "no YAML frontmatter",
		},
		{
			name: "unclosed frontmatter",
			content: `---
name: test
description: test
No closing ---`,
			wantErr:     true,
			errContains: "unclosed frontmatter",
		},
		{
			name: "invalid yaml",
			content: `---
name: [invalid
description: test
---
Body`,
			wantErr:     true,
			errContains: "parsing YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := ParseContent(tt.content, "/test/path.md")
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, s)
			require.Equal(t, "/test/path.md", s.FilePath)
			require.Equal(t, "/test", s.Path)

			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

func TestSubagentValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		subagent    Subagent
		wantErr     bool
		errContains string
	}{
		{
			name: "valid subagent",
			subagent: Subagent{
				Name:        "code-reviewer",
				Description: "Reviews code",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			subagent: Subagent{
				Description: "Reviews code",
			},
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name: "missing description",
			subagent: Subagent{
				Name: "test",
			},
			wantErr:     true,
			errContains: "description is required",
		},
		{
			name: "invalid name - uppercase",
			subagent: Subagent{
				Name:        "CodeReviewer",
				Description: "Reviews code",
			},
			wantErr:     true,
			errContains: "lowercase alphanumeric",
		},
		{
			name: "invalid name - starts with number",
			subagent: Subagent{
				Name:        "1-reviewer",
				Description: "Reviews code",
			},
			wantErr:     true,
			errContains: "lowercase alphanumeric",
		},
		{
			name: "invalid name - consecutive hyphens",
			subagent: Subagent{
				Name:        "code--reviewer",
				Description: "Reviews code",
			},
			wantErr:     true,
			errContains: "lowercase alphanumeric",
		},
		{
			name: "invalid name - trailing hyphen",
			subagent: Subagent{
				Name:        "code-reviewer-",
				Description: "Reviews code",
			},
			wantErr:     true,
			errContains: "lowercase alphanumeric",
		},
		{
			name: "name too long",
			subagent: Subagent{
				Name:        "this-is-a-very-long-name-that-exceeds-the-maximum-allowed-length-for-names",
				Description: "Reviews code",
			},
			wantErr:     true,
			errContains: "exceeds",
		},
		{
			name: "invalid model",
			subagent: Subagent{
				Name:        "test",
				Description: "Test",
				Model:       "invalid-model",
			},
			wantErr:     true,
			errContains: "invalid model",
		},
		{
			name: "invalid permission mode",
			subagent: Subagent{
				Name:           "test",
				Description:    "Test",
				PermissionMode: "invalid-mode",
			},
			wantErr:     true,
			errContains: "invalid permission_mode",
		},
		{
			name: "valid with all models",
			subagent: Subagent{
				Name:        "test",
				Description: "Test",
				Model:       ModelOpus,
			},
			wantErr: false,
		},
		{
			name: "valid with all permission modes",
			subagent: Subagent{
				Name:           "test",
				Description:    "Test",
				PermissionMode: PermissionBypassPermissions,
			},
			wantErr: false,
		},
		{
			name: "valid with yolo permission mode",
			subagent: Subagent{
				Name:           "test",
				Description:    "Test",
				PermissionMode: PermissionYolo,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.subagent.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestModelTypeIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model ModelType
		valid bool
	}{
		{ModelSonnet, true},
		{ModelOpus, true},
		{ModelHaiku, true},
		{ModelInherit, true},
		{"", true}, // Empty is valid (defaults to inherit).
		{"invalid", false},
		{"SONNET", false}, // Case-sensitive.
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.valid, tt.model.IsValid())
		})
	}
}

func TestPermissionModeIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode  PermissionMode
		valid bool
	}{
		{PermissionDefault, true},
		{PermissionAcceptEdits, true},
		{PermissionDontAsk, true},
		{PermissionBypassPermissions, true},
		{PermissionPlan, true},
		{"", true}, // Empty is valid (defaults to default).
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.valid, tt.mode.IsValid())
		})
	}
}

func TestSubagentClone(t *testing.T) {
	t.Parallel()

	original := &Subagent{
		Name:            "test",
		Description:     "Test subagent",
		Tools:           []string{"Glob", "Grep"},
		DisallowedTools: []string{"Bash"},
		Skills:          []string{"skill1"},
		Hooks: &Hooks{
			PreToolExecution:  "echo pre",
			PostToolExecution: "echo post",
		},
	}

	clone := original.Clone()

	// Verify values are equal.
	require.Equal(t, original.Name, clone.Name)
	require.Equal(t, original.Description, clone.Description)
	require.Equal(t, original.Tools, clone.Tools)
	require.Equal(t, original.DisallowedTools, clone.DisallowedTools)
	require.Equal(t, original.Skills, clone.Skills)
	require.Equal(t, original.Hooks, clone.Hooks)

	// Modify clone and verify original is unchanged.
	clone.Name = "modified"
	clone.Tools[0] = "Modified"
	clone.Hooks.PreToolExecution = "modified"

	require.Equal(t, "test", original.Name)
	require.Equal(t, "Glob", original.Tools[0])
	require.Equal(t, "echo pre", original.Hooks.PreToolExecution)
}

func TestSubagentCloneNil(t *testing.T) {
	t.Parallel()

	var s *Subagent
	require.Nil(t, s.Clone())
}

func TestParse(t *testing.T) {
	t.Parallel()

	// Create a temp file.
	tmpDir := t.TempDir()
	agentFile := filepath.Join(tmpDir, "test-agent.md")

	content := `---
name: test-agent
description: A test agent
model: haiku
---
Test system prompt.
`
	err := os.WriteFile(agentFile, []byte(content), 0o644)
	require.NoError(t, err)

	s, err := Parse(agentFile)
	require.NoError(t, err)
	require.NotNil(t, s)

	require.Equal(t, "test-agent", s.Name)
	require.Equal(t, "A test agent", s.Description)
	require.Equal(t, ModelHaiku, s.Model)
	require.Equal(t, "Test system prompt.", s.SystemPrompt)
	require.Equal(t, agentFile, s.FilePath)
	require.Equal(t, tmpDir, s.Path)
	require.False(t, s.ModTime.IsZero())
}

func TestParseNonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := Parse("/nonexistent/path.md")
	require.Error(t, err)
	require.Contains(t, err.Error(), "reading file")
}

func TestValidateTools(t *testing.T) {
	t.Parallel()

	available := []string{"Glob", "Grep", "View", "Bash"}

	tests := []struct {
		name        string
		tools       []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "all valid tools",
			tools:   []string{"Glob", "Grep"},
			wantErr: false,
		},
		{
			name:    "empty tools",
			tools:   nil,
			wantErr: false,
		},
		{
			name:        "unknown tool",
			tools:       []string{"Glob", "Unknown"},
			wantErr:     true,
			errContains: "Unknown",
		},
		{
			name:        "multiple unknown tools",
			tools:       []string{"Unknown1", "Unknown2"},
			wantErr:     true,
			errContains: "Unknown1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Subagent{Tools: tt.tools}
			err := s.ValidateTools(available)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSplitFrontmatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		wantFront   string
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "basic frontmatter",
			content: `---
key: value
---
Body content`,
			wantFront: "key: value",
			wantBody:  "Body content",
		},
		{
			name:      "frontmatter with windows line endings",
			content:   "---\r\nkey: value\r\n---\r\nBody",
			wantFront: "key: value",
			wantBody:  "Body",
		},
		{
			name: "empty body",
			content: `---
key: value
---
`,
			wantFront: "key: value",
			wantBody:  "",
		},
		{
			name:        "no frontmatter",
			content:     "Just body content",
			wantErr:     true,
			errContains: "no YAML frontmatter",
		},
		{
			name: "unclosed frontmatter",
			content: `---
key: value
No closing`,
			wantErr:     true,
			errContains: "unclosed frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			front, body, err := splitFrontmatter(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantFront, front)
			require.Equal(t, tt.wantBody, body)
		})
	}
}

func TestParseContentWithModTime(t *testing.T) {
	t.Parallel()

	content := `---
name: test
description: Test
---
Body`

	modTime := time.Now().Add(-time.Hour)
	s, err := ParseContent(content, "/test.md", modTime)
	require.NoError(t, err)
	require.Equal(t, modTime.Unix(), s.ModTime.Unix())
}
