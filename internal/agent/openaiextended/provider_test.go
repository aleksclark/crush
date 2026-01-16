package openaiextended

import (
	"encoding/json"
	"testing"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	openaisdk "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared"
	"github.com/stretchr/testify/require"
)

func TestUsageFunc_StandardOpenAI(t *testing.T) {
	t.Parallel()

	// Standard OpenAI response with cached_tokens only.
	rawJSON := `{
		"prompt_tokens": 100,
		"completion_tokens": 50,
		"total_tokens": 150,
		"prompt_tokens_details": {
			"cached_tokens": 80,
			"audio_tokens": 0
		},
		"completion_tokens_details": {
			"reasoning_tokens": 10,
			"accepted_prediction_tokens": 0,
			"rejected_prediction_tokens": 0
		}
	}`

	response := makeChatCompletion(t, rawJSON)
	usage, metadata := UsageFunc(response)

	require.Equal(t, int64(100), usage.InputTokens)
	require.Equal(t, int64(50), usage.OutputTokens)
	require.Equal(t, int64(150), usage.TotalTokens)
	require.Equal(t, int64(80), usage.CacheReadTokens)
	require.Equal(t, int64(10), usage.ReasoningTokens)
	require.Equal(t, int64(0), usage.CacheCreationTokens) // Standard OpenAI doesn't have this.
	require.NotNil(t, metadata)
}

func TestUsageFunc_LiteLLMFormat(t *testing.T) {
	t.Parallel()

	// LiteLLM proxying Bedrock/Anthropic with cache_creation_tokens.
	rawJSON := `{
		"prompt_tokens": 200,
		"completion_tokens": 100,
		"total_tokens": 300,
		"prompt_tokens_details": {
			"cached_tokens": 150,
			"cache_creation_tokens": 40,
			"audio_tokens": 0
		},
		"completion_tokens_details": {
			"reasoning_tokens": 0,
			"accepted_prediction_tokens": 0,
			"rejected_prediction_tokens": 0
		}
	}`

	response := makeChatCompletion(t, rawJSON)
	usage, _ := UsageFunc(response)

	require.Equal(t, int64(200), usage.InputTokens)
	require.Equal(t, int64(100), usage.OutputTokens)
	require.Equal(t, int64(300), usage.TotalTokens)
	require.Equal(t, int64(150), usage.CacheReadTokens)
	require.Equal(t, int64(40), usage.CacheCreationTokens) // LiteLLM extended field.
}

func TestUsageFunc_AnthropicFormatFallback(t *testing.T) {
	t.Parallel()

	// Direct Anthropic-style fields at root level.
	rawJSON := `{
		"prompt_tokens": 300,
		"completion_tokens": 75,
		"total_tokens": 375,
		"cache_read_input_tokens": 200,
		"cache_creation_input_tokens": 60,
		"prompt_tokens_details": {},
		"completion_tokens_details": {}
	}`

	response := makeChatCompletion(t, rawJSON)
	usage, _ := UsageFunc(response)

	require.Equal(t, int64(300), usage.InputTokens)
	require.Equal(t, int64(75), usage.OutputTokens)
	require.Equal(t, int64(375), usage.TotalTokens)
	require.Equal(t, int64(200), usage.CacheReadTokens)    // From cache_read_input_tokens.
	require.Equal(t, int64(60), usage.CacheCreationTokens) // From cache_creation_input_tokens.
}

func TestUsageFunc_MixedFormat(t *testing.T) {
	t.Parallel()

	// SDK provides cached_tokens, but we need cache_creation from raw JSON.
	rawJSON := `{
		"prompt_tokens": 500,
		"completion_tokens": 200,
		"total_tokens": 700,
		"prompt_tokens_details": {
			"cached_tokens": 350,
			"cache_creation_tokens": 100
		},
		"completion_tokens_details": {
			"reasoning_tokens": 25,
			"accepted_prediction_tokens": 5,
			"rejected_prediction_tokens": 2
		}
	}`

	response := makeChatCompletion(t, rawJSON)
	usage, metadata := UsageFunc(response)

	require.Equal(t, int64(500), usage.InputTokens)
	require.Equal(t, int64(200), usage.OutputTokens)
	require.Equal(t, int64(700), usage.TotalTokens)
	require.Equal(t, int64(350), usage.CacheReadTokens)
	require.Equal(t, int64(100), usage.CacheCreationTokens)
	require.Equal(t, int64(25), usage.ReasoningTokens)

	// Check provider metadata for prediction tokens.
	pm, ok := metadata.(*openai.ProviderMetadata)
	require.True(t, ok)
	require.Equal(t, int64(5), pm.AcceptedPredictionTokens)
	require.Equal(t, int64(2), pm.RejectedPredictionTokens)
}

