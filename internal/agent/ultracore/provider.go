// Package ultracore provides a fantasy.Provider backed by Ultracore Model Gateway.
package ultracore

import (
	"context"
	"fmt"
	"sync"

	"charm.land/fantasy"
	modelv1 "github.com/aleksclark/ultracore/gen/proto/ultracore/model/v1"
	"github.com/aleksclark/ultracore/modelgateway"
	"github.com/aleksclark/ultracore/modelgateway/grpcclient"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// Name is the Crush provider type for Ultracore Model Gateway.
	Name = "ultracore"

	// DefaultAddress is the default local gateway address.
	DefaultAddress = "127.0.0.1:9090"
)

func init() {
	// Ensure W3C/OTEL propagation works even if Crush has not configured OTEL yet.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

type options struct {
	address string
	name    string
	headers map[string]string
	conn    *grpc.ClientConn
}

// Option configures the Ultracore provider.
type Option func(*options)

// WithAddress sets the gateway address (host:port).
func WithAddress(address string) Option {
	return func(o *options) {
		o.address = address
	}
}

// WithName overrides the provider name returned by fantasy.Provider.Name.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithHeaders sets default headers applied to every model call.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		o.headers = headers
	}
}

// WithConn injects an already-open gRPC connection (mainly for tests).
func WithConn(conn *grpc.ClientConn) Option {
	return func(o *options) {
		o.conn = conn
	}
}

type provider struct {
	opts   options
	client modelv1.ModelGatewayClient
	conn   *grpc.ClientConn

	mu     sync.Mutex
	closed bool
}

// New creates a fantasy.Provider that routes LanguageModel calls through an
// Ultracore model gateway.
func New(opts ...Option) (fantasy.Provider, error) {
	o := options{
		address: DefaultAddress,
		name:    Name,
		headers: map[string]string{},
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.address == "" {
		o.address = DefaultAddress
	}

	p := &provider{opts: o}
	if o.conn != nil {
		p.conn = o.conn
		p.client = modelv1.NewModelGatewayClient(o.conn)
		return p, nil
	}

	conn, err := grpc.NewClient(
		o.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial ultracore gateway %s: %w", o.address, err)
	}
	p.conn = conn
	p.client = modelv1.NewModelGatewayClient(conn)
	return p, nil
}

func (p *provider) Name() string {
	return p.opts.name
}

func (p *provider) LanguageModel(_ context.Context, modelID string) (fantasy.LanguageModel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, fmt.Errorf("ultracore provider is closed")
	}

	// modelID is expected as "provider/model" for multi-provider gateways,
	// or just "model" when the gateway provider id is embedded in WithName.
	gatewayProvider, gatewayModel := splitModelID(modelID, p.opts.name)
	var clientOpts []grpcclient.Option
	if len(p.opts.headers) > 0 {
		clientOpts = append(clientOpts, grpcclient.WithDefaultHeaders(modelgateway.Headers(p.opts.headers)))
	}
	return grpcclient.New(p.client, gatewayProvider, gatewayModel, clientOpts...), nil
}

// Close closes the underlying gRPC connection when owned by this provider.
func (p *provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.conn == nil || p.opts.conn != nil {
		return nil
	}
	err := p.conn.Close()
	p.conn = nil
	return err
}

func splitModelID(modelID, defaultProvider string) (provider, model string) {
	// Prefer explicit "provider/model" form.
	for i := 0; i < len(modelID); i++ {
		if modelID[i] == '/' {
			// Keep provider/model split on first slash only if provider looks like a gateway key.
			// Models such as "google-ai-studio/gemini-..." are valid gateway model ids under a
			// single gateway provider, so only split when defaultProvider is the gateway endpoint
			// name and the left side is not itself a path-like model family for that provider.
			left, right := modelID[:i], modelID[i+1:]
			if left != "" && right != "" && defaultProvider == Name {
				return left, right
			}
			break
		}
	}
	if defaultProvider == "" || defaultProvider == Name {
		// Fall back to a conventional gateway provider name.
		return "openai", modelID
	}
	return defaultProvider, modelID
}
