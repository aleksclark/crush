package permission

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermissionWrapper_ModeBypassPermissions(t *testing.T) {
	t.Parallel()

	underlying := NewPermissionService("/tmp", false, nil)
	wrapped := WrapWithMode(underlying, ModeBypassPermissions)

	// All requests should be auto-approved.
	tests := []string{"bash", "edit", "write", "view", "glob"}
	for _, tool := range tests {
		t.Run(tool, func(t *testing.T) {
			t.Parallel()
			granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
				ToolName: tool,
			})
			require.NoError(t, err)
			require.True(t, granted)
		})
	}

	// SkipRequests should return true.
	require.True(t, wrapped.SkipRequests())
}

func TestPermissionWrapper_ModeYolo(t *testing.T) {
	t.Parallel()

	underlying := NewPermissionService("/tmp", false, nil)
	wrapped := WrapWithMode(underlying, ModeYolo)

	// All requests should be auto-approved (same as bypassPermissions).
	tests := []string{"bash", "edit", "write", "view", "glob"}
	for _, tool := range tests {
		t.Run(tool, func(t *testing.T) {
			t.Parallel()
			granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
				ToolName: tool,
			})
			require.NoError(t, err)
			require.True(t, granted)
		})
	}

	// SkipRequests should return true.
	require.True(t, wrapped.SkipRequests())
}

func TestPermissionWrapper_ModeDontAsk(t *testing.T) {
	t.Parallel()

	underlying := NewPermissionService("/tmp", false, nil)
	wrapped := WrapWithMode(underlying, ModeDontAsk)

	// All requests should be auto-denied.
	tests := []string{"bash", "edit", "write", "view", "glob"}
	for _, tool := range tests {
		t.Run(tool, func(t *testing.T) {
			t.Parallel()
			granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
				ToolName: tool,
			})
			require.ErrorIs(t, err, ErrorPermissionDenied)
			require.False(t, granted)
		})
	}
}

func TestPermissionWrapper_ModePlan(t *testing.T) {
	t.Parallel()

	underlying := NewPermissionService("/tmp", false, nil)
	wrapped := WrapWithMode(underlying, ModePlan)

	// Read-only tools should be auto-approved.
	readOnlyTests := []string{"glob", "grep", "ls", "view", "sourcegraph", "fetch", "lsp_diagnostics", "lsp_references"}
	for _, tool := range readOnlyTests {
		t.Run("allowed_"+tool, func(t *testing.T) {
			t.Parallel()
			granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
				ToolName: tool,
			})
			require.NoError(t, err)
			require.True(t, granted)
		})
	}

	// Write tools should be auto-denied.
	writeTests := []string{"bash", "edit", "write", "multiedit"}
	for _, tool := range writeTests {
		t.Run("denied_"+tool, func(t *testing.T) {
			t.Parallel()
			granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
				ToolName: tool,
			})
			require.ErrorIs(t, err, ErrorPermissionDenied)
			require.False(t, granted)
		})
	}
}

func TestPermissionWrapper_ModeAcceptEdits(t *testing.T) {
	t.Parallel()

	// Use skip mode on underlying so we can test bubbling behavior.
	underlying := NewPermissionService("/tmp", true, nil)
	wrapped := WrapWithMode(underlying, ModeAcceptEdits)

	// Edit tools should be auto-approved.
	editTests := []string{"edit", "multiedit", "write"}
	for _, tool := range editTests {
		t.Run("auto_approved_"+tool, func(t *testing.T) {
			t.Parallel()
			granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
				ToolName: tool,
			})
			require.NoError(t, err)
			require.True(t, granted)
		})
	}

	// Bash should bubble to underlying (which has skip=true, so it approves).
	t.Run("bubbles_bash", func(t *testing.T) {
		t.Parallel()
		granted, err := wrapped.Request(context.Background(), CreatePermissionRequest{
			ToolName: "bash",
		})
		require.NoError(t, err)
		require.True(t, granted, "bash should bubble to underlying service")
	})
}

func TestPermissionWrapper_ModeDefault(t *testing.T) {
	t.Parallel()

	underlying := NewPermissionService("/tmp", true, nil)

	// For default mode, WrapWithMode should return the underlying service.
	wrapped := WrapWithMode(underlying, ModeDefault)
	require.Equal(t, underlying, wrapped, "default mode should return underlying service")

	// Same for empty string.
	wrapped2 := WrapWithMode(underlying, "")
	require.Equal(t, underlying, wrapped2, "empty mode should return underlying service")
}

func TestDescribeEffectivePermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode     PermissionMode
		contains string
	}{
		{ModeBypassPermissions, "yolo"},
		{ModeYolo, "yolo"},
		{ModeDontAsk, "denied"},
		{ModePlan, "Read-only"},
		{ModeAcceptEdits, "edits auto-approved"},
		{ModeDefault, "prompt"},
		{"", "prompt"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			t.Parallel()
			desc := DescribeEffectivePermissions(tt.mode)
			require.Contains(t, desc, tt.contains)
		})
	}
}

func TestPermissionMode_IsValid(t *testing.T) {
	t.Parallel()

	validModes := []PermissionMode{
		"",
		ModeDefault,
		ModeAcceptEdits,
		ModeDontAsk,
		ModeBypassPermissions,
		ModeYolo,
		ModePlan,
	}

	for _, mode := range validModes {
		t.Run("valid_"+string(mode), func(t *testing.T) {
			t.Parallel()
			require.True(t, mode.IsValid())
		})
	}

	invalidModes := []PermissionMode{
		"invalid",
		"YOLO",
		"Default",
	}

	for _, mode := range invalidModes {
		t.Run("invalid_"+string(mode), func(t *testing.T) {
			t.Parallel()
			require.False(t, mode.IsValid())
		})
	}
}

func TestPermissionMode_Normalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    PermissionMode
		expected PermissionMode
	}{
		{ModeYolo, ModeBypassPermissions},
		{ModeBypassPermissions, ModeBypassPermissions},
		{ModeDefault, ModeDefault},
		{ModeDontAsk, ModeDontAsk},
		{ModeAcceptEdits, ModeAcceptEdits},
		{ModePlan, ModePlan},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.input.Normalize())
		})
	}
}