func TestUsageFunc_EmptyRawJSON(t *testing.T) {
	t.Parallel()

	// Minimal response with no raw JSON.
	response := openaisdk.ChatCompletion{
		Usage: openaisdk.CompletionUsage{
			PromptTokens:     50,
			CompletionTokens: 25,
			TotalTokens:      75,
		},
	}

	usage, _ := UsageFunc(response)

	require.Equal(t, int64(50), usage.InputTokens)
	require.Equal(t, int64(25), usage.OutputTokens)
	require.Equal(t, int64(75), usage.TotalTokens)
	require.Equal(t, int64(0), usage.CacheReadTokens)
	require.Equal(t, int64(0), usage.CacheCreationTokens)
}

func TestStreamUsageFunc_LiteLLMFormat(t *testing.T) {
	t.Parallel()

	rawJSON := `{
		"prompt_tokens": 400,
		"completion_tokens": 150,
		"total_tokens": 550,
		"prompt_tokens_details": {
			"cached_tokens": 300,
			"cache_creation_tokens": 80
		},
		"completion_tokens_details": {
			"reasoning_tokens": 15
		}
	}`

	chunk := makeChatCompletionChunk(t, rawJSON)
	usage, metadata := StreamUsageFunc(chunk, nil, nil)

	require.Equal(t, int64(400), usage.InputTokens)
	require.Equal(t, int64(150), usage.OutputTokens)
	require.Equal(t, int64(550), usage.TotalTokens)
	require.Equal(t, int64(300), usage.CacheReadTokens)
	require.Equal(t, int64(80), usage.CacheCreationTokens)
	require.Equal(t, int64(15), usage.ReasoningTokens)
	require.NotNil(t, metadata)
}

func TestStreamUsageFunc_ZeroTotalTokens(t *testing.T) {
	t.Parallel()

	// When total_tokens is 0, return empty usage.
	chunk := openaisdk.ChatCompletionChunk{
		Usage: openaisdk.CompletionUsage{
			TotalTokens: 0,
		},
	}

	usage, metadata := StreamUsageFunc(chunk, nil, nil)

	require.Equal(t, int64(0), usage.InputTokens)
	require.Equal(t, int64(0), usage.OutputTokens)
	require.Nil(t, metadata)
}

func TestStreamUsageFunc_PreservesMetadata(t *testing.T) {
	t.Parallel()

	rawJSON := `{
		"prompt_tokens": 100,
		"completion_tokens": 50,
		"total_tokens": 150,
		"prompt_tokens_details": {},
		"completion_tokens_details": {
			"accepted_prediction_tokens": 10,
			"rejected_prediction_tokens": 5
		}
	}`

	chunk := makeChatCompletionChunk(t, rawJSON)

	// Pass existing metadata.
	existingMetadata := fantasy.ProviderMetadata{
		openai.Name: &openai.ProviderMetadata{
			AcceptedPredictionTokens: 3,
		},
	}

	usage, metadata := StreamUsageFunc(chunk, nil, existingMetadata)

	require.Equal(t, int64(100), usage.InputTokens)
	require.NotNil(t, metadata)
	pm, ok := metadata[openai.Name].(*openai.ProviderMetadata)
	require.True(t, ok)
	require.Equal(t, int64(10), pm.AcceptedPredictionTokens)
	require.Equal(t, int64(5), pm.RejectedPredictionTokens)
}

