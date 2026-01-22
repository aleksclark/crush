package subagent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoverPaths(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("project")
	userDir := filepath.Join("home", "user", ".config", "crush")

	paths := DiscoverPaths(projectDir, userDir, nil)

	// Should have 4 paths: project/.crush/agents, project/.claude/agents, userDir/agents, ~/.claude/agents
	require.Len(t, paths, 4)
	require.Equal(t, filepath.Join(projectDir, ".crush", "agents"), paths[0])
	require.Equal(t, filepath.Join(projectDir, ".claude", "agents"), paths[1])
	require.Equal(t, filepath.Join(userDir, "agents"), paths[2])
	// Last path is ~/.claude/agents
	home, _ := os.UserHomeDir()
	require.Equal(t, filepath.Join(home, ".claude", "agents"), paths[3])
}

func TestDiscoverPathsEmptyWorkingDir(t *testing.T) {
	t.Parallel()

	userDir := filepath.Join("home", "user", ".config", "crush")

	paths := DiscoverPaths("", userDir, nil)

	// Should have 2 paths: userDir/agents and ~/.claude/agents
	require.Len(t, paths, 2)
	require.Equal(t, filepath.Join(userDir, "agents"), paths[0])
	home, _ := os.UserHomeDir()
	require.Equal(t, filepath.Join(home, ".claude", "agents"), paths[1])
}

func TestDiscoverPathsEmptyUserDir(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("project")

	paths := DiscoverPaths(projectDir, "", nil)

	// Should have 3 paths: project/.crush/agents, project/.claude/agents, ~/.claude/agents
	require.Len(t, paths, 3)
	require.Equal(t, filepath.Join(projectDir, ".crush", "agents"), paths[0])
	require.Equal(t, filepath.Join(projectDir, ".claude", "agents"), paths[1])
	home, _ := os.UserHomeDir()
	require.Equal(t, filepath.Join(home, ".claude", "agents"), paths[2])
}

func TestDiscoverPathsWithAdditionalPaths(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("project")
	userDir := filepath.Join("home", "user", ".config", "crush")
	additionalPaths := []string{"/custom/agents", "/another/path"}

	paths := DiscoverPaths(projectDir, userDir, additionalPaths)

	// Should have 6 paths: 2 additional + 2 project + 1 user + 1 home/.claude/agents
	require.Len(t, paths, 6)
	// Additional paths have highest priority.
	require.Equal(t, "/custom/agents", paths[0])
	require.Equal(t, "/another/path", paths[1])
	// Then project-level.
	require.Equal(t, filepath.Join(projectDir, ".crush", "agents"), paths[2])
	require.Equal(t, filepath.Join(projectDir, ".claude", "agents"), paths[3])
	// Then user-level.
	require.Equal(t, filepath.Join(userDir, "agents"), paths[4])
	// Then ~/.claude/agents.
	home, _ := os.UserHomeDir()
	require.Equal(t, filepath.Join(home, ".claude", "agents"), paths[5])
}

func TestDiscover(t *testing.T) {
	t.Parallel()

	// Create temp directories.
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project", ".crush", "agents")
	userDir := filepath.Join(tmpDir, "user", "agents")

	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.MkdirAll(userDir, 0o755))

	// Create test agents.
	projectAgent := `---
name: project-agent
description: Project level agent
model: sonnet
---
Project agent prompt.
`
	userAgent := `---
name: user-agent
description: User level agent
model: haiku
---
User agent prompt.
`
	// Agent in project that shadows user agent.
	shadowAgent := `---
name: shadow-agent
description: Project shadow
---
Shadow prompt.
`
	userShadowAgent := `---
name: shadow-agent
description: User shadow (should be ignored)
---
User shadow prompt.
`

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "project-agent.md"), []byte(projectAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "user-agent.md"), []byte(userAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "shadow-agent.md"), []byte(shadowAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "shadow-agent.md"), []byte(userShadowAgent), 0o644))

	// Discover agents.
	paths := []string{projectDir, userDir}
	agents := Discover(paths)

	// Should find 3 agents: project-agent, user-agent, shadow-agent (project version).
	require.Len(t, agents, 3)

	// Find the shadow agent and verify it's the project version.
	var shadow *Subagent
	for _, a := range agents {
		if a.Name == "shadow-agent" {
			shadow = a
			break
		}
	}
	require.NotNil(t, shadow)
	require.Equal(t, "Project shadow", shadow.Description)
}

