// Package openaiextended provides an OpenAI-compatible provider with extended
// cache token support for Bedrock/LiteLLM.
package openaiextended

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	openaisdk "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared"
)

const (
	// Name is the name of the extended OpenAI-compatible provider.
	Name = "openai-extended"
)

// Option defines a function that configures extended provider options.
type Option = func(*options)

type options struct {
	openaiOptions        []openai.Option
	languageModelOptions []openai.LanguageModelOption
	sdkOptions           []option.RequestOption
	objectMode           fantasy.ObjectMode
}

// New creates a new extended OpenAI-compatible provider with custom usage
// functions that support Bedrock/LiteLLM cache tokens.
func New(opts ...Option) (fantasy.Provider, error) {
	providerOptions := options{
		openaiOptions: []openai.Option{
			openai.WithName(Name),
		},
		languageModelOptions: []openai.LanguageModelOption{
			openai.WithLanguageModelPrepareCallFunc(PrepareCallFunc),
			openai.WithLanguageModelStreamExtraFunc(StreamExtraFunc),
			openai.WithLanguageModelExtraContentFunc(ExtraContentFunc),
			openai.WithLanguageModelToPromptFunc(ToPromptFunc),
			// Custom usage functions for extended cache token support.
			openai.WithLanguageModelUsageFunc(UsageFunc),
			openai.WithLanguageModelStreamUsageFunc(StreamUsageFunc),
		},
		objectMode: fantasy.ObjectModeTool, // Default to tool mode.
	}
	for _, o := range opts {
		o(&providerOptions)
	}

	// Handle object mode: convert unsupported modes to tool.
	objectMode := providerOptions.objectMode
	if objectMode == fantasy.ObjectModeAuto || objectMode == fantasy.ObjectModeJSON {
		objectMode = fantasy.ObjectModeTool
	}

	providerOptions.openaiOptions = append(
		providerOptions.openaiOptions,
		openai.WithSDKOptions(providerOptions.sdkOptions...),
		openai.WithLanguageModelOptions(providerOptions.languageModelOptions...),
		openai.WithObjectMode(objectMode),
	)
	return openai.New(providerOptions.openaiOptions...)
}

// WithBaseURL sets the base URL for the provider.
func WithBaseURL(url string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithBaseURL(url))
	}
}

// WithAPIKey sets the API key for the provider.
func WithAPIKey(apiKey string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithAPIKey(apiKey))
	}
}

// WithName sets the name for the provider.
func WithName(name string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithName(name))
	}
}

// WithHeaders sets the headers for the provider.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithHeaders(headers))
	}
}

// WithHTTPClient sets the HTTP client for the provider.
func WithHTTPClient(client option.HTTPClient) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithHTTPClient(client))
	}
}

// WithSDKOptions sets the SDK options for the provider.
func WithSDKOptions(opts ...option.RequestOption) Option {
	return func(o *options) {
		o.sdkOptions = append(o.sdkOptions, opts...)
	}
}

// WithObjectMode sets the object generation mode for the provider.
func WithObjectMode(om fantasy.ObjectMode) Option {
	return func(o *options) {
		o.objectMode = om
	}
}

// WithUseResponsesAPI configures the provider to use the responses API.
func WithUseResponsesAPI() Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithUseResponsesAPI())
	}
}

// Global type identifiers for the extended provider data.
const (
	TypeProviderOptions = Name + ".options"
)