func TestParseOptions(t *testing.T) {
	t.Parallel()

	t.Run("empty options", func(t *testing.T) {
		t.Parallel()
		opts, err := ParseOptions(map[string]any{})
		require.NoError(t, err)
		require.NotNil(t, opts)
		require.Nil(t, opts.User)
		require.Nil(t, opts.ReasoningEffort)
	})

	t.Run("with user", func(t *testing.T) {
		t.Parallel()
		opts, err := ParseOptions(map[string]any{
			"user": "test-user",
		})
		require.NoError(t, err)
		require.NotNil(t, opts.User)
		require.Equal(t, "test-user", *opts.User)
	})

	t.Run("with reasoning effort", func(t *testing.T) {
		t.Parallel()
		opts, err := ParseOptions(map[string]any{
			"reasoning_effort": "high",
		})
		require.NoError(t, err)
		require.NotNil(t, opts.ReasoningEffort)
		require.Equal(t, openai.ReasoningEffortHigh, *opts.ReasoningEffort)
	})

	t.Run("with cache control", func(t *testing.T) {
		t.Parallel()
		opts, err := ParseOptions(map[string]any{
			"cache_control": map[string]any{
				"type": "ephemeral",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, opts.CacheControl)
		require.Equal(t, "ephemeral", opts.CacheControl.Type)
	})
}

func TestProviderOptionsJSON(t *testing.T) {
	t.Parallel()

	user := "test-user"
	effort := openai.ReasoningEffortMedium
	opts := ProviderOptions{
		User:            &user,
		ReasoningEffort: &effort,
	}

	// Marshal produces type-wrapped JSON.
	data, err := json.Marshal(opts)
	require.NoError(t, err)

	// Should include type info wrapper.
	require.Contains(t, string(data), TypeProviderOptions)
	require.Contains(t, string(data), `"type":"openai-extended.options"`)
	require.Contains(t, string(data), `"user":"test-user"`)
	require.Contains(t, string(data), `"reasoning_effort":"medium"`)

	// NOTE: Direct json.Unmarshal doesn't restore fields because UnmarshalJSON
	// expects pre-unwrapped data (from registry). This is by design for
	// fantasy's polymorphic type system. The registry handles full round-trips.
}

func TestNew(t *testing.T) {
	t.Parallel()

	provider, err := New(
		WithBaseURL("https://api.example.com"),
		WithAPIKey("test-key"),
		WithName("custom-name"),
	)
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestNewWithObjectMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode fantasy.ObjectMode
	}{
		{"tool mode", fantasy.ObjectModeTool},
		{"auto mode converts to tool", fantasy.ObjectModeAuto},
		{"json mode converts to tool", fantasy.ObjectModeJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, err := New(
				WithBaseURL("https://api.example.com"),
				WithAPIKey("test-key"),
				WithObjectMode(tt.mode),
			)
			require.NoError(t, err)
			require.NotNil(t, provider)
		})
	}
}

// makeChatCompletion creates a ChatCompletion with raw JSON for usage.
func makeChatCompletion(t *testing.T, usageJSON string) openaisdk.ChatCompletion {
	t.Helper()

	// Build full response JSON.
	fullJSON := `{
		"id": "test-id",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "test-model",
		"choices": [],
		"usage": ` + usageJSON + `
	}`

	var response openaisdk.ChatCompletion
	err := json.Unmarshal([]byte(fullJSON), &response)
	require.NoError(t, err)

	return response
}

// makeChatCompletionChunk creates a ChatCompletionChunk with raw JSON for usage.
func makeChatCompletionChunk(t *testing.T, usageJSON string) openaisdk.ChatCompletionChunk {
	t.Helper()

	// Build full chunk JSON.
	fullJSON := `{
		"id": "test-id",
		"object": "chat.completion.chunk",
		"created": 1234567890,
		"model": "test-model",
		"choices": [],
		"usage": ` + usageJSON + `
	}`

	var chunk openaisdk.ChatCompletionChunk
	err := json.Unmarshal([]byte(fullJSON), &chunk)
	require.NoError(t, err)

	return chunk
}

func TestToPromptFunc_SystemMessageCacheControl(t *testing.T) {
	t.Parallel()

	prompt := fantasy.Prompt{
		fantasy.Message{
			Role: fantasy.MessageRoleSystem,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: "You are a helpful assistant."},
				fantasy.TextPart{Text: "Here is some context."},
			},
			ProviderOptions: fantasy.ProviderOptions{
				Name: &ProviderOptions{
					CacheControl: &CacheControl{Type: "ephemeral"},
				},
			},
		},
	}

	messages, warnings := ToPromptFunc(prompt, "", "")
	require.Empty(t, warnings)
	require.Len(t, messages, 1)

	// Marshal to JSON to verify cache_control is present.
	data, err := json.Marshal(messages[0])
	require.NoError(t, err)

	// Should contain cache_control on the last content part.
	require.Contains(t, string(data), `"cache_control"`)
	require.Contains(t, string(data), `"type":"ephemeral"`)
}

