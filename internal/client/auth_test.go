package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientSendsServerAuthToken(t *testing.T) {
	t.Parallel()

	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	c, err := NewClient(t.TempDir(), "tcp", u.Host, WithAuthToken("server-secret"))
	require.NoError(t, err)

	require.NoError(t, c.Health(t.Context()))
	require.Equal(t, "Bearer server-secret", authorization)
}

func TestClientOmitsAuthorizationWithoutServerAuthToken(t *testing.T) {
	t.Parallel()

	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	c, err := NewClient(t.TempDir(), "tcp", u.Host)
	require.NoError(t, err)

	require.NoError(t, c.Health(t.Context()))
	require.Empty(t, authorization)
}

func TestClientReportsUnauthorizedServer(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	c, err := NewClient(t.TempDir(), "tcp", u.Host, WithAuthToken("wrong"))
	require.NoError(t, err)

	require.ErrorIs(t, c.Health(t.Context()), ErrUnauthorized)
	_, err = c.VersionInfo(t.Context())
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClientSendsServerAuthTokenOnEventStream(t *testing.T) {
	t.Parallel()

	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	c, err := NewClient(t.TempDir(), "tcp", u.Host, WithAuthToken("server-secret"))
	require.NoError(t, err)

	_, err = c.SubscribeEvents(t.Context(), "workspace")
	require.NoError(t, err)
	require.Equal(t, "Bearer server-secret", authorization)
}
