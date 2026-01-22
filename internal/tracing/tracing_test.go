package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveTraceID(t *testing.T) {
	t.Parallel()

	// Test with a valid UUID.
	traceID1 := deriveTraceID("550e8400-e29b-41d4-a716-446655440000")
	require.NotEmpty(t, traceID1.String())
	require.NotEqual(t, "00000000000000000000000000000000", traceID1.String())

	// Test that same input produces same output (deterministic).
	traceID2 := deriveTraceID("550e8400-e29b-41d4-a716-446655440000")
	require.Equal(t, traceID1, traceID2)

	// Test with a different UUID.
	traceID3 := deriveTraceID("123e4567-e89b-12d3-a456-426614174000")
	require.NotEqual(t, traceID1, traceID3)

	// Test with a non-UUID string.
	traceID4 := deriveTraceID("some-random-session-id")
	require.NotEmpty(t, traceID4.String())
}

func TestSessionSpanWithoutInit(t *testing.T) {
	t.Parallel()

	// Without initialization, spans should be no-ops but not panic.
	span := StartSession(context.Background(), "test-session", "test prompt")
	require.NotNil(t, span)
	require.NotNil(t, span.Context())

	span.SetAttributes()
	span.End()
}

func TestToolSpanWithoutInit(t *testing.T) {
	t.Parallel()

	span := StartToolCall(context.Background(), "test-tool", "tool-123", map[string]any{"key": "value"})
	require.NotNil(t, span)
	require.NotNil(t, span.Context())

	span.SetOutput("test output")
	span.End()
}

func TestMCPSpanWithoutInit(t *testing.T) {
	t.Parallel()

	span := StartMCPCall(context.Background(), "test-server", "test-tool", "session-123")
	require.NotNil(t, span)
	require.NotNil(t, span.Context())

	span.SetResult(true, 100, "text")
	span.End()
}

func TestSkillSpanWithoutInit(t *testing.T) {
	t.Parallel()

	span := StartSkillUsage(context.Background(), "test-skill", "read", "session-123")
	require.NotNil(t, span)
	require.NotNil(t, span.Context())

	span.SetResult(true, "/path/to/file", 50)
	span.End()
}

func TestEnabledReturnsFalseWithoutInit(t *testing.T) {
	// Note: This test may pass or fail depending on whether other tests
	// have initialized tracing. The init is a sync.Once so we can only
	// test the initial state if no other test has run Init().
	// We skip if tracing is already enabled.
	if Enabled() {
		t.Skip("tracing already initialized by another test")
	}
	require.False(t, Enabled())
}
