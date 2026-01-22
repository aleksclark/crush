// Package tracing provides OpenTelemetry tracing functionality for Crush.
//
// It exports spans for agent sessions, tool calls, MCP invocations, and skill
// usage. The trace ID is derived from the session ID for consistent correlation
// across distributed analysis.
package tracing

import (
	"context"
	"encoding/hex"
	"log/slog"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the name of the tracer used by Crush.
	TracerName = "crush"
)

var (
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
	initOnce       sync.Once
	enabled        bool
	mu             sync.RWMutex
)

// Config holds the configuration for tracing.
type Config struct {
	// Endpoint is the OTLP endpoint (e.g., "localhost:4317").
	Endpoint string
	// ServiceName is the name of the service reported to the collector.
	ServiceName string
	// ServiceVersion is the version of the service.
	ServiceVersion string
	// Insecure disables TLS for the connection.
	Insecure bool
}

// Init initializes the OpenTelemetry tracer with the given configuration.
// If the endpoint is empty or initialization fails, tracing is disabled.
func Init(cfg Config) error {
	var initErr error
	initOnce.Do(func() {
		if cfg.Endpoint == "" {
			slog.Debug("Tracing disabled: no endpoint configured")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create OTLP exporter options.
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		// Create the exporter.
		exporter, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			slog.Warn("Failed to create OTLP exporter", "error", err)
			initErr = err
			return
		}

		// Create resource with service information.
		serviceName := cfg.ServiceName
		if serviceName == "" {
			serviceName = "crush"
		}
		serviceVersion := cfg.ServiceVersion
		if serviceVersion == "" {
			serviceVersion = "unknown"
		}

		res := resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			attribute.String("host.name", hostname()),
		)

		// Create the tracer provider.
		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)

		// Register as global tracer provider.
		otel.SetTracerProvider(tracerProvider)
		tracer = tracerProvider.Tracer(TracerName)

		mu.Lock()
		enabled = true
		mu.Unlock()

		slog.Info("Tracing initialized", "endpoint", cfg.Endpoint)
	})

	return initErr
}

// Shutdown gracefully shuts down the tracer provider.
func Shutdown(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if tracerProvider == nil {
		return nil
	}

	return tracerProvider.Shutdown(ctx)
}

// Enabled returns true if tracing is initialized and enabled.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// SessionSpan represents an active session span.
type SessionSpan struct {
	span    trace.Span
	ctx     context.Context
	traceID trace.TraceID
}

// StartSession starts a new root span for a session.
// The session ID is used to derive a deterministic trace ID for correlation.
func StartSession(ctx context.Context, sessionID, prompt string) *SessionSpan {
	if !Enabled() {
		return &SessionSpan{ctx: ctx}
	}

	// Derive trace ID from session ID for consistent correlation.
	traceID := deriveTraceID(sessionID)
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     trace.SpanID{},
		TraceFlags: trace.FlagsSampled,
	})

	ctx = trace.ContextWithSpanContext(ctx, spanCtx)

	ctx, span := tracer.Start(ctx, "session",
		trace.WithAttributes(
			attribute.String("session.id", sessionID),
			attribute.String("session.prompt", truncate(prompt, 500)),
		),
		trace.WithSpanKind(trace.SpanKindServer),
	)

	return &SessionSpan{
		span:    span,
		ctx:     ctx,
		traceID: traceID,
	}
}

// End ends the session span.
func (s *SessionSpan) End() {
	if s.span != nil {
		s.span.End()
	}
}