// Register provider-specific types with the global registry.
func init() {
	fantasy.RegisterProviderType(TypeProviderOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
}

// CacheControl represents cache control settings for LiteLLM proxy passthrough.
type CacheControl struct {
	Type string `json:"type"`
}

// ProviderOptions represents additional options for the extended provider.
type ProviderOptions struct {
	User            *string                 `json:"user"`
	ReasoningEffort *openai.ReasoningEffort `json:"reasoning_effort"`
	CacheControl    *CacheControl           `json:"cache_control,omitempty"`
}

// Options implements the ProviderOptions interface.
func (*ProviderOptions) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ProviderOptions.
func (o ProviderOptions) MarshalJSON() ([]byte, error) {
	type plain ProviderOptions
	return fantasy.MarshalProviderType(TypeProviderOptions, plain(o))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ProviderOptions.
func (o *ProviderOptions) UnmarshalJSON(data []byte) error {
	type plain ProviderOptions
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*o = ProviderOptions(p)
	return nil
}

// NewProviderOptions creates new provider options for the extended provider.
func NewProviderOptions(opts *ProviderOptions) fantasy.ProviderOptions {
	return fantasy.ProviderOptions{
		Name: opts,
	}
}

// GetCacheControl extracts cache control settings from provider options.
func GetCacheControl(providerOptions fantasy.ProviderOptions) *CacheControl {
	if providerOptions == nil {
		return nil
	}
	v, ok := providerOptions[Name]
	if !ok {
		return nil
	}
	opts, ok := v.(*ProviderOptions)
	if !ok || opts == nil {
		return nil
	}
	return opts.CacheControl
}

// ParseOptions parses provider options from a map for the extended provider.
func ParseOptions(data map[string]any) (*ProviderOptions, error) {
	var options ProviderOptions
	if err := fantasy.ParseOptions(data, &options); err != nil {
		return nil, err
	}
	return &options, nil
}

// ReasoningData represents reasoning data for OpenAI-compatible provider.
type ReasoningData struct {
	ReasoningContent string `json:"reasoning_content"`
}

// Extended usage types for cache token support.
type extendedPromptTokensDetails struct {
	CachedTokens        int64 `json:"cached_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	AudioTokens         int64 `json:"audio_tokens"`
}

type extendedUsage struct {
	PromptTokensDetails      extendedPromptTokensDetails `json:"prompt_tokens_details"`
	CacheReadInputTokens     int64                       `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64                       `json:"cache_creation_input_tokens"`
}

// UsageFunc extracts extended cache token information from responses.
func UsageFunc(response openaisdk.ChatCompletion) (fantasy.Usage, fantasy.ProviderOptionsData) {
	completionTokenDetails := response.Usage.CompletionTokensDetails
	promptTokenDetails := response.Usage.PromptTokensDetails

	providerMetadata := &openai.ProviderMetadata{}
	if len(response.Choices) > 0 && len(response.Choices[0].Logprobs.Content) > 0 {
		providerMetadata.Logprobs = response.Choices[0].Logprobs.Content
	}
	if completionTokenDetails.AcceptedPredictionTokens > 0 {
		providerMetadata.AcceptedPredictionTokens = completionTokenDetails.AcceptedPredictionTokens
	}
	if completionTokenDetails.RejectedPredictionTokens > 0 {
		providerMetadata.RejectedPredictionTokens = completionTokenDetails.RejectedPredictionTokens
	}

	usage := fantasy.Usage{
		InputTokens:     response.Usage.PromptTokens,
		OutputTokens:    response.Usage.CompletionTokens,
		TotalTokens:     response.Usage.TotalTokens,
		ReasoningTokens: completionTokenDetails.ReasoningTokens,
		CacheReadTokens: promptTokenDetails.CachedTokens,
	}

	// Extract extended cache tokens from raw JSON.
	rawJSON := response.Usage.RawJSON()
	if rawJSON != "" {
		var ext extendedUsage
		if err := json.Unmarshal([]byte(rawJSON), &ext); err == nil {
			if ext.PromptTokensDetails.CacheCreationTokens > 0 {
				usage.CacheCreationTokens = ext.PromptTokensDetails.CacheCreationTokens
			}
			if usage.CacheReadTokens == 0 && ext.CacheReadInputTokens > 0 {
				usage.CacheReadTokens = ext.CacheReadInputTokens
			}
			if usage.CacheCreationTokens == 0 && ext.CacheCreationInputTokens > 0 {
				usage.CacheCreationTokens = ext.CacheCreationInputTokens
			}
		}
	}

	return usage, providerMetadata
}

// StreamUsageFunc extracts extended cache token information from streaming responses.
func StreamUsageFunc(chunk openaisdk.ChatCompletionChunk, _ map[string]any, metadata fantasy.ProviderMetadata) (fantasy.Usage, fantasy.ProviderMetadata) {
	if chunk.Usage.TotalTokens == 0 {
		return fantasy.Usage{}, nil
	}

	streamProviderMetadata := &openai.ProviderMetadata{}
	if metadata != nil {
		if pm, ok := metadata[openai.Name]; ok {
			if converted, ok := pm.(*openai.ProviderMetadata); ok {
				streamProviderMetadata = converted
			}
		}
	}

	completionTokenDetails := chunk.Usage.CompletionTokensDetails
	promptTokenDetails := chunk.Usage.PromptTokensDetails

	usage := fantasy.Usage{
		InputTokens:     chunk.Usage.PromptTokens,
		OutputTokens:    chunk.Usage.CompletionTokens,
		TotalTokens:     chunk.Usage.TotalTokens,
		ReasoningTokens: completionTokenDetails.ReasoningTokens,
		CacheReadTokens: promptTokenDetails.CachedTokens,
	}

	// Extract extended cache tokens from raw JSON.
	rawJSON := chunk.Usage.RawJSON()
	if rawJSON != "" {
		var ext extendedUsage
		if err := json.Unmarshal([]byte(rawJSON), &ext); err == nil {
			if ext.PromptTokensDetails.CacheCreationTokens > 0 {
				usage.CacheCreationTokens = ext.PromptTokensDetails.CacheCreationTokens
			}
			if usage.CacheReadTokens == 0 && ext.CacheReadInputTokens > 0 {
				usage.CacheReadTokens = ext.CacheReadInputTokens
			}
			if usage.CacheCreationTokens == 0 && ext.CacheCreationInputTokens > 0 {
				usage.CacheCreationTokens = ext.CacheCreationInputTokens
			}
		}
	}

	if completionTokenDetails.AcceptedPredictionTokens > 0 {
		streamProviderMetadata.AcceptedPredictionTokens = completionTokenDetails.AcceptedPredictionTokens
	}
	if completionTokenDetails.RejectedPredictionTokens > 0 {
		streamProviderMetadata.RejectedPredictionTokens = completionTokenDetails.RejectedPredictionTokens
	}

	return usage, fantasy.ProviderMetadata{
		openai.Name: streamProviderMetadata,
	}
}

// PrepareCallFunc prepares the call for the language model.
func PrepareCallFunc(_ fantasy.LanguageModel, params *openaisdk.ChatCompletionNewParams, call fantasy.Call) ([]fantasy.CallWarning, error) {
	providerOptions := &ProviderOptions{}
	if v, ok := call.ProviderOptions[Name]; ok {
		providerOptions, ok = v.(*ProviderOptions)
		if !ok {
			return nil, &fantasy.Error{Title: "invalid argument", Message: "provider options should be *openaiextended.ProviderOptions"}
		}
	}

	if providerOptions.ReasoningEffort != nil {
		switch *providerOptions.ReasoningEffort {
		case openai.ReasoningEffortMinimal:
			params.ReasoningEffort = shared.ReasoningEffortMinimal
		case openai.ReasoningEffortLow:
			params.ReasoningEffort = shared.ReasoningEffortLow
		case openai.ReasoningEffortMedium:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case openai.ReasoningEffortHigh:
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			return nil, fmt.Errorf("reasoning model `%s` not supported", *providerOptions.ReasoningEffort)
		}
	}

	if providerOptions.User != nil {
		params.User = param.NewOpt(*providerOptions.User)
	}

	// Add cache_control to tools if present in tool provider options.
	// We iterate through call.Tools to find cache control settings and apply them
	// to the corresponding params.Tools entries.
	for i, tool := range call.Tools {
		if i >= len(params.Tools) {
			break
		}
		funcTool, ok := tool.(fantasy.FunctionTool)
		if !ok {
			continue
		}
		cacheControl := GetCacheControl(funcTool.ProviderOptions)
		if cacheControl == nil {
			continue
		}
		// Apply cache_control to the function definition in params.Tools.
		if params.Tools[i].OfFunction != nil {
			params.Tools[i].OfFunction.Function.SetExtraFields(map[string]any{
				"cache_control": map[string]string{"type": cacheControl.Type},
			})
		}
	}

	return nil, nil
}

// ExtraContentFunc adds extra content to the response.
func ExtraContentFunc(choice openaisdk.ChatCompletionChoice) []fantasy.Content {
	var content []fantasy.Content
	reasoningData := ReasoningData{}
	err := json.Unmarshal([]byte(choice.Message.RawJSON()), &reasoningData)
	if err != nil {
		return content
	}
	if reasoningData.ReasoningContent != "" {
		content = append(content, fantasy.ReasoningContent{
			Text: reasoningData.ReasoningContent,
		})
	}
	return content
}

const reasoningStartedCtx = "reasoning_started"

func extractReasoningContext(ctx map[string]any) bool {
	reasoningStarted, ok := ctx[reasoningStartedCtx]
	if !ok {
		return false
	}
	b, ok := reasoningStarted.(bool)
	if !ok {
		return false
	}
	return b
}

// StreamExtraFunc handles extra functionality for streaming responses.
func StreamExtraFunc(chunk openaisdk.ChatCompletionChunk, yield func(fantasy.StreamPart) bool, ctx map[string]any) (map[string]any, bool) {
	if len(chunk.Choices) == 0 {
		return ctx, true
	}

	reasoningStarted := extractReasoningContext(ctx)

	for inx, choice := range chunk.Choices {
		reasoningData := ReasoningData{}
		err := json.Unmarshal([]byte(choice.Delta.RawJSON()), &reasoningData)
		if err != nil {
			yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeError,
				Error: &fantasy.Error{Title: "stream error", Message: "error unmarshalling delta", Cause: err},
			})
			return ctx, false
		}

		emitEvent := func(reasoningContent string) bool {
			if !reasoningStarted {
				shouldContinue := yield(fantasy.StreamPart{
					Type: fantasy.StreamPartTypeReasoningStart,
					ID:   fmt.Sprintf("%d", inx),
				})
				if !shouldContinue {
					return false
				}
			}

			return yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeReasoningDelta,
				ID:    fmt.Sprintf("%d", inx),
				Delta: reasoningContent,
			})
		}
		if reasoningData.ReasoningContent != "" {
			if !reasoningStarted {
				ctx[reasoningStartedCtx] = true
			}
			return ctx, emitEvent(reasoningData.ReasoningContent)
		}
		if reasoningStarted && (choice.Delta.Content != "" || len(choice.Delta.ToolCalls) > 0) {
			ctx[reasoningStartedCtx] = false
			return ctx, yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeReasoningEnd,
				ID:   fmt.Sprintf("%d", inx),
			})
		}
	}
	return ctx, true
}

