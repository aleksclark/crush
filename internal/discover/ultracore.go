package discover

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	modelv1 "github.com/aleksclark/ultracore/gen/proto/ultracore/model/v1"
	"github.com/charmbracelet/crush/internal/agent/ultracore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	// Register so type "ultracore" is accepted as a known custom provider
	// (schema enum + load.go validation). Discovery itself uses gRPC
	// ListModels rather than HTTP /models.
	RegisterEnricher(ultracore.Name, &ultracoreEnricher{})
}

// ultracoreEnricher is a no-op. Gateway ListModels already returns the
// metadata Crush needs; registering it only marks the type as known.
type ultracoreEnricher struct{}

func (e *ultracoreEnricher) EnrichModels(_ context.Context, _ Config, _ Resolver, models []catwalk.Model) ([]catwalk.Model, error) {
	return models, nil
}

// DiscoverUltracoreModels lists models from an Ultracore model gateway via
// gRPC ListModels. BaseURL is treated as host:port (optional http(s):// and
// trailing path are stripped). Existing model IDs in cfg are preserved and
// win over gateway entries with the same ID.
func DiscoverUltracoreModels(ctx context.Context, cfg Config, resolver Resolver) ([]catwalk.Model, error) {
	address, err := resolveGatewayAddress(cfg.BaseURL, resolver)
	if err != nil {
		return nil, fmt.Errorf("discover models for provider %s: %w", cfg.ID, err)
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("discover models for provider %s: dial %s: %w", cfg.ID, address, err)
	}
	defer conn.Close()

	client := modelv1.NewModelGatewayClient(conn)
	// Filter by gateway provider when the Crush provider id is a concrete
	// upstream name (anthropic, openai, ...). A top-level "ultracore"
	// provider lists everything.
	filter := ""
	if cfg.ID != "" && cfg.ID != ultracore.Name {
		filter = cfg.ID
	}

	resp, err := client.ListModels(ctx, &modelv1.ListModelsRequest{Provider: filter})
	if err != nil {
		return nil, fmt.Errorf("discover models for provider %s: %w", cfg.ID, err)
	}

	existing := make(map[string]struct{}, len(cfg.ExistingModels))
	for _, m := range cfg.ExistingModels {
		existing[m.ID] = struct{}{}
	}

	result := make([]catwalk.Model, len(cfg.ExistingModels))
	copy(result, cfg.ExistingModels)

	for _, gm := range resp.GetModels() {
		if gm == nil || gm.GetRef() == nil {
			continue
		}
		id := gm.GetRef().GetModel()
		if id == "" {
			continue
		}
		if _, ok := existing[id]; ok {
			continue
		}
		result = append(result, gatewayModelToCatwalk(gm))
	}
	return result, nil
}

// ListUltracoreGatewayModels returns every model the gateway exposes,
// grouped by upstream provider id. Used to expand a single ultracore
// endpoint into per-provider Crush configs so each shows as configured.
func ListUltracoreGatewayModels(ctx context.Context, address string) (map[string][]catwalk.Model, error) {
	address = normalizeGatewayAddress(address)
	if address == "" {
		return nil, fmt.Errorf("gateway address is empty")
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gateway %s: %w", address, err)
	}
	defer conn.Close()

	client := modelv1.NewModelGatewayClient(conn)
	resp, err := client.ListModels(ctx, &modelv1.ListModelsRequest{})
	if err != nil {
		return nil, err
	}

	byProvider := make(map[string][]catwalk.Model)
	for _, gm := range resp.GetModels() {
		if gm == nil || gm.GetRef() == nil {
			continue
		}
		provider := gm.GetRef().GetProvider()
		modelID := gm.GetRef().GetModel()
		if provider == "" || modelID == "" {
			continue
		}
		byProvider[provider] = append(byProvider[provider], gatewayModelToCatwalk(gm))
	}
	return byProvider, nil
}

