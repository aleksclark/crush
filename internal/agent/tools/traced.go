package tools

import (
	"context"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/tracing"
)

// TracedTool wraps an AgentTool to add tracing spans for each invocation.
type TracedTool struct {
	tool fantasy.AgentTool
}

// WrapWithTracing wraps a tool to add tracing spans.
func WrapWithTracing(tool fantasy.AgentTool) fantasy.AgentTool {
	return &TracedTool{tool: tool}
}

// WrapAllWithTracing wraps all tools in the slice with tracing.
func WrapAllWithTracing(tools []fantasy.AgentTool) []fantasy.AgentTool {
	wrapped := make([]fantasy.AgentTool, len(tools))
	for i, tool := range tools {
		wrapped[i] = WrapWithTracing(tool)
	}
	return wrapped
}

// Info returns the tool info from the wrapped tool.
func (t *TracedTool) Info() fantasy.ToolInfo {
	return t.tool.Info()
}

// Run executes the wrapped tool with tracing.
func (t *TracedTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Extract input parameters for tracing.
	input := make(map[string]any)
	if params.Input != "" {
		input["raw_input"] = params.Input
	}

	// Start tracing span.
	span := tracing.StartToolCall(ctx, params.Name, params.ID, input)
	defer span.End()

	// Execute the tool.
	resp, err := t.tool.Run(span.Context(), params)

	// Record result.
	if err != nil {
		span.SetError(err)
	} else if resp.Content != "" {
		span.SetOutput(resp.Content)
	}

	return resp, err
}

// ProviderOptions returns the provider options from the wrapped tool.
func (t *TracedTool) ProviderOptions() fantasy.ProviderOptions {
	return t.tool.ProviderOptions()
}

// SetProviderOptions sets provider options on the wrapped tool.
func (t *TracedTool) SetProviderOptions(opts fantasy.ProviderOptions) {
	t.tool.SetProviderOptions(opts)
}
