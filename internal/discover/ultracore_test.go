package discover

import (
	"context"
	"net"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	modelv1 "github.com/aleksclark/ultracore/gen/proto/ultracore/model/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockGateway struct {
	modelv1.UnimplementedModelGatewayServer
	models []*modelv1.Model
}

func (m *mockGateway) ListModels(_ context.Context, req *modelv1.ListModelsRequest) (*modelv1.ListModelsResponse, error) {
	out := make([]*modelv1.Model, 0, len(m.models))
	for _, model := range m.models {
		if req.Provider != "" && model.GetRef().GetProvider() != req.Provider {
			continue
		}
		out = append(out, model)
	}
	return &modelv1.ListModelsResponse{Models: out}, nil
}

func startMockGateway(t *testing.T, models []*modelv1.Model) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := grpc.NewServer()
	modelv1.RegisterModelGatewayServer(s, &mockGateway{models: models})
	go s.Serve(lis)
	t.Cleanup(func() {
		s.Stop()
		lis.Close()
	})
	return lis.Addr().String()
}

func sampleModels() []*modelv1.Model {
	maxTok := int64(4096)
	return []*modelv1.Model{
		{
			Ref:                    &modelv1.ModelRef{Provider: "anthropic", Model: "claude-haiku-4-5"},
			Name:                   "Claude Haiku 4.5",
			ContextWindow:          200000,
			DefaultMaxOutputTokens: &maxTok,
			SupportsReasoning:      true,
			SupportsAttachments:    true,
		},
		{
			Ref:           &modelv1.ModelRef{Provider: "openai", Model: "gpt-4.1-mini"},
			Name:          "GPT-4.1 mini",
			ContextWindow: 128000,
		},
		{
			Ref:   &modelv1.ModelRef{Provider: "openai", Model: "gpt-4o"},
			Name:  "GPT-4o",
			// zero context -> default applied
		},
	}
}

func TestNormalizeGatewayAddress(t *testing.T) {
	require.Equal(t, "127.0.0.1:9090", normalizeGatewayAddress("127.0.0.1:9090"))
	require.Equal(t, "127.0.0.1:9090", normalizeGatewayAddress("http://127.0.0.1:9090"))
	require.Equal(t, "127.0.0.1:9090", normalizeGatewayAddress("http://127.0.0.1:9090/v1"))
	require.Equal(t, "127.0.0.1:9090", normalizeGatewayAddress("https://127.0.0.1:9090/foo?x=1"))
	require.Equal(t, "localhost:9090", normalizeGatewayAddress("localhost"))
	require.Equal(t, "", normalizeGatewayAddress("  "))
}

func TestDiscoverUltracoreModels(t *testing.T) {
	addr := startMockGateway(t, sampleModels())

	models, err := DiscoverUltracoreModels(context.Background(), Config{
		ID:      "ultracore",
		BaseURL: addr,
	}, &mockResolver{})
	require.NoError(t, err)
	require.Len(t, models, 3)

	// Filter by provider id
	models, err = DiscoverUltracoreModels(context.Background(), Config{
		ID:      "anthropic",
		BaseURL: addr,
	}, &mockResolver{})
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "claude-haiku-4-5", models[0].ID)
	require.Equal(t, "Claude Haiku 4.5", models[0].Name)
	require.Equal(t, int64(200000), models[0].ContextWindow)
	require.Equal(t, int64(4096), models[0].DefaultMaxTokens)
	require.True(t, models[0].CanReason)
	require.True(t, models[0].SupportsImages)
}

func TestDiscoverUltracoreModels_ExistingWin(t *testing.T) {
	addr := startMockGateway(t, sampleModels())
	models, err := DiscoverUltracoreModels(context.Background(), Config{
		ID:      "openai",
		BaseURL: addr,
		ExistingModels: []catwalk.Model{{
			ID:   "gpt-4o",
			Name: "My GPT-4o",
		}},
	}, &mockResolver{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(models), 2)
	require.Equal(t, "gpt-4o", models[0].ID)
	require.Equal(t, "My GPT-4o", models[0].Name)
}

func TestListUltracoreGatewayModels(t *testing.T) {
	addr := startMockGateway(t, sampleModels())
	by, err := ListUltracoreGatewayModels(context.Background(), addr)
	require.NoError(t, err)
	require.Len(t, by, 2)
	require.Len(t, by["anthropic"], 1)
	require.Len(t, by["openai"], 2)
	// default context applied
	var zeroCW catwalk.Model
	for _, m := range by["openai"] {
		if m.ID == "gpt-4o" {
			zeroCW = m
		}
	}
	require.Equal(t, int64(128_000), zeroCW.ContextWindow)
	require.Equal(t, int64(8192), zeroCW.DefaultMaxTokens)
}

func TestIsKnownCustomProvider_Ultracore(t *testing.T) {
	require.True(t, IsKnownCustomProvider("ultracore"))
}
