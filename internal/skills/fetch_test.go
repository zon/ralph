package skills

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fetchRewriteTransport struct {
	old       http.RoundTripper
	serverURL string
}

func (rt *fetchRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "raw.githubusercontent.com" {
		newReq := *req
		newReq.URL.Scheme = "http"
		newReq.URL.Host = rt.serverURL
		req = &newReq
	}
	return rt.old.RoundTrip(req)
}

func TestFetchAll(t *testing.T) {
	t.Run("fetches each named skill", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Contains(t, r.URL.String(), "ralph-write-spec")
			require.Contains(t, r.URL.String(), "SKILL.md")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("skill content"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &fetchRewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		skills, err := FetchAll("main", []string{"ralph-write-spec"})
		require.NoError(t, err)
		require.Len(t, skills, 1)
		require.Equal(t, "ralph-write-spec", skills[0].Name)
		require.Equal(t, "skill content", skills[0].Content)
	})

	t.Run("fetch failure returns error and no partial result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &fetchRewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		_, err := FetchAll("main", []string{"ralph-write-spec"})
		require.Error(t, err)
	})
}

func TestRewriteLinks(t *testing.T) {
	t.Run("relative link rewritten to full raw URL", func(t *testing.T) {
		input := "Some content with relative [link](docs/formats/specs.md)"
		output := rewriteLinks(input, "main")
		require.Equal(t, "Some content with relative [link](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/specs.md)", output)
	})

	t.Run("existing ralph raw URL branch updated", func(t *testing.T) {
		input := "Content with ralph raw https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/spec.md"
		output := rewriteLinks(input, "v2")
		require.Equal(t, "Content with ralph raw https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/spec.md", output)
	})

	t.Run("non-ralph absolute URLs unchanged", func(t *testing.T) {
		input := "Content with external link https://example.com/docs/spec.md"
		output := rewriteLinks(input, "main")
		require.Equal(t, input, output)
	})
}