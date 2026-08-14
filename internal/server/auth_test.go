package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerRequiresConfiguredAuthToken(t *testing.T) {
	t.Parallel()

	srv := NewServer(nil, "tcp", "127.0.0.1:0", WithAuthToken("server-secret"))
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(httpSrv.Close)

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", authorization: "Basic server-secret", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", authorization: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid", authorization: "Bearer server-secret", wantStatus: http.StatusOK},
		{name: "case insensitive scheme", authorization: "bearer server-secret", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpSrv.URL+"/v1/health", nil)
			require.NoError(t, err)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}

			resp, err := httpSrv.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus == http.StatusUnauthorized {
				require.Equal(t, "Bearer", resp.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

func TestServerAllowsRequestsWithoutConfiguredAuthToken(t *testing.T) {
	t.Parallel()

	srv := NewServer(nil, "tcp", "127.0.0.1:0")
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(httpSrv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpSrv.URL+"/v1/health", nil)
	require.NoError(t, err)
	resp, err := httpSrv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
