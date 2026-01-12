package agent

import (
	"errors"
	"net/http"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestIsInputTooLongError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-provider error",
			err:      errors.New("some random error"),
			expected: false,
		},
		{
			name: "provider error with unrelated message",
			err: &fantasy.ProviderError{
				Message: "rate limit exceeded",
			},
			expected: false,
		},
		{
			name: "input is too long (exact match from user's error)",
			err: &fantasy.ProviderError{
				Message: "Input is too long for requested model",
			},
			expected: true,
		},
		{
			name: "input is too long (lowercase)",
			err: &fantasy.ProviderError{
				Message: "input is too long for the model",
			},
			expected: true,
		},
		{
			name: "context_length_exceeded",
			err: &fantasy.ProviderError{
				Message: "This model's maximum context length is 128000 tokens. However, your messages resulted in 150000 tokens. context_length_exceeded",
			},
			expected: true,
		},
		{
			name: "maximum context length",
			err: &fantasy.ProviderError{
				Message: "This request exceeds the maximum context length of 128000 tokens",
			},
			expected: true,
		},
		{
			name: "token limit",
			err: &fantasy.ProviderError{
				Message: "Request exceeds the token limit for this model",
			},
			expected: true,
		},
		{
			name: "exceeds model maximum",
			err: &fantasy.ProviderError{
				Message: "Your input exceeds the model's maximum context window",
			},
			expected: true,
		},
		{
			name: "prompt is too long",
			err: &fantasy.ProviderError{
				Message: "The prompt is too long. Please reduce the number of messages.",
			},
			expected: true,
		},
		{
			name: "request too large",
			err: &fantasy.ProviderError{
				Message: "request too large for model context",
			},
			expected: true,
		},
		{
			name: "wrapped provider error",
			err: errors.Join(
				errors.New("wrapper error"),
				&fantasy.ProviderError{
					Message: "Input is too long for requested model",
				},
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isInputTooLongError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-provider error",
			err:      errors.New("some random error"),
			expected: false,
		},
		{
			name: "provider error with unrelated message",
			err: &fantasy.ProviderError{
				Message: "invalid json",
			},
			expected: false,
		},
		{
			name: "429 status code",
			err: &fantasy.ProviderError{
				Message:    "Too Many Requests",
				StatusCode: http.StatusTooManyRequests,
			},
			expected: true,
		},
		{
			name: "rate limit message",
			err: &fantasy.ProviderError{
				Message: "Rate limit exceeded. Please try again later.",
			},
			expected: true,
		},
		{
			name: "too many requests message",
			err: &fantasy.ProviderError{
				Message: "Too many requests in the last minute",
			},
			expected: true,
		},
		{
			name: "too many tokens",
			err: &fantasy.ProviderError{
				Message: "too many tokens: reduce your prompt",
			},
			expected: true,
		},
		{
			name: "quota exceeded",
			err: &fantasy.ProviderError{
				Message: "You have exceeded your quota for this model",
			},
			expected: true,
		},
		{
			name: "throttled",
			err: &fantasy.ProviderError{
				Message: "Request throttled due to high load",
			},
			expected: true,
		},
		{
			name: "overloaded",
			err: &fantasy.ProviderError{
				Message: "The server is currently overloaded",
			},
			expected: true,
		},
		{
			name: "capacity",
			err: &fantasy.ProviderError{
				Message: "No capacity available for this model",
			},
			expected: true,
		},
		{
			name: "requests per minute",
			err: &fantasy.ProviderError{
				Message: "You have exceeded 60 requests per minute",
			},
			expected: true,
		},
		{
			name: "tokens per minute",
			err: &fantasy.ProviderError{
				Message: "Exceeded tokens per minute limit",
			},
			expected: true,
		},
		{
			name: "wrapped rate limit error",
			err: errors.Join(
				errors.New("wrapper error"),
				&fantasy.ProviderError{
					Message:    "Rate limited",
					StatusCode: http.StatusTooManyRequests,
				},
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isRateLimitError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-provider error",
			err:      errors.New("some random error"),
			expected: false,
		},
		{
			name: "408 request timeout",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusRequestTimeout,
			},
			expected: true,
		},
		{
			name: "409 conflict",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusConflict,
			},
			expected: true,
		},
		{
			name: "429 too many requests",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusTooManyRequests,
			},
			expected: true,
		},
		{
			name: "rate limit message without 429",
			err: &fantasy.ProviderError{
				Message:    "Rate limit exceeded",
				StatusCode: http.StatusOK,
			},
			expected: true,
		},
		{
			name: "retry error wrapper",
			err: &fantasy.RetryError{
				Errors: []error{
					&fantasy.ProviderError{Message: "first attempt"},
					&fantasy.ProviderError{Message: "second attempt"},
				},
			},
			expected: true,
		},
		{
			name: "500 server error (not retryable by default)",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusInternalServerError,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isRetryableError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}