// ToPromptFunc converts a fantasy prompt to OpenAI format with reasoning support.
func ToPromptFunc(prompt fantasy.Prompt, _, _ string) ([]openaisdk.ChatCompletionMessageParamUnion, []fantasy.CallWarning) {
	var messages []openaisdk.ChatCompletionMessageParamUnion
	var warnings []fantasy.CallWarning
	for _, msg := range prompt {
		// Check for cache control in openaiextended-specific provider options.
		cacheControl := GetCacheControl(msg.ProviderOptions)

		switch msg.Role {
		case fantasy.MessageRoleSystem:
			// Collect text parts from the system message.
			var textParts []string
			for _, c := range msg.Content {
				if c.GetType() != fantasy.ContentTypeText {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "system prompt can only have text content",
					})
					continue
				}
				textPart, ok := fantasy.AsContentType[fantasy.TextPart](c)
				if !ok {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "system prompt text part does not have the right type",
					})
					continue
				}
				text := textPart.Text
				if strings.TrimSpace(text) != "" {
					textParts = append(textParts, text)
				}
			}
			if len(textParts) == 0 {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "system prompt has no text parts",
				})
				continue
			}
			// Use content parts array only when cache_control is present (for
			// LiteLLM cache passthrough).
			if cacheControl != nil {
				var systemPromptParts []openaisdk.ChatCompletionContentPartTextParam
				for _, text := range textParts {
					systemPromptParts = append(systemPromptParts, openaisdk.ChatCompletionContentPartTextParam{
						Text: text,
					})
				}
				// Add cache_control to the last system content part.
				systemPromptParts[len(systemPromptParts)-1].SetExtraFields(map[string]any{
					"cache_control": map[string]string{"type": cacheControl.Type},
				})
				messages = append(messages, openaisdk.SystemMessage(systemPromptParts))
			} else {
				// Simple string for system message when no cache control.
				messages = append(messages, openaisdk.SystemMessage(strings.Join(textParts, "\n\n")))
			}
		case fantasy.MessageRoleUser:
			// Simple case: single text content without cache control.
			if len(msg.Content) == 1 && msg.Content[0].GetType() == fantasy.ContentTypeText && cacheControl == nil {
				textPart, ok := fantasy.AsContentType[fantasy.TextPart](msg.Content[0])
				if !ok {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "user message text part does not have the right type",
					})
					continue
				}
				messages = append(messages, openaisdk.UserMessage(textPart.Text))
				continue
			}
			var content []openaisdk.ChatCompletionContentPartUnionParam
			for _, c := range msg.Content {
				switch c.GetType() {
				case fantasy.ContentTypeText:
					textPart, ok := fantasy.AsContentType[fantasy.TextPart](c)
					if !ok {
						continue
					}
					textParam := openaisdk.ChatCompletionContentPartTextParam{
						Text: textPart.Text,
					}
					content = append(content, openaisdk.ChatCompletionContentPartUnionParam{
						OfText: &textParam,
					})
				case fantasy.ContentTypeFile:
					filePart, ok := fantasy.AsContentType[fantasy.FilePart](c)
					if !ok {
						continue
					}
					if strings.HasPrefix(filePart.MediaType, "image/") {
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						data := "data:" + filePart.MediaType + ";base64," + base64Encoded
						imageURL := openaisdk.ChatCompletionContentPartImageImageURLParam{URL: data}
						if providerOptions, ok := filePart.ProviderOptions[openai.Name]; ok {
							if detail, ok := providerOptions.(*openai.ProviderFileOptions); ok {
								imageURL.Detail = detail.ImageDetail
							}
						}
						content = append(content, openaisdk.ChatCompletionContentPartUnionParam{
							OfImageURL: &openaisdk.ChatCompletionContentPartImageParam{ImageURL: imageURL},
						})
					}
				}
			}
			if len(content) == 0 {
				continue
			}
			// Add cache_control to the last user content part if present.
			if cacheControl != nil && len(content) > 0 {
				lastPart := &content[len(content)-1]
				if lastPart.OfText != nil {
					lastPart.OfText.SetExtraFields(map[string]any{
						"cache_control": map[string]string{"type": cacheControl.Type},
					})
				}
			}
			messages = append(messages, openaisdk.UserMessage(content))
		case fantasy.MessageRoleAssistant:
			if len(msg.Content) == 1 && msg.Content[0].GetType() == fantasy.ContentTypeText {
				textPart, ok := fantasy.AsContentType[fantasy.TextPart](msg.Content[0])
				if !ok {
					continue
				}
				messages = append(messages, openaisdk.AssistantMessage(textPart.Text))
				continue
			}
			assistantMsg := openaisdk.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}
			var reasoningText string
			for _, c := range msg.Content {
				switch c.GetType() {
				case fantasy.ContentTypeText:
					textPart, ok := fantasy.AsContentType[fantasy.TextPart](c)
					if !ok {
						continue
					}
					assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(textPart.Text),
					}
				case fantasy.ContentTypeReasoning:
					reasoningPart, ok := fantasy.AsContentType[fantasy.ReasoningPart](c)
					if !ok {
						continue
					}
					reasoningText = reasoningPart.Text
				case fantasy.ContentTypeToolCall:
					toolCallPart, ok := fantasy.AsContentType[fantasy.ToolCallPart](c)
					if !ok {
						continue
					}
					assistantMsg.ToolCalls = append(assistantMsg.ToolCalls,
						openaisdk.ChatCompletionMessageToolCallUnionParam{
							OfFunction: &openaisdk.ChatCompletionMessageFunctionToolCallParam{
								ID:   toolCallPart.ToolCallID,
								Type: "function",
								Function: openaisdk.ChatCompletionMessageFunctionToolCallFunctionParam{
									Name:      toolCallPart.ToolName,
									Arguments: toolCallPart.Input,
								},
							},
						})
				}
			}
			if reasoningText != "" {
				assistantMsg.SetExtraFields(map[string]any{
					"reasoning_content": reasoningText,
				})
			}
			messages = append(messages, openaisdk.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})
		case fantasy.MessageRoleTool:
			for _, c := range msg.Content {
				if c.GetType() != fantasy.ContentTypeToolResult {
					continue
				}
				toolResultPart, ok := fantasy.AsContentType[fantasy.ToolResultPart](c)
				if !ok {
					continue
				}
				switch toolResultPart.Output.GetType() {
				case fantasy.ToolResultContentTypeText:
					output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentText](toolResultPart.Output)
					if !ok {
						continue
					}
					messages = append(messages, openaisdk.ToolMessage(output.Text, toolResultPart.ToolCallID))
				case fantasy.ToolResultContentTypeError:
					output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](toolResultPart.Output)
					if !ok {
						continue
					}
					messages = append(messages, openaisdk.ToolMessage(output.Error.Error(), toolResultPart.ToolCallID))
				}
			}
		}
	}
	return messages, warnings
}
