package agent

import (
	"errors"
	"net/http"
	"strings"

	"charm.land/fantasy"
)

var (
	ErrRequestCancelled = errors.New("request canceled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
	ErrEmptyPrompt      = errors.New("prompt is empty")
	ErrSessionMissing   = errors.New("session id is missing")
)

// inputTooLongPatterns contains known error message patterns that indicate
// the input/context is too long for the model.
var inputTooLongPatterns = []string{
	"input is too long",
	"context_length_exceeded",
	"maximum context length",
	"token limit",
	"exceeds the model's maximum",
	"prompt is too long",
	"request too large",
}

// rateLimitPatterns contains known error message patterns that indicate
// rate limiting (beyond just 429 status code).
var rateLimitPatterns = []string{
	"rate limit",
	"too many requests",
	"too many tokens",
	"quota exceeded",
	"exceeded your quota",
	"exceeded quota",
	"throttled",
	"capacity",
	"overloaded",
	"try again later",
	"request limit",
	"requests per minute",
	"tokens per minute",
}

// isInputTooLongError checks if the error indicates that the input/context
// is too long for the model. This can happen when the conversation history
// exceeds the model's context window.
func isInputTooLongError(err error) bool {
	var providerErr *fantasy.ProviderError
	if !errors.As(err, &providerErr) {
		return false
	}

	msg := strings.ToLower(providerErr.Message)
	for _, pattern := range inputTooLongPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// isRateLimitError checks if the error indicates rate limiting.
// This includes 429 status codes and various rate-limit related messages.
func isRateLimitError(err error) bool {
	var providerErr *fantasy.ProviderError
	if !errors.As(err, &providerErr) {
		return false
	}

	// Check for 429 Too Many Requests status code.
	if providerErr.StatusCode == http.StatusTooManyRequests {
		return true
	}

	// Check for rate-limit related message patterns.
	msg := strings.ToLower(providerErr.Message)
	for _, pattern := range rateLimitPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// isRetryableError checks if the error is retryable (rate limit, timeout, etc.).
// This is used to determine if we should wait and retry after exhausting retries.
func isRetryableError(err error) bool {
	// Check for retry errors first - these indicate the built-in retry
	// mechanism has been exhausted but the error is still retryable.
	var retryErr *fantasy.RetryError
	if errors.As(err, &retryErr) {
		return true
	}

	var providerErr *fantasy.ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.IsRetryable() || isRateLimitError(err)
	}

	return false
}