func TestToPromptFunc_SystemMessageNoCacheControl(t *testing.T) {
	t.Parallel()

	prompt := fantasy.Prompt{
		fantasy.Message{
			Role: fantasy.MessageRoleSystem,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: "You are a helpful assistant."},
			},
		},
	}

	messages, warnings := ToPromptFunc(prompt, "", "")
	require.Empty(t, warnings)
	require.Len(t, messages, 1)

	// Marshal to JSON to verify no cache_control.
	data, err := json.Marshal(messages[0])
	require.NoError(t, err)

	// Should NOT contain cache_control.
	require.NotContains(t, string(data), `"cache_control"`)
}

func TestToPromptFunc_UserMessageCacheControl(t *testing.T) {
	t.Parallel()

	prompt := fantasy.Prompt{
		fantasy.Message{
			Role: fantasy.MessageRoleUser,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: "What is the meaning of life?"},
			},
			ProviderOptions: fantasy.ProviderOptions{
				Name: &ProviderOptions{
					CacheControl: &CacheControl{Type: "ephemeral"},
				},
			},
		},
	}

	messages, warnings := ToPromptFunc(prompt, "", "")
	require.Empty(t, warnings)
	require.Len(t, messages, 1)

	// Marshal to JSON to verify cache_control is present.
	data, err := json.Marshal(messages[0])
	require.NoError(t, err)

	// Should contain cache_control.
	require.Contains(t, string(data), `"cache_control"`)
	require.Contains(t, string(data), `"type":"ephemeral"`)
}

func TestToPromptFunc_UserMessageMultiContentCacheControl(t *testing.T) {
	t.Parallel()

	prompt := fantasy.Prompt{
		fantasy.Message{
			Role: fantasy.MessageRoleUser,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: "First part."},
				fantasy.TextPart{Text: "Second part."},
			},
			ProviderOptions: fantasy.ProviderOptions{
				Name: &ProviderOptions{
					CacheControl: &CacheControl{Type: "ephemeral"},
				},
			},
		},
	}

	messages, warnings := ToPromptFunc(prompt, "", "")
	require.Empty(t, warnings)
	require.Len(t, messages, 1)

	// Marshal to JSON.
	data, err := json.Marshal(messages[0])
	require.NoError(t, err)

	// Should contain cache_control (on the last text part).
	require.Contains(t, string(data), `"cache_control"`)
	require.Contains(t, string(data), `"type":"ephemeral"`)
}

func TestPrepareCallFunc_ToolCacheControl(t *testing.T) {
	t.Parallel()

	params := &openaisdk.ChatCompletionNewParams{
		Tools: []openaisdk.ChatCompletionToolUnionParam{
			{
				OfFunction: &openaisdk.ChatCompletionFunctionToolParam{
					Function: shared.FunctionDefinitionParam{
						Name:        "get_weather",
						Description: param.NewOpt("Get the weather"),
					},
				},
			},
			{
				OfFunction: &openaisdk.ChatCompletionFunctionToolParam{
					Function: shared.FunctionDefinitionParam{
						Name:        "get_time",
						Description: param.NewOpt("Get the time"),
					},
				},
			},
		},
	}

	call := fantasy.Call{
		Tools: []fantasy.Tool{
			fantasy.FunctionTool{
				Name:        "get_weather",
				Description: "Get the weather",
			},
			fantasy.FunctionTool{
				Name:        "get_time",
				Description: "Get the time",
				ProviderOptions: fantasy.ProviderOptions{
					Name: &ProviderOptions{
						CacheControl: &CacheControl{Type: "ephemeral"},
					},
				},
			},
		},
	}

	warnings, err := PrepareCallFunc(nil, params, call)
	require.NoError(t, err)
	require.Empty(t, warnings)

	// First tool should NOT have cache_control.
	data1, err := json.Marshal(params.Tools[0])
	require.NoError(t, err)
	require.NotContains(t, string(data1), `"cache_control"`)

	// Second tool should have cache_control.
	data2, err := json.Marshal(params.Tools[1])
	require.NoError(t, err)
	require.Contains(t, string(data2), `"cache_control"`)
	require.Contains(t, string(data2), `"type":"ephemeral"`)
}
