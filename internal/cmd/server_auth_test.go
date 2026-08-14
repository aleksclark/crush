package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfiguredServerToken(t *testing.T) {
	t.Setenv("CRUSH_SERVER_TOKEN", "environment-token")

	require.Equal(t, "environment-token", configuredServerToken(""))
	require.Equal(t, "flag-token", configuredServerToken("flag-token"))
}

func TestConsumeServerTokenRemovesItFromEnvironment(t *testing.T) {
	t.Setenv("CRUSH_SERVER_TOKEN", "environment-token")

	require.Equal(t, "environment-token", consumeServerToken(""))
	_, exists := os.LookupEnv("CRUSH_SERVER_TOKEN")
	require.False(t, exists)
}

func TestWithoutServerToken(t *testing.T) {
	env := []string{"HOME=/tmp/home", "CRUSH_SERVER_TOKEN=secret", "PATH=/bin"}

	require.Equal(t, []string{"HOME=/tmp/home", "PATH=/bin"}, withoutServerToken(env))
}

func TestServerTokenFlagIsPersistent(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("server-token")
	require.NotNil(t, flag)
}

func TestProbeHealthSendsServerAuthToken(t *testing.T) {
	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	hostURL := &url.URL{Scheme: "tcp", Host: u.Host}
	require.NoError(t, probeHealth(t.Context(), srv.Client(), srv.URL+"/v1/health", hostURL, "server-secret"))
	require.Equal(t, "Bearer server-secret", authorization)
}
