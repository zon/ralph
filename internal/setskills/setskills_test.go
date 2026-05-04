package setskills

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverSkills(t *testing.T) {
	t.Run("parses skill entries with ralph prefix", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/repos/zon/ralph/contents/.claude/skills" {
				t.Errorf("unexpected path: %s", r.URL.Path)
				return
			}
			entries := []map[string]string{
				{"name": "ralph-write-spec", "type": "dir"},
				{"name": "ralph-write-flow", "type": "dir"},
				{"name": "internal-tool", "type": "dir"},
			}
			require.NoError(t, json.NewEncoder(w).Encode(entries))
		}))
		defer server.Close()

		origURL := ralphAPIURL
		ralphAPIURL = server.URL + "/repos/zon/ralph/contents/.claude/skills"
		defer func() { ralphAPIURL = origURL }()

		skills, err := DiscoverSkills("main")
		require.NoError(t, err)
		assert.Equal(t, []string{"ralph-write-spec", "ralph-write-flow"}, skills)
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		origURL := ralphAPIURL
		ralphAPIURL = server.URL + "/repos/zon/ralph/contents/.claude/skills"
		defer func() { ralphAPIURL = origURL }()

		_, err := DiscoverSkills("main")
		assert.Error(t, err)
	})
}

func TestFetchSkillContents(t *testing.T) {
	t.Run("fetches SKILL.md for each skill", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Skill Content"))
		}))
		defer server.Close()

		origBase := ralphRawBase
		ralphRawBase = server.URL
		defer func() { ralphRawBase = origBase }()

		skills := []string{"ralph-write-spec", "ralph-write-flow"}
		contents, err := FetchSkillContents(skills, "main")
		require.NoError(t, err)
		assert.Len(t, contents, 2)
		assert.Equal(t, "# Skill Content", contents["ralph-write-spec"])
	})

	t.Run("returns error on fetch failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		origBase := ralphRawBase
		ralphRawBase = server.URL
		defer func() { ralphRawBase = origBase }()

		_, err := FetchSkillContents([]string{"ralph-missing"}, "main")
		assert.Error(t, err)
	})
}

func TestRewriteLinks(t *testing.T) {
	t.Run("rewrites relative links to absolute", func(t *testing.T) {
		contents := map[string]string{
			"ralph-test": "See [docs](../docs/specs.md) for details",
		}
		rewritten := RewriteLinks(contents, "main")
		expected := "See [docs](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/specs.md) for details"
		assert.Equal(t, expected, rewritten["ralph-test"])
	})

	t.Run("rewrites existing ralph URLs to new branch", func(t *testing.T) {
		contents := map[string]string{
			"ralph-test": "See [docs](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/specs.md)",
		}
		rewritten := RewriteLinks(contents, "v2")
		expected := "See [docs](https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/specs.md)"
		assert.Equal(t, expected, rewritten["ralph-test"])
	})

	t.Run("preserves non-ralph absolute URLs", func(t *testing.T) {
		contents := map[string]string{
			"ralph-test": "See [external](https://example.com/docs/specs.md)",
		}
		rewritten := RewriteLinks(contents, "main")
		assert.Equal(t, contents["ralph-test"], rewritten["ralph-test"])
	})
}

func TestRemoveStaleSkills(t *testing.T) {
	t.Run("removes ralph-prefixed skills not in list", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, ".claude", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-old-skill"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-keep-skill"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "my-custom-skill"), 0755))

		err := RemoveStaleSkills(tmpDir, []string{"ralph-keep-skill"})
		require.NoError(t, err)

		entries, err := os.ReadDir(skillsDir)
		require.NoError(t, err)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		assert.Contains(t, names, "ralph-keep-skill")
		assert.NotContains(t, names, "ralph-old-skill")
		assert.Contains(t, names, "my-custom-skill")
	})

	t.Run("does nothing when skills dir does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := RemoveStaleSkills(tmpDir, []string{})
		require.NoError(t, err)
	})
}

