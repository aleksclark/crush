package subagent

import (
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSubagentToAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		subagent Subagent
		check    func(t *testing.T, agent config.Agent)
	}{
		{
			name: "basic subagent",
			subagent: Subagent{
				Name:        "test-agent",
				Description: "Test description",
				FilePath:    "/path/to/agent.md",
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, "test-agent", agent.ID)
				require.Equal(t, "test-agent", agent.Name)
				require.Equal(t, "Test description", agent.Description)
				require.Equal(t, "/path/to/agent.md", agent.SourcePath)
				require.True(t, agent.IsSubagent)
				require.Equal(t, config.SelectedModelTypeLarge, agent.Model)
			},
		},
		{
			name: "subagent with sonnet model",
			subagent: Subagent{
				Name:        "sonnet-agent",
				Description: "Sonnet agent",
				Model:       ModelSonnet,
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, config.SelectedModelTypeLarge, agent.Model)
			},
		},
		{
			name: "subagent with opus model",
			subagent: Subagent{
				Name:        "opus-agent",
				Description: "Opus agent",
				Model:       ModelOpus,
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, config.SelectedModelTypeLarge, agent.Model)
			},
		},
		{
			name: "subagent with haiku model",
			subagent: Subagent{
				Name:        "haiku-agent",
				Description: "Haiku agent",
				Model:       ModelHaiku,
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, config.SelectedModelTypeSmall, agent.Model)
			},
		},
		{
			name: "subagent with inherit model",
			subagent: Subagent{
				Name:        "inherit-agent",
				Description: "Inherit agent",
				Model:       ModelInherit,
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, config.SelectedModelTypeLarge, agent.Model)
			},
		},
		{
			name: "subagent with tools",
			subagent: Subagent{
				Name:            "tools-agent",
				Description:     "Agent with tools",
				Tools:           []string{"Glob", "Grep"},
				DisallowedTools: []string{"Bash"},
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, []string{"Glob", "Grep"}, agent.AllowedTools)
				require.Equal(t, []string{"Bash"}, agent.DisallowedTools)
			},
		},
		{
			name: "subagent with permission mode",
			subagent: Subagent{
				Name:           "permission-agent",
				Description:    "Agent with permissions",
				PermissionMode: PermissionPlan,
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, string(PermissionPlan), agent.PermissionMode)
			},
		},
		{
			name: "subagent with skills",
			subagent: Subagent{
				Name:        "skills-agent",
				Description: "Agent with skills",
				Skills:      []string{"skill1", "skill2"},
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, []string{"skill1", "skill2"}, agent.Skills)
			},
		},
		{
			name: "subagent with system prompt",
			subagent: Subagent{
				Name:         "prompt-agent",
				Description:  "Agent with custom prompt",
				SystemPrompt: "Custom system prompt here.",
			},
			check: func(t *testing.T, agent config.Agent) {
				require.Equal(t, "Custom system prompt here.", agent.SystemPrompt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			agent := tt.subagent.ToAgent()
			tt.check(t, agent)
		})
	}
}

func TestFromRegistry(t *testing.T) {
	t.Parallel()

	// Create a registry with some subagents.
	r := NewRegistry(nil)

	// Manually add subagents to the registry for testing.
	r.mu.Lock()
	r.subagents["agent1"] = &Subagent{
		Name:        "agent1",
		Description: "Agent 1",
		Model:       ModelSonnet,
	}
	r.subagents["agent2"] = &Subagent{
		Name:        "agent2",
		Description: "Agent 2",
		Model:       ModelHaiku,
	}
	r.mu.Unlock()

	agents := FromRegistry(r)

	require.Len(t, agents, 2)
	require.Contains(t, agents, "agent1")
	require.Contains(t, agents, "agent2")

	require.Equal(t, "agent1", agents["agent1"].Name)
	require.Equal(t, config.SelectedModelTypeLarge, agents["agent1"].Model)

	require.Equal(t, "agent2", agents["agent2"].Name)
	require.Equal(t, config.SelectedModelTypeSmall, agents["agent2"].Model)
}