func gatewayModelToCatwalk(gm *modelv1.Model) catwalk.Model {
	name := gm.GetName()
	if name == "" {
		name = gm.GetRef().GetModel()
	}
	m := catwalk.Model{
		ID:             gm.GetRef().GetModel(),
		Name:           name,
		ContextWindow:  gm.GetContextWindow(),
		CanReason:      gm.GetSupportsReasoning() || looksLikeReasoningModelID(gm.GetRef().GetModel()),
		SupportsImages: gm.GetSupportsAttachments(),
	}
	if gm.DefaultMaxOutputTokens != nil {
		m.DefaultMaxTokens = gm.GetDefaultMaxOutputTokens()
	}
	if m.ContextWindow == 0 {
		m.ContextWindow = 128_000
	}
	if m.DefaultMaxTokens == 0 {
		m.DefaultMaxTokens = 8192
	}
	if m.CanReason && len(m.ReasoningLevels) == 0 {
		m.ReasoningLevels = defaultReasoningLevels(m.ID)
		if m.DefaultReasoningEffort == "" && len(m.ReasoningLevels) > 0 {
			m.DefaultReasoningEffort = m.ReasoningLevels[0]
		}
	}
	return m
}

func looksLikeReasoningModelID(modelID string) bool {
	id := strings.ToLower(modelID)
	if i := strings.LastIndexByte(id, '/'); i >= 0 && i+1 < len(id) {
		id = id[i+1:]
	}
	switch {
	case strings.HasPrefix(id, "o1"), strings.HasPrefix(id, "o3"), strings.HasPrefix(id, "o4"):
		return true
	case strings.Contains(id, "gpt-5"):
		return !strings.Contains(id, "gpt-5-chat")
	case strings.Contains(id, "claude"):
		return true
	case strings.Contains(id, "gemini") && (strings.Contains(id, "thinking") || strings.Contains(id, "2.5") || strings.Contains(id, "3.")):
		return true
	case strings.Contains(id, "grok") && strings.Contains(id, "reasoning"):
		return true
	case strings.Contains(id, "deepseek") && (strings.Contains(id, "r1") || strings.Contains(id, "reasoner")):
		return true
	default:
		return false
	}
}

func defaultReasoningLevels(modelID string) []string {
	id := strings.ToLower(modelID)
	if i := strings.LastIndexByte(id, '/'); i >= 0 && i+1 < len(id) {
		id = id[i+1:]
	}
	// OpenAI-style effort levels (also used by many compat endpoints).
	if strings.Contains(id, "gpt-5") || strings.HasPrefix(id, "o1") || strings.HasPrefix(id, "o3") || strings.HasPrefix(id, "o4") {
		return []string{"low", "medium", "high", "xhigh"}
	}
	// Anthropic / others: empty levels → Crush shows a binary Thinking toggle.
	return nil
}

func resolveGatewayAddress(baseURL string, resolver Resolver) (string, error) {
	resolved, err := resolver.ResolveValue(baseURL)
	if err != nil {
		return "", err
	}
	address := normalizeGatewayAddress(resolved)
	if address == "" {
		return "", fmt.Errorf("empty gateway address")
	}
	return address, nil
}

// normalizeGatewayAddress turns values like "http://127.0.0.1:9090/v1" or
// "127.0.0.1:9090" into a bare host:port suitable for grpc.NewClient.
func normalizeGatewayAddress(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	// Drop path/query if present.
	if i := strings.IndexAny(s, "/?"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// host:port — if only host, append default port.
	if _, _, err := net.SplitHostPort(s); err != nil {
		// IPv6 without brackets or bare hostname.
		if !strings.Contains(s, ":") {
			return net.JoinHostPort(s, "9090")
		}
	}
	return s
}

// DefaultUltracoreAddress returns the default local gateway address.
func DefaultUltracoreAddress() string {
	return ultracore.DefaultAddress
}

// ProbeUltracoreGateway reports whether a gateway is reachable at address
// within a short timeout. Used to auto-wire providers when the daemon is up.
func ProbeUltracoreGateway(ctx context.Context, address string) bool {
	address = normalizeGatewayAddress(address)
	if address == "" {
		return false
	}
	d := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
