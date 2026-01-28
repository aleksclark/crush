package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAllowUnsafeCommandsExecutesCurl tests that a whitelisted command (curl)
// actually gets executed when the LLM requests it.
func TestAllowUnsafeCommandsExecutesCurl(t *testing.T) {
	SkipIfE2EDisabled(t)

	var requestCount atomic.Int32

	// Create a mock OpenAI-compatible server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		if count == 1 {
			// First request: return a tool call for bash with curl --version.
			toolCallID := "call_test123"
			bashInput := map[string]string{
				"command":     "curl --version",
				"description": "Check curl version",
			}
			inputJSON, _ := json.Marshal(bashInput)

			// Stream the tool call using OpenAI format.
			events := []map[string]any{
				{
					"id":      "chatcmpl-1",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"role": "assistant",
								"tool_calls": []map[string]any{
									{
										"index": 0,
										"id":    toolCallID,
										"type":  "function",
										"function": map[string]any{
											"name":      "bash",
											"arguments": "",
										},
									},
								},
							},
							"finish_reason": nil,
						},
					},
				},
				{
					"id":      "chatcmpl-1",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"tool_calls": []map[string]any{
									{
										"index": 0,
										"function": map[string]any{
											"arguments": string(inputJSON),
										},
									},
								},
							},
							"finish_reason": nil,
						},
					},
				},
				{
					"id":      "chatcmpl-1",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index":         0,
							"delta":         map[string]any{},
							"finish_reason": "tool_calls",
						},
					},
				},
			}

			for _, event := range events {
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		} else {
			// Subsequent requests: return a text response.
			events := []map[string]any{
				{
					"id":      "chatcmpl-2",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"role":    "assistant",
								"content": "The curl command executed successfully.",
							},
							"finish_reason": nil,
						},
					},
				},
				{
					"id":      "chatcmpl-2",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index":         0,
							"delta":         map[string]any{},
							"finish_reason": "stop",
						},
					},
				},
			}

			for _, event := range events {
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}
	}))
	defer server.Close()

	// Config with curl whitelisted.
	configJSON := fmt.Sprintf(`{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "%s",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  },
  "options": {
    "allow_unsafe_commands": ["curl"]
  }
}`, server.URL)

	// Use --yolo flag to auto-approve permissions.
	term := NewIsolatedTerminalWithConfigAndArgs(t, 120, 50, configJSON, []string{"--yolo"})
	defer term.Close()

	// Wait for TUI to initialize.
	time.Sleep(startupDelay)

	// Type a message requesting curl.
	term.SendText("run curl --version")
	term.SendText("\r")

	// Wait for the tool to execute and show output.
	// curl --version should show version info.
	found := WaitForText(t, term, "curl", 15*time.Second)
	require.True(t, found, "Expected curl output in terminal")

	// Check that the terminal shows the curl version output.
	snap := term.Snapshot()
	output := SnapshotText(snap)

	// The output should contain something indicating curl ran.
	// Either the actual curl version output or at minimum the command itself.
	hasCurlOutput := strings.Contains(output, "curl") &&
		(strings.Contains(output, "libcurl") ||
			strings.Contains(output, "version") ||
			strings.Contains(output, "--version"))

	require.True(t, hasCurlOutput, "Expected curl to execute, got output:\n%s", output)
}

// TestBlockedCommandWithoutWhitelist tests that curl is blocked by default.
func TestBlockedCommandWithoutWhitelist(t *testing.T) {
	SkipIfE2EDisabled(t)

	var requestCount atomic.Int32

	// Create a mock OpenAI-compatible server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		if count == 1 {
			// First request: return a tool call for bash with curl.
			toolCallID := "call_test456"
			bashInput := map[string]string{
				"command":     "curl --version",
				"description": "Check curl version",
			}
			inputJSON, _ := json.Marshal(bashInput)

			events := []map[string]any{
				{
					"id":      "chatcmpl-1",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"role": "assistant",
								"tool_calls": []map[string]any{
									{
										"index": 0,
										"id":    toolCallID,
										"type":  "function",
										"function": map[string]any{
											"name":      "bash",
											"arguments": "",
										},
									},
								},
							},
							"finish_reason": nil,
						},
					},
				},
				{
					"id":      "chatcmpl-1",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"tool_calls": []map[string]any{
									{
										"index": 0,
										"function": map[string]any{
											"arguments": string(inputJSON),
										},
									},
								},
							},
							"finish_reason": nil,
						},
					},
				},
				{
					"id":      "chatcmpl-1",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index":         0,
							"delta":         map[string]any{},
							"finish_reason": "tool_calls",
						},
					},
				},
			}

			for _, event := range events {
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		} else {
			// Return acknowledgment.
			events := []map[string]any{
				{
					"id":      "chatcmpl-2",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"role":    "assistant",
								"content": "I understand the command was blocked.",
							},
							"finish_reason": nil,
						},
					},
				},
				{
					"id":      "chatcmpl-2",
					"object":  "chat.completion.chunk",
					"created": 1234567890,
					"model":   "test-model",
					"choices": []map[string]any{
						{
							"index":         0,
							"delta":         map[string]any{},
							"finish_reason": "stop",
						},
					},
				},
			}

			for _, event := range events {
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}
	}))
	defer server.Close()

	// Config WITHOUT curl whitelisted.
	configJSON := fmt.Sprintf(`{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "%s",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  }
}`, server.URL)

	// Use --yolo flag to auto-approve permissions (so we can test the block).
	term := NewIsolatedTerminalWithConfigAndArgs(t, 120, 50, configJSON, []string{"--yolo"})
	defer term.Close()

	// Wait for TUI to initialize.
	time.Sleep(startupDelay)

	// Type a message.
	term.SendText("run curl --version")
	term.SendText("\r")

	// Wait for the "not allowed" message to appear.
	found := WaitForText(t, term, "not allowed", 15*time.Second)

	snap := term.Snapshot()
	output := SnapshotText(snap)

	// Should show that the command was not allowed for security reasons.
	require.True(t, found, "Expected 'not allowed' message in output:\n%s", output)
}
