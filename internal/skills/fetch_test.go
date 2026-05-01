package skills

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	t.Run("fetches skill content and rewrites links", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/ralph-write-spec/SKILL.md")

			content := `---
name: ralph-write-spec
---

# Write Spec

See docs/planning/specs.md for details.
`
			w.Write([]byte(content))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &fetchTestTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		result, err := Fetch("ralph-write-spec", "v2")
		require.NoError(t, err)
		assert.Contains(t, result, "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/")
		assert.Contains(t, result, "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/planning/specs.md")
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &fetchTestTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		_, err := Fetch("ralph-nonexistent", "main")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFetchFailed)
	})
}

func TestFetchAll(t *testing.T) {
	t.Run("fetches multiple skills", func(t *testing.T) {
		skills := map[string]string{
			"ralph-write-spec": `# SKILL.md content for ralph-write-spec`,
			"ralph-write-flow": `# SKILL.md content for ralph-write-flow`,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			for name, content := range skills {
				if strings.Contains(path, name) {
					w.Write([]byte(content))
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &fetchTestTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		results, err := FetchAll([]string{"ralph-write-spec", "ralph-write-flow"}, "main")
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns error if any fetch fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")
		oldClient := httpClient
		httpClient = &http.Client{Transport: &fetchTestTransport{old: http.DefaultTransport, serverURL: serverURL}}
		defer func() { httpClient = oldClient }()

		_, err := FetchAll([]string{"ralph-write-spec", "ralph-nonexistent"}, "main")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFetchFailed)
	})
}

func TestRewriteLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		branch   string
		expected string
	}{
		{
			name:     "relative link expanded to absolute",
			content:  "See docs/planning/specs.md for details",
			branch:   "main",
			expected: "See https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/planning/specs.md for details",
		},
		{
			name:     "ralph raw URL branch replaced",
			content:  "Link: https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/planning/specs.md",
			branch:   "v2",
			expected: "Link: https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/planning/specs.md",
		},
		{
			name:     "non-ralph URL unchanged",
			content:  "See https://example.com/docs/planning/specs.md",
			branch:   "main",
			expected: "See https://example.com/docs/planning/specs.md",
		},
		{
			name:     "no links unchanged",
			content:  "Just plain text content",
			branch:   "main",
			expected: "Just plain text content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RewriteLinks(tt.content, tt.branch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type fetchTestTransport struct {
	old       http.RoundTripper
	serverURL string
}

func (ft *fetchTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.Host, "raw.githubusercontent.com") {
		req = cloneRequest(req)
		req.URL.Scheme = "http"
		req.URL.Host = ft.serverURL
		req.URL.Path = strings.Replace(req.URL.Path, "raw.githubusercontent.com", "", 1)
		req.URL.RawPath = ""
	}
	return ft.old.RoundTrip(req)
}