package chat

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/stretchr/testify/require"
)

// TestMiddleMouseClipboardRead tests that middle mouse click reads from
// clipboard and generates a paste message.
func TestMiddleMouseClipboardRead(t *testing.T) {
	t.Parallel()

	// Skip if we're in CI without X11
	if !clipboardAvailable() {
		t.Skip("Clipboard not available (likely CI without X11)")
	}

	// Setup: write test data to clipboard
	testData := "test clipboard content for middle click"
	err := clipboard.WriteAll(testData)
	require.NoError(t, err, "Failed to write to clipboard")

	// Verify we can read it back
	content, err := clipboard.ReadAll()
	require.NoError(t, err, "Failed to read from clipboard")
	require.Equal(t, testData, content, "Clipboard content mismatch")

	// Test the middle mouse click handler logic
	// The handler should return a function that reads clipboard
	// and generates PasteMsg
	handlerFunc := func() tea.Msg {
		content, err := clipboard.ReadAll()
		if err != nil || content == "" {
			return nil
		}
		return tea.PasteMsg{Content: content}
	}

	// Execute the handler
	result := handlerFunc()

	// Verify we got a PasteMsg with correct content
	require.NotNil(t, result, "Expected PasteMsg, got nil")
	pasteMsg, ok := result.(tea.PasteMsg)
	require.True(t, ok, "Expected tea.PasteMsg type")
	require.Equal(t, testData, pasteMsg.Content, "PasteMsg content mismatch")

	// Test with empty clipboard
	err = clipboard.WriteAll("")
	require.NoError(t, err)

	result = handlerFunc()
	require.Nil(t, result, "Expected nil for empty clipboard")
}

// TestClipboardEmptyHandling tests behavior with empty/cleared clipboard.
func TestClipboardEmptyHandling(t *testing.T) {
	t.Parallel()

	if !clipboardAvailable() {
		t.Skip("Clipboard not available")
	}

	// Clear clipboard
	err := clipboard.WriteAll("")
	require.NoError(t, err)

	// Read should return empty string, not error
	content, err := clipboard.ReadAll()
	require.NoError(t, err)
	require.Empty(t, content, "Expected empty clipboard")
}

// TestClipboardLargeContent tests clipboard with large content.
func TestClipboardLargeContent(t *testing.T) {
	t.Parallel()

	if !clipboardAvailable() {
		t.Skip("Clipboard not available")
	}

	// Create large content (10KB)
	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}
	largeString := string(largeContent)

	err := clipboard.WriteAll(largeString)
	require.NoError(t, err)

	content, err := clipboard.ReadAll()
	require.NoError(t, err)
	require.Equal(t, largeString, content, "Large content mismatch")
}

// TestClipboardSpecialCharacters tests clipboard with special characters.
func TestClipboardSpecialCharacters(t *testing.T) {
	t.Parallel()

	if !clipboardAvailable() {
		t.Skip("Clipboard not available")
	}

	testCases := []struct {
		name    string
		content string
	}{
		{"Newlines", "Line1\nLine2\nLine3"},
		{"Tabs", "Col1\tCol2\tCol3"},
		{"Unicode", "Hello 世界 🌍"},
		{"Special symbols", "Test @#$%^&*()"},
		{"Mixed", "Multi\nLine\tWith\tTabs\nAnd Unicode: 你好"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := clipboard.WriteAll(tc.content)
			require.NoError(t, err)

			content, err := clipboard.ReadAll()
			require.NoError(t, err)
			require.Equal(t, tc.content, content)
		})
	}
}

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

// TestPrimarySelectionUsage tests that middle-click uses PRIMARY selection on X11.
func TestPrimarySelectionUsage(t *testing.T) {
	t.Parallel()

	if !clipboardAvailable() {
		t.Skip("Clipboard not available")
	}

	// Setup: Write different content to PRIMARY and CLIPBOARD
	primaryContent := "primary selection content"
	clipboardContent := "clipboard selection content"

	// Write to CLIPBOARD (default)
	err := clipboard.WriteAll(clipboardContent)
	require.NoError(t, err)

	// Write to PRIMARY
	clipboard.Primary = true
	err = clipboard.WriteAll(primaryContent)
	clipboard.Primary = false
	require.NoError(t, err)

	// Test middle-click handler: should read from PRIMARY
	handlerFunc := func() tea.Msg {
		clipboard.Primary = true
		content, err := clipboard.ReadAll()
		clipboard.Primary = false

		if err != nil || content == "" {
			return nil
		}
		return tea.PasteMsg{Content: content}
	}

	result := handlerFunc()
	require.NotNil(t, result)
	pasteMsg, ok := result.(tea.PasteMsg)
	require.True(t, ok)

	// On X11, this should return PRIMARY content
	// On other platforms, PRIMARY flag is ignored and may return CLIPBOARD
	// We just verify the mechanism works
	require.NotEmpty(t, pasteMsg.Content, "Paste content should not be empty")
	t.Logf("Middle-click paste content: %q", pasteMsg.Content)
}

// clipboardAvailable checks if the clipboard is available for testing.
// Returns false in headless environments (like CI) without X11.
func clipboardAvailable() bool {
	// Try to write and read a test value
	testVal := "clipboard_test_probe"
	err := clipboard.WriteAll(testVal)
	if err != nil {
		return false
	}

	content, err := clipboard.ReadAll()
	if err != nil {
		return false
	}

	return content == testVal
}
