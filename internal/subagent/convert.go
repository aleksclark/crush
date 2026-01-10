package subagent

import (
	"github.com/charmbracelet/crush/internal/config"
)

// ToAgent converts a Subagent definition to a config.Agent.
func (s *Subagent) ToAgent() config.Agent {
	agent := config.Agent{
		ID:              s.Name,
		Name:            s.Name,
		Description:     s.Description,
		AllowedTools:    s.Tools,
		DisallowedTools: s.DisallowedTools,
		PermissionMode:  string(s.PermissionMode),
		Skills:          s.Skills,
		SystemPrompt:    s.SystemPrompt,
		SourcePath:      s.FilePath,
		IsSubagent:      true,
	}

	// Map model type.
	switch s.Model {
	case ModelSonnet, ModelOpus, "":
		agent.Model = config.SelectedModelTypeLarge
	case ModelHaiku:
		agent.Model = config.SelectedModelTypeSmall
	case ModelInherit:
		// Use large as default for inherit, coordinator will handle actual inheritance.
		agent.Model = config.SelectedModelTypeLarge
	default:
		agent.Model = config.SelectedModelTypeLarge
	}

	return agent
}

// FromRegistry converts all subagents in a registry to a map of config.Agent.
func FromRegistry(registry *Registry) map[string]config.Agent {
	subagents := registry.List()
	agents := make(map[string]config.Agent, len(subagents))

	for _, s := range subagents {
		agents[s.Name] = s.ToAgent()
	}

	return agents
}
