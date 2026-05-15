package skills

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type rewriteTransport struct {
	old       http.RoundTripper
	serverURL string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.github.com" {
		newReq := *req
		newReq.URL.Scheme = "http"
		newReq.URL.Host = rt.serverURL
		req = &newReq
	}
	return rt.old.RoundTrip(req)
}

func TestDiscover(t *testing.T) {
	t.Run("only ralph-prefixed skills returned", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/repos/zon/ralph/contents/.claude/skills", r.URL.Path)
			require.Equal(t, "main", r.URL.Query().Get("ref"))

			entries := []apiEntry{
				{Name: "ralph-write-spec", Type: "dir"},
				{Name: "internal-tool", Type: "dir"},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(entries)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		names, err := Discover("main")
		require.NoError(t, err)
		require.Contains(t, names, "ralph-write-spec")
		require.NotContains(t, names, "internal-tool")
	})

	t.Run("discovery failure returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		_, err := Discover("main")
		require.Error(t, err)
	})
}