// SetError sets an error on the session span.
func (s *SessionSpan) SetError(err error) {
	if s.span != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// SetAttributes sets additional attributes on the session span.
func (s *SessionSpan) SetAttributes(attrs ...attribute.KeyValue) {
	if s.span != nil {
		s.span.SetAttributes(attrs...)
	}
}

// Context returns the context with the session span.
func (s *SessionSpan) Context() context.Context {
	return s.ctx
}

// TraceID returns the trace ID for this session.
func (s *SessionSpan) TraceID() string {
	return s.traceID.String()
}

// ToolSpan represents an active tool call span.
type ToolSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartToolCall starts a child span for a tool call.
func StartToolCall(ctx context.Context, toolName, toolID string, input map[string]any) *ToolSpan {
	if !Enabled() {
		return &ToolSpan{ctx: ctx}
	}

	attrs := []attribute.KeyValue{
		attribute.String("tool.name", toolName),
		attribute.String("tool.id", toolID),
	}

	// Add input parameters as attributes (truncated for large values).
	for k, v := range input {
		attrs = append(attrs, attribute.String("tool.input."+k, truncateAny(v, 200)))
	}

	ctx, span := tracer.Start(ctx, "tool."+toolName,
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return &ToolSpan{
		span: span,
		ctx:  ctx,
	}
}

// End ends the tool span.
func (t *ToolSpan) End() {
	if t.span != nil {
		t.span.End()
	}
}

// SetError sets an error on the tool span.
func (t *ToolSpan) SetError(err error) {
	if t.span != nil {
		t.span.RecordError(err)
		t.span.SetStatus(codes.Error, err.Error())
	}
}

// SetOutput sets the tool output as an attribute.
func (t *ToolSpan) SetOutput(output string) {
	if t.span != nil {
		t.span.SetAttributes(attribute.String("tool.output", truncate(output, 500)))
	}
}

// Context returns the context with the tool span.
func (t *ToolSpan) Context() context.Context {
	return t.ctx
}

// MCPSpan represents an active MCP tool invocation span.
type MCPSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartMCPCall starts a child span for an MCP tool invocation.
func StartMCPCall(ctx context.Context, serverName, toolName, sessionID string) *MCPSpan {
	if !Enabled() {
		return &MCPSpan{ctx: ctx}
	}

	ctx, span := tracer.Start(ctx, "mcp."+serverName+"."+toolName,
		trace.WithAttributes(
			attribute.String("mcp.server", serverName),
			attribute.String("mcp.tool", toolName),
			attribute.String("session.id", sessionID),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)

	return &MCPSpan{
		span: span,
		ctx:  ctx,
	}
}

// End ends the MCP span.
func (m *MCPSpan) End() {
	if m.span != nil {
		m.span.End()
	}
}

// SetError sets an error on the MCP span.
func (m *MCPSpan) SetError(err error) {
	if m.span != nil {
		m.span.RecordError(err)
		m.span.SetStatus(codes.Error, err.Error())
	}
}

// SetResult sets the MCP call result attributes.
func (m *MCPSpan) SetResult(success bool, durationMs int64, contentType string) {
	if m.span != nil {
		m.span.SetAttributes(
			attribute.Bool("mcp.success", success),
			attribute.Int64("mcp.duration_ms", durationMs),
			attribute.String("mcp.content_type", contentType),
		)
	}
}

// SetInput sets the MCP call input as an attribute.
func (m *MCPSpan) SetInput(input string) {
	if m.span != nil {
		m.span.SetAttributes(attribute.String("mcp.input", truncate(input, 500)))
	}
}

// SetOutput sets the MCP call output as an attribute.
func (m *MCPSpan) SetOutput(output string) {
	if m.span != nil {
		m.span.SetAttributes(attribute.String("mcp.output", truncate(output, 500)))
	}
}

// Context returns the context with the MCP span.
func (m *MCPSpan) Context() context.Context {
	return m.ctx
}

// SkillSpan represents an active skill usage span.
type SkillSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartSkillUsage starts a child span for skill usage.
func StartSkillUsage(ctx context.Context, skillName, action, sessionID string) *SkillSpan {
	if !Enabled() {
		return &SkillSpan{ctx: ctx}
	}

	ctx, span := tracer.Start(ctx, "skill."+skillName,
		trace.WithAttributes(
			attribute.String("skill.name", skillName),
			attribute.String("skill.action", action),
			attribute.String("session.id", sessionID),
		),
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return &SkillSpan{
		span: span,
		ctx:  ctx,
	}
}

// End ends the skill span.
func (s *SkillSpan) End() {
	if s.span != nil {
		s.span.End()
	}
}

// SetError sets an error on the skill span.
func (s *SkillSpan) SetError(err error) {
	if s.span != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// SetResult sets the skill usage result attributes.
func (s *SkillSpan) SetResult(success bool, filePath string, durationMs int64) {
	if s.span != nil {
		attrs := []attribute.KeyValue{
			attribute.Bool("skill.success", success),
			attribute.Int64("skill.duration_ms", durationMs),
		}
		if filePath != "" {
			attrs = append(attrs, attribute.String("skill.file_path", filePath))
		}
		s.span.SetAttributes(attrs...)
	}
}

// Context returns the context with the skill span.
func (s *SkillSpan) Context() context.Context {
	return s.ctx
}

// LLMSpan represents an active LLM call span.
type LLMSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartLLMCall starts a child span for an LLM API call.
func StartLLMCall(ctx context.Context, provider, model string, inputTokens int64) *LLMSpan {
	if !Enabled() {
		return &LLMSpan{ctx: ctx}
	}

	ctx, span := tracer.Start(ctx, "llm."+provider,
		trace.WithAttributes(
			attribute.String("llm.provider", provider),
			attribute.String("llm.model", model),
			attribute.Int64("llm.input_tokens", inputTokens),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)

	return &LLMSpan{
		span: span,
		ctx:  ctx,
	}
}

// End ends the LLM span.
func (l *LLMSpan) End() {
	if l.span != nil {
		l.span.End()
	}
}

// SetError sets an error on the LLM span.
func (l *LLMSpan) SetError(err error) {
	if l.span != nil {
		l.span.RecordError(err)
		l.span.SetStatus(codes.Error, err.Error())
	}
}

// SetUsage sets the LLM usage attributes.
func (l *LLMSpan) SetUsage(outputTokens, cacheReadTokens, cacheCreationTokens int64) {
	if l.span != nil {
		l.span.SetAttributes(
			attribute.Int64("llm.output_tokens", outputTokens),
			attribute.Int64("llm.cache_read_tokens", cacheReadTokens),
			attribute.Int64("llm.cache_creation_tokens", cacheCreationTokens),
		)
	}
}

// SetFinishReason sets the finish reason for the LLM call.
func (l *LLMSpan) SetFinishReason(reason string) {
	if l.span != nil {
		l.span.SetAttributes(attribute.String("llm.finish_reason", reason))
	}
}

// SetRequest sets the request messages sent to the LLM.
// Messages are JSON-encoded and truncated to avoid excessive span size.
func (l *LLMSpan) SetRequest(messages string) {
	if l.span != nil {
		l.span.SetAttributes(attribute.String("llm.request", truncate(messages, 10000)))
	}
}

// SetResponse sets the response content from the LLM.
func (l *LLMSpan) SetResponse(response string) {
	if l.span != nil {
		l.span.SetAttributes(attribute.String("llm.response", truncate(response, 10000)))
	}
}

// Context returns the context with the LLM span.
func (l *LLMSpan) Context() context.Context {
	return l.ctx
}

// deriveTraceID creates a deterministic trace ID from a session ID.
// This allows correlating all spans from the same session.
func deriveTraceID(sessionID string) trace.TraceID {
	var traceID trace.TraceID

	// Try to parse UUID directly (remove hyphens).
	cleaned := ""
	for _, c := range sessionID {
		if c != '-' {
			cleaned += string(c)
		}
	}

	// If it's a valid 32-char hex string (UUID without hyphens), use it directly.
	if len(cleaned) == 32 {
		if decoded, err := hex.DecodeString(cleaned); err == nil && len(decoded) == 16 {
			copy(traceID[:], decoded)
			return traceID
		}
	}

	// Otherwise, hash the session ID to create a trace ID.
	// Use a simple hash to maintain determinism.
	hash := make([]byte, 16)
	for i, c := range []byte(sessionID) {
		hash[i%16] ^= c
	}
	copy(traceID[:], hash)

	return traceID
}

// hostname returns the hostname or a default value.
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// truncateAny converts any value to a truncated string.
func truncateAny(v any, maxLen int) string {
	s := ""
	switch val := v.(type) {
	case string:
		s = val
	case []byte:
		s = string(val)
	default:
		s = "<complex>"
	}
	return truncate(s, maxLen)
}
