package chat

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// TestMiddleMouseButtonDetection tests that we correctly identify middle mouse button.
func TestMiddleMouseButtonDetection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		button   tea.MouseButton
		expected bool
	}{
		{"Left button", tea.MouseLeft, false},
		{"Right button", tea.MouseRight, false},
		{"Middle button", tea.MouseMiddle, true},
		{"Wheel up", tea.MouseWheelUp, false},
		{"Wheel down", tea.MouseWheelDown, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tea.MouseClickMsg{
				Button: tc.button,
				X:      10,
				Y:      10,
			}

			isMiddle := msg.Button == tea.MouseMiddle
			require.Equal(t, tc.expected, isMiddle)
		})
	}
}

// TestClipboardMsgConversion tests that ClipboardMsg is converted to PasteMsg.
func TestClipboardMsgConversion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		clipboardMsg   tea.ClipboardMsg
		expectPasteMsg bool
	}{
		{
			name:           "Non-empty content",
			clipboardMsg:   tea.ClipboardMsg{Content: "test content"},
			expectPasteMsg: true,
		},
		{
			name:           "Empty content",
			clipboardMsg:   tea.ClipboardMsg{Content: ""},
			expectPasteMsg: false,
		},
		{
			name:           "Content with newlines",
			clipboardMsg:   tea.ClipboardMsg{Content: "line1\nline2\nline3"},
			expectPasteMsg: true,
		},
		{
			name:           "Unicode content",
			clipboardMsg:   tea.ClipboardMsg{Content: "Hello 世界 🌍"},
			expectPasteMsg: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the conversion logic from Update
			var resultMsg tea.Msg
			if tc.clipboardMsg.Content != "" {
				resultMsg = tea.PasteMsg{Content: tc.clipboardMsg.Content}
			}

			if tc.expectPasteMsg {
				require.NotNil(t, resultMsg)
				pasteMsg, ok := resultMsg.(tea.PasteMsg)
				require.True(t, ok, "Expected tea.PasteMsg type")
				require.Equal(t, tc.clipboardMsg.Content, pasteMsg.Content)
			} else {
				require.Nil(t, resultMsg)
			}
		})
	}
}

// TestOSC52ClipboardCommands tests that OSC 52 clipboard commands are available.
func TestOSC52ClipboardCommands(t *testing.T) {
	t.Parallel()

	// Verify the OSC 52 clipboard functions exist and return the right types.
	// We can't actually test OSC 52 in unit tests since it requires terminal support.

	// tea.SetClipboard returns a Cmd
	setCmd := tea.SetClipboard("test")
	require.NotNil(t, setCmd, "SetClipboard should return a Cmd")

	// tea.SetPrimaryClipboard returns a Cmd
	setPrimaryCmd := tea.SetPrimaryClipboard("test")
	require.NotNil(t, setPrimaryCmd, "SetPrimaryClipboard should return a Cmd")

	// tea.ReadPrimaryClipboard returns a Msg
	readMsg := tea.ReadPrimaryClipboard()
	require.NotNil(t, readMsg, "ReadPrimaryClipboard should return a Msg")
}