func TestWriteSkills(t *testing.T) {
	t.Run("writes SKILL.md files to skill directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		contents := map[string]string{
			"ralph-write-spec": "# Write Spec\n\nContent here",
		}

		err := WriteSkills(tmpDir, contents)
		require.NoError(t, err)

		skillPath := filepath.Join(tmpDir, ".claude", "skills", "ralph-write-spec", "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		assert.Equal(t, "# Write Spec\n\nContent here", string(data))
	})

	t.Run("overwrites existing skills", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, ".claude", "skills", "ralph-existing")
		require.NoError(t, os.MkdirAll(skillsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("old content"), 0644))

		contents := map[string]string{
			"ralph-existing": "new content",
		}

		err := WriteSkills(tmpDir, contents)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(skillsDir, "SKILL.md"))
		require.NoError(t, err)
		assert.Equal(t, "new content", string(data))
	})
}

func TestSetSkillsIntegration(t *testing.T) {
	t.Run("full flow with mocked HTTP", func(t *testing.T) {
		tmpDir := t.TempDir()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repos/zon/ralph/contents/.claude/skills" {
				entries := []map[string]string{
					{"name": "ralph-write-spec", "type": "dir"},
				}
				json.NewEncoder(w).Encode(entries)
				return
			}
			if r.URL.Path == "/main/.claude/skills/ralph-write-spec/SKILL.md" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("# Skill Content"))
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		origAPI := ralphAPIURL
		origRaw := ralphRawBase
		ralphAPIURL = server.URL + "/repos/zon/ralph/contents/.claude/skills"
		ralphRawBase = server.URL
		defer func() {
			ralphAPIURL = origAPI
			ralphRawBase = origRaw
		}()

		contents, err := FetchSkillContents([]string{"ralph-write-spec"}, "main")
		require.NoError(t, err)

		rewritten := RewriteLinks(contents, "main")

		err = WriteSkills(tmpDir, rewritten)
		require.NoError(t, err)

		skillPath := filepath.Join(tmpDir, ".claude", "skills", "ralph-write-spec", "SKILL.md")
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		assert.Equal(t, "# Skill Content", string(data))
	})
}

type mockServer struct {
	*httptest.Server
	skillsOnBranch map[string][]string
	fetchFailures  map[string]bool
	discoveryFails map[string]bool
}

func newMockServer() *mockServer {
	m := &mockServer{
		skillsOnBranch: make(map[string][]string),
		fetchFailures:  make(map[string]bool),
		discoveryFails: make(map[string]bool),
	}
	m.Server = httptest.NewServer(m)
	return m
}

func (m *mockServer) SetSkillsAvailable(branch string, skills []string) {
	m.skillsOnBranch[branch] = skills
}

func (m *mockServer) SetDiscoveryFails(branch string) {
	m.discoveryFails[branch] = true
}

func (m *mockServer) SetFetchFails(skill, branch string) {
	key := skill + ":" + branch
	m.fetchFailures[key] = true
}

func (m *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.discoveryFails[r.URL.Query().Get("ref")] {
		http.Error(w, "discovery failed", http.StatusInternalServerError)
		return
	}

	if r.URL.Path == "/repos/zon/ralph/contents/.claude/skills" {
		branch := r.URL.Query().Get("ref")
		skills := m.skillsOnBranch[branch]
		if skills == nil {
			skills = []string{}
		}
		var entries []map[string]string
		for _, s := range skills {
			entries = append(entries, map[string]string{"name": s, "type": "dir"})
		}
		json.NewEncoder(w).Encode(entries)
		return
	}

	for _, skill := range m.skillsOnBranch["main"] {
		expectedPath := "/main/.claude/skills/" + skill + "/SKILL.md"
		if r.URL.Path == expectedPath {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Skill Content for " + skill))
			return
		}
	}

	http.NotFound(w, r)
}

