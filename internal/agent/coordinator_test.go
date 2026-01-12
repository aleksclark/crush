package agent

import (
	"net/http"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestIsContextLengthExceeded(t *testing.T) {
	t.Parallel()

	c := &coordinator{}

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
			err:      &fantasy.Error{Message: "some error"},
			expected: false,
		},
		{
			name: "provider error with wrong status code",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusUnauthorized,
				Message:    "context_length_exceeded",
			},
			expected: false,
		},
		{
			name: "OpenAI context_length_exceeded",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "This model's maximum context length is 128000 tokens. However, you requested 150000 tokens (context_length_exceeded).",
			},
			expected: true,
		},
		{
			name: "Anthropic prompt is too long",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "prompt is too long: 250000 tokens > 200000 maximum",
			},
			expected: true,
		},
		{
			name: "Anthropic max_tokens error",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "max_tokens exceeds model limit",
			},
			expected: true,
		},
		{
			name: "Generic input too long",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "Input is too long for the model",
			},
			expected: true,
		},
		{
			name: "Google exceeds the maximum",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "Request exceeds the maximum allowed input tokens",
			},
			expected: true,
		},
		{
			name: "Token limit exceeded",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "Token limit exceeded for this request",
			},
			expected: true,
		},
		{
			name: "Too many tokens",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "Too many tokens in request",
			},
			expected: true,
		},
		{
			name: "Request too large",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "request too large for model context window",
			},
			expected: true,
		},
		{
			name: "Unrelated 400 error",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid JSON in request body",
			},
			expected: false,
		},
		{
			name: "Maximum context length",
			err: &fantasy.ProviderError{
				StatusCode: http.StatusBadRequest,
				Message:    "maximum context length is 8192",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := c.isContextLengthExceeded(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}