func TestDiscoverEmptyPaths(t *testing.T) {
	t.Parallel()

	agents := Discover(nil)
	require.Empty(t, agents)
}

func TestDiscoverNonExistentPath(t *testing.T) {
	t.Parallel()

	agents := Discover([]string{"/nonexistent/path"})
	require.Empty(t, agents)
}

func TestDiscoverInvalidAgent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	// Valid agent.
	validAgent := `---
name: valid-agent
description: Valid agent
---
Prompt.
`
	// Invalid agent (missing description).
	invalidAgent := `---
name: invalid-agent
---
Prompt.
`
	// Invalid YAML.
	brokenAgent := `---
name: [broken
---
Prompt.
`

	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "valid.md"), []byte(validAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "invalid.md"), []byte(invalidAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "broken.md"), []byte(brokenAgent), 0o644))

	agents := Discover([]string{agentsDir})

	// Only valid agent should be discovered.
	require.Len(t, agents, 1)
	require.Equal(t, "valid-agent", agents[0].Name)
}

func TestDiscoverFromDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agent := `---
name: test-agent
description: Test agent
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte(agent), 0o644))

	agents := DiscoverFromDir(agentsDir)
	require.Len(t, agents, 1)
	require.Equal(t, "test-agent", agents[0].Name)
}

func TestFindAgentFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	// Create agent files.
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte("test"), 0o644))

	// Create nested agent.
	nestedDir := filepath.Join(agentsDir, "nested-agent")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "nested-agent.md"), []byte("nested"), 0o644))

	paths := []string{agentsDir}

	// Find direct agent.
	path, ok := FindAgentFile("test-agent", paths)
	require.True(t, ok)
	require.Equal(t, filepath.Join(agentsDir, "test-agent.md"), path)

	// Find nested agent.
	path, ok = FindAgentFile("nested-agent", paths)
	require.True(t, ok)
	require.Equal(t, filepath.Join(nestedDir, "nested-agent.md"), path)

	// Not found.
	_, ok = FindAgentFile("nonexistent", paths)
	require.False(t, ok)
}

func TestDiscoverPriority(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	highPriorityDir := filepath.Join(tmpDir, "high")
	lowPriorityDir := filepath.Join(tmpDir, "low")

	require.NoError(t, os.MkdirAll(highPriorityDir, 0o755))
	require.NoError(t, os.MkdirAll(lowPriorityDir, 0o755))

	highAgent := `---
name: high-agent
description: High priority
---
High.
`
	lowAgent := `---
name: low-agent
description: Low priority
---
Low.
`
	require.NoError(t, os.WriteFile(filepath.Join(highPriorityDir, "high.md"), []byte(highAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(lowPriorityDir, "low.md"), []byte(lowAgent), 0o644))

	// High priority path first.
	agents := Discover([]string{highPriorityDir, lowPriorityDir})

	require.Len(t, agents, 2)

	// Verify high priority agent has higher priority value.
	var high, low *Subagent
	for _, a := range agents {
		switch a.Name {
		case "high-agent":
			high = a
		case "low-agent":
			low = a
		}
	}

	require.NotNil(t, high)
	require.NotNil(t, low)
	require.Greater(t, high.Priority, low.Priority)
}

func TestDiscoverIgnoresNonMdFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	// Create various files.
	validAgent := `---
name: valid
description: Valid
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "valid.md"), []byte(validAgent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "readme.txt"), []byte("readme"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "config.yaml"), []byte("config: true"), 0o644))

	agents := Discover([]string{agentsDir})

	require.Len(t, agents, 1)
	require.Equal(t, "valid", agents[0].Name)
}