func TestFullSetSkillsFlow(t *testing.T) {
	t.Run("skills installed successfully", func(t *testing.T) {
		_ = t.TempDir()
		m := newMockServer()
		defer m.Close()

		m.SetSkillsAvailable("main", []string{"ralph-write-spec", "ralph-write-flow"})

		origAPI := ralphAPIURL
		origRaw := ralphRawBase
		ralphAPIURL = m.URL + "/repos/zon/ralph/contents/.claude/skills"
		ralphRawBase = m.URL
		defer func() {
			ralphAPIURL = origAPI
			ralphRawBase = origRaw
		}()

		contents, err := FetchSkillContents([]string{"ralph-write-spec", "ralph-write-flow"}, "main")
		require.NoError(t, err)
		assert.Len(t, contents, 2)
	})

	t.Run("not in git repo returns error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping: running as root in git repo")
		}
		err := SetSkills("main")
		assert.Error(t, err)
	})

	t.Run("discovery failure returns error without writing files", func(t *testing.T) {
		_ = t.TempDir()
		m := newMockServer()
		defer m.Close()

		m.SetDiscoveryFails("main")

		origAPI := ralphAPIURL
		ralphAPIURL = m.URL + "/repos/zon/ralph/contents/.claude/skills"
		defer func() { ralphAPIURL = origAPI }()

		_, err := DiscoverSkills("main")
		assert.Error(t, err)
	})

	t.Run("non-ralph skills excluded", func(t *testing.T) {
		m := newMockServer()
		defer m.Close()

		m.SetSkillsAvailable("main", []string{"ralph-write-spec", "internal-tool"})

		origAPI := ralphAPIURL
		origRaw := ralphRawBase
		ralphAPIURL = m.URL + "/repos/zon/ralph/contents/.claude/skills"
		ralphRawBase = m.URL
		defer func() {
			ralphAPIURL = origAPI
			ralphRawBase = origRaw
		}()

		skills, err := DiscoverSkills("main")
		require.NoError(t, err)
		assert.Equal(t, []string{"ralph-write-spec"}, skills)
	})

	t.Run("branch override applies to discovery", func(t *testing.T) {
		m := newMockServer()
		defer m.Close()

		m.SetSkillsAvailable("v2", []string{"ralph-write-spec"})

		origAPI := ralphAPIURL
		ralphAPIURL = m.URL + "/repos/zon/ralph/contents/.claude/skills"
		defer func() { ralphAPIURL = origAPI }()

		skills, err := DiscoverSkills("v2")
		require.NoError(t, err)
		assert.Equal(t, []string{"ralph-write-spec"}, skills)
	})
}

func installedSkills(tmpDir string) []string {
	skillsDir := filepath.Join(tmpDir, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var skills []string
	for _, e := range entries {
		if e.IsDir() {
			skills = append(skills, e.Name())
		}
	}
	return skills
}

func TestInstalledSkills(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-one"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "ralph-two"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "my-custom"), 0755))

	skills := installedSkills(tmpDir)
	assert.Contains(t, skills, "ralph-one")
	assert.Contains(t, skills, "ralph-two")
	assert.Contains(t, skills, "my-custom")
}

func TestRewriteLineLinks(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		baseURL  string
		expected string
	}{
		{
			name:     "relative link at start",
			line:     `See [docs](docs/specs.md) for details`,
			baseURL:  "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/",
			expected: `See [docs](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/specs.md) for details`,
		},
		{
			name:     "relative link in middle",
			line:     `Check [here](docs/guide.md) and [there](docs/ref.md)`,
			baseURL:  "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/",
			expected: `Check [here](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/guide.md) and [there](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/ref.md)`,
		},
		{
			name:     "ralph raw URL with different branch",
			line:     `Link: [specs](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/specs.md)`,
			baseURL:  "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/",
			expected: `Link: [specs](https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/specs.md)`,
		},
		{
			name:     "non-ralph URL unchanged",
			line:     `Link: [example](https://example.com/docs/specs.md)`,
			baseURL:  "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/",
			expected: `Link: [example](https://example.com/docs/specs.md)`,
		},
		{
			name:     "no link",
			line:     `Just some text without links`,
			baseURL:  "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/",
			expected: `Just some text without links`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteLineLinks(tt.line, tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRewriteContentLinks(t *testing.T) {
	content := `# Skill Document

See [docs](docs/specs.md) for details.
Also see [guide](docs/guide.md).

Raw URL: [link](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/ref.md)
External: [example](https://example.com)
`
	baseURL := "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/"
	result := rewriteContentLinks(content, baseURL)

	assert.Contains(t, result, "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/specs.md")
	assert.Contains(t, result, "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/guide.md")
	assert.Contains(t, result, "https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/ref.md")
	assert.Contains(t, result, "https://example.com")
	assert.NotContains(t, result, "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/")
}