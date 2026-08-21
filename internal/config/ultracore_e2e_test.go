//go:build e2e

package config

import (
	"context"
	"os"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/env"
	"github.com/stretchr/testify/require"
)

// Live gateway smoke test: CRUSH_ULTRACORE_E2E=1 go test -tags e2e -run Live
func TestLiveUltracoreExpand(t *testing.T) {
	if os.Getenv("CRUSH_ULTRACORE_E2E") == "" {
		t.Skip("set CRUSH_ULTRACORE_E2E=1")
	}
	addr := os.Getenv("ULTRACORE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9090"
	}
	cfg := &Config{
		Providers: csync.NewMapFrom(map[string]ProviderConfig{
			"ultracore": {
				Type:    "ultracore",
				BaseURL: addr,
				Name:    "Ultracore",
			},
		}),
		Options: &Options{DisableDefaultProviders: true},
	}
	cfg.setDefaults("/tmp", "")
	e := env.NewFromMap(map[string]string{})
	resolver := NewShellVariableResolver(e)
	err := cfg.configureProviders(context.Background(), testStore(cfg), e, resolver, nil)
	require.NoError(t, err)
	require.True(t, cfg.IsConfigured())
	require.GreaterOrEqual(t, cfg.Providers.Len(), 2)

	t.Logf("providers=%d", cfg.Providers.Len())
	for id, pc := range cfg.Providers.Seq2() {
		t.Logf("  %s type=%s models=%d", id, pc.Type, len(pc.Models))
		require.Equal(t, catwalk.Type("ultracore"), pc.Type)
		require.NotEmpty(t, pc.Models)
	}
}
