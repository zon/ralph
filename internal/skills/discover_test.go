package skills

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestDiscover_OnlyRalphPrefixedSkillsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/zon/ralph/contents/.claude/skills", r.URL.Path)
		assert.Equal(t, "main", r.URL.Query().Get("ref"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"name": "ralph-write-spec", "type": "dir"},
			{"name": "internal-tool", "type": "dir"},
			{"name": ".github", "type": "dir"}
		]`))
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return server.Client().Transport.RoundTrip(req)
	})}
	defer func() { httpClient = oldClient }()

	names, err := Discover("main")
	require.NoError(t, err, "Discover failed")

	assert.Contains(t, names, "ralph-write-spec", "ralph-write-spec should be in the result")
	assert.NotContains(t, names, "internal-tool", "internal-tool should not be in the result")
	assert.NotContains(t, names, ".github", ".github should not be in the result")
}

func TestDiscover_ReturnsErrorOnNetworkFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	server.Close()

	oldClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})}
	defer func() { httpClient = oldClient }()

	_, err := Discover("main")
	require.Error(t, err, "Discover should return error on network failure")
}

func TestDiscover_ReturnsErrorOnNon200Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return server.Client().Transport.RoundTrip(req)
	})}
	defer func() { httpClient = oldClient }()

	_, err := Discover("main")
	require.Error(t, err, "Discover should return error on non-200 response")
}