package skills

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTransport struct {
	old      http.RoundTripper
	serverURL string
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.github.com" || strings.HasPrefix(req.URL.Host, "api.github.com") {
		req = cloneRequest(req)
		req.URL.Scheme = "http"
		req.URL.Host = tt.serverURL
		req.URL.Path = strings.Replace(req.URL.Path, "api.github.com", "", 1)
		req.URL.RawPath = ""
	}
	return tt.old.RoundTrip(req)
}

func cloneRequest(req *http.Request) *http.Request {
	newReq := *req
	newReq.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	return &newReq
}

func TestDiscover(t *testing.T) {
	t.Run("returns only ralph-prefixed skills", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/zon/ralph/contents/.claude/skills", r.URL.Path)
			assert.Equal(t, "v2", r.URL.Query().Get("ref"))

			contents := []GitHubContent{
				{Name: "ralph-write-spec", Type: "dir"},
				{Name: "ralph-write-flow", Type: "dir"},
				{Name: "internal-tool", Type: "dir"},
				{Name: "README.md", Type: "file"},
			}
			json.NewEncoder(w).Encode(contents)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		oldClient := httpClient
		httpClient = &http.Client{Transport: &testTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		skills, err := Discover("v2")
		require.NoError(t, err)
		assert.Equal(t, []string{"ralph-write-spec", "ralph-write-flow"}, skills)
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		oldClient := httpClient
		httpClient = &http.Client{Transport: &testTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		_, err := Discover("main")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDiscoveryFailed)
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		oldClient := httpClient
		httpClient = &http.Client{Transport: &testTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		_, err := Discover("main")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDiscoveryFailed)
	})
}

func TestDiscover_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	skills, err := Discover("main")
	if err != nil && !strings.Contains(err.Error(), "connection refused") {
		require.NoError(t, err)
	}
	if len(skills) > 0 {
		assert.NotEmpty(t, skills)
		for _, s := range skills {
			assert.True(t, strings.HasPrefix(s, "ralph-"))
		}
	}
}