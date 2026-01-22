package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/tracing"
	"github.com/stretchr/testify/require"
)

func TestIntegrationWithJaeger(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize tracing with Jaeger endpoint.
	err := tracing.Init(tracing.Config{
		Endpoint:       "localhost:4317",
		ServiceName:    "crush-test",
		ServiceVersion: "test",
		Insecure:       true,
	})
	require.NoError(t, err)

	require.True(t, tracing.Enabled())

	// Create a session span.
	ctx := context.Background()
	sessionSpan := tracing.StartSession(ctx, "test-session-123", "test prompt for tracing")
	ctx = sessionSpan.Context()

	// Simulate LLM call (step 1).
	llmSpan := tracing.StartLLMCall(ctx, "anthropic", "claude-sonnet-4-20250514", 1000)
	time.Sleep(10 * time.Millisecond) // Simulate API call.
	llmSpan.SetUsage(500, 800, 0)
	llmSpan.SetFinishReason("tool_calls")
	llmSpan.End()

	// Simulate some tool calls.
	toolSpan := tracing.StartToolCall(ctx, "bash", "tool-1", map[string]any{
		"command": "ls -la",
	})
	time.Sleep(10 * time.Millisecond) // Simulate work.
	toolSpan.SetOutput("file1.txt\nfile2.txt")
	toolSpan.End()

	// Simulate an MCP call.
	mcpSpan := tracing.StartMCPCall(ctx, "filesystem", "read_file", "test-session-123")
	mcpSpan.SetInput(`{"path": "/tmp/test.txt"}`)
	time.Sleep(5 * time.Millisecond) // Simulate work.
	mcpSpan.SetOutput("file contents here...")
	mcpSpan.SetResult(true, 5, "text")
	mcpSpan.End()

	// Simulate LLM call (step 2).
	llmSpan2 := tracing.StartLLMCall(ctx, "anthropic", "claude-sonnet-4-20250514", 2000)
	time.Sleep(5 * time.Millisecond) // Simulate API call.
	llmSpan2.SetUsage(300, 1500, 0)
	llmSpan2.SetFinishReason("stop")
	llmSpan2.End()

	// End session span.
	sessionSpan.End()

	// Shutdown to flush all traces.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = tracing.Shutdown(shutdownCtx)
	require.NoError(t, err)

	t.Log("Traces sent to Jaeger. Check http://localhost:16686 to view them.")
}
