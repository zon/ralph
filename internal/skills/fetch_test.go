package skills

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type srcBranch struct {
	t       *testing.T
	branch  string
	skills  map[string]string
	failFor map[string]bool
}

func newSrcBranch(t *testing.T) *srcBranch {
	return &srcBranch{
		t:       t,
		branch:  "main",
		skills:  make(map[string]string),
		failFor: make(map[string]bool),
	}
}

func (s *srcBranch) onBranch(branch string) *srcBranch {
	s.branch = branch
	return s
}

func (s *srcBranch) with(skill *skillBuilder) *srcBranch {
	s.skills[skill.name] = skill.body
	return s
}

func (s *srcBranch) failsFetchFor(name string) *srcBranch {
	s.failFor[name] = true
	return s
}

type skillBuilder struct {
	name string
	body string
}

func aRalphSkill(name string) *skillBuilder {
	return &skillBuilder{name: name}
}

func (s *skillBuilder) withBody(body string) *skillBuilder {
	s.body = body
	return s
}

func (s *srcBranch) fetchAll(names ...string) ([]Skill, error) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/zon/ralph/refs/heads/") && strings.HasSuffix(r.URL.Path, "/SKILL.md") {
			parts := strings.Split(r.URL.Path, "/")
			for i, p := range parts {
				if p == "skills" && i+1 < len(parts) {
					skillName := parts[i+1]
					if s.failFor[skillName] {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if body, ok := s.skills[skillName]; ok {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(body))
						return
					}
				}
			}
		}
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

	return FetchAll(s.branch, names)
}

func TestFetchAll_RewritesRelativeLinks(t *testing.T) {
	source := newSrcBranch(t).onBranch("main")
	source.with(aRalphSkill("ralph-write-spec").withBody("[spec](docs/formats/specs.md)"))

	skills, err := source.fetchAll("ralph-write-spec")
	require.NoError(t, err, "FetchAll failed")

	require.Len(t, skills, 1)
	assert.Equal(t, "ralph-write-spec", skills[0].Name)
	require.Len(t, skills[0].Paths, 1)
	assert.Equal(t, "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/specs.md", skills[0].Paths[0])
}

func TestFetchAll_UpdatesExistingRalphURLs(t *testing.T) {
	source := newSrcBranch(t).onBranch("v2")
	source.with(aRalphSkill("ralph-write-spec").withBody("[spec](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/specs.md)"))

	skills, err := source.fetchAll("ralph-write-spec")
	require.NoError(t, err, "FetchAll failed")

	require.Len(t, skills, 1)
	assert.Equal(t, "ralph-write-spec", skills[0].Name)
	require.Len(t, skills[0].Paths, 1)
	assert.Equal(t, "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/formats/specs.md", skills[0].Paths[0])
}

func TestFetchAll_LeavesNonRalphURLsUnchanged(t *testing.T) {
	source := newSrcBranch(t).onBranch("main")
	source.with(aRalphSkill("ralph-write-spec").withBody("[spec](https://example.com/docs/guide.md)"))

	skills, err := source.fetchAll("ralph-write-spec")
	require.NoError(t, err, "FetchAll failed")

	require.Len(t, skills, 1)
	assert.Equal(t, "ralph-write-spec", skills[0].Name)
	require.Len(t, skills[0].Paths, 0)
}

func TestFetchAll_FetchFailureAbortsBatch(t *testing.T) {
	source := newSrcBranch(t)
	source.with(aRalphSkill("ralph-write-spec"))
	source.with(aRalphSkill("ralph-write-flow"))
	source.failsFetchFor("ralph-write-flow")

	_, err := source.fetchAll("ralph-write-spec", "ralph-write-flow")

	require.Error(t, err)
}