package setup

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/skills"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type testTargetRepo struct {
	dir string
}

func newTestTargetRepo(t *testing.T) *testTargetRepo {
	dir := t.TempDir()
	initializeGitRepo(t, dir)
	t.Chdir(dir)
	return &testTargetRepo{dir: dir}
}

func initializeGitRepo(t *testing.T, dir string) {
	require.NoError(t, os.MkdirAll(dir, 0755))
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "config", "--local", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "config", "--local", "user.name", "Test User")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Test\n"), 0644))
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}

func (r *testTargetRepo) withInstalledSkill(t *testing.T, name, body string) {
	skillsDir := filepath.Join(r.dir, ".claude", "skills", name)
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	skillPath := filepath.Join(skillsDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte(body), 0644))
}

func (r *testTargetRepo) path() string {
	return r.dir
}

func (r *testTargetRepo) skillPath(name string) string {
	return filepath.Join(r.dir, ".claude", "skills", name, "SKILL.md")
}

type testSourceBranch struct {
	t            *testing.T
	branchName   string
	skills       map[string]string
	failFetchFor map[string]bool
	failDiscover bool
	server       *httptest.Server
}

func newTestSourceBranch(t *testing.T) *testSourceBranch {
	return &testSourceBranch{
		t:            t,
		branchName:   "main",
		skills:       make(map[string]string),
		failFetchFor: make(map[string]bool),
	}
}

func (s *testSourceBranch) onBranch(name string) *testSourceBranch {
	s.branchName = name
	return s
}

func (s *testSourceBranch) withRalphSkill(name, body string) *testSourceBranch {
	s.skills[name] = body
	return s
}

func (s *testSourceBranch) withNonRalphSkill(name, body string) *testSourceBranch {
	s.skills[name] = body
	return s
}

func (s *testSourceBranch) failsDiscovery() *testSourceBranch {
	s.failDiscover = true
	return s
}

func (s *testSourceBranch) failsFetchFor(name string) *testSourceBranch {
	s.failFetchFor[name] = true
	return s
}

func (s *testSourceBranch) branch() string {
	return s.branchName
}

func (s *testSourceBranch) startServer() {
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if s.failDiscover && strings.Contains(path, "contents/.claude/skills") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		if strings.Contains(path, "/repos/zon/ralph/contents/.claude/skills") {
			w.WriteHeader(http.StatusOK)
			var entries []string
			for name := range s.skills {
				entries = append(entries, `{"name": "`+name+`", "type": "dir"}`)
			}
			w.Write([]byte(`[` + strings.Join(entries, ",") + `]`))
			return
		}

		if strings.Contains(path, "/zon/ralph/refs/heads/") && strings.HasSuffix(path, "/SKILL.md") {
			parts := strings.Split(path, "/")
			for i, p := range parts {
				if p == "skills" && i+1 < len(parts) {
					skillName := parts[i+1]
					if s.failFetchFor[skillName] {
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
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func (s *testSourceBranch) close() {
	if s.server != nil {
		s.server.Close()
	}
}

func testDefaultBranch() string {
	return "main"
}

func testRawURL(branch, path string) string {
	return "https://raw.githubusercontent.com/zon/ralph/refs/heads/" + branch + "/" + path
}

func testRequireOK(t *testing.T, err error) {
	require.NoError(t, err)
}

func testRequireInstalled(t *testing.T, target *testTargetRepo, name string) {
	_, err := os.Stat(target.skillPath(name))
	require.NoError(t, err, "skill %q should be installed", name)
}

func testRequireNotInstalled(t *testing.T, target *testTargetRepo, name string) {
	_, err := os.Stat(target.skillPath(name))
	require.True(t, os.IsNotExist(err), "skill %q should not be installed", name)
}

func testRequireInstalledBody(t *testing.T, target *testTargetRepo, name, body string) {
	content, err := os.ReadFile(target.skillPath(name))
	require.NoError(t, err)
	require.Equal(t, body, string(content))
}

func testRequireEmpty(t *testing.T, target *testTargetRepo) {
	skillsDir := filepath.Join(target.path(), ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	require.Len(t, entries, 0, "target should have no skills installed")
}

func testRequireSkillLink(t *testing.T, target *testTargetRepo, name, expectedURL string) {
	content, err := os.ReadFile(target.skillPath(name))
	require.NoError(t, err)
	require.Contains(t, string(content), expectedURL, "skill should link to %s", expectedURL)
}

func testRequireError(t *testing.T, err error) {
	require.Error(t, err)
}

func testOutsideAnyRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
}

func TestSetSkills_InstallsRalphSkills(t *testing.T) {
	target := newTestTargetRepo(t)
	source := newTestSourceBranch(t).
		withRalphSkill("ralph-write-spec", "spec content").
		withNonRalphSkill("internal-tool", "tool content")
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills(source.branch())

	testRequireOK(t, err)
	testRequireInstalled(t, target, "ralph-write-spec")
	testRequireNotInstalled(t, target, "internal-tool")
}

func TestSetSkills_OverwritesExistingSkill(t *testing.T) {
	target := newTestTargetRepo(t)
	target.withInstalledSkill(t, "ralph-write-spec", "old")
	source := newTestSourceBranch(t).withRalphSkill("ralph-write-spec", "new")
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills(source.branch())

	testRequireOK(t, err)
	testRequireInstalledBody(t, target, "ralph-write-spec", "new")
}

func TestSetSkills_RemovesStaleRalphSkill(t *testing.T) {
	target := newTestTargetRepo(t)
	target.withInstalledSkill(t, "ralph-old-skill", "stale")
	source := newTestSourceBranch(t).withRalphSkill("ralph-write-spec", "spec content")
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills(source.branch())

	testRequireOK(t, err)
	testRequireNotInstalled(t, target, "ralph-old-skill")
}

func TestSetSkills_LeavesNonRalphSkillsUntouched(t *testing.T) {
	target := newTestTargetRepo(t)
	target.withInstalledSkill(t, "my-custom-skill", "mine")
	source := newTestSourceBranch(t).withRalphSkill("ralph-write-spec", "spec content")
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills(source.branch())

	testRequireOK(t, err)
	testRequireInstalledBody(t, target, "my-custom-skill", "mine")
}

func TestSetSkills_RewritesLinksToResolvedBranch(t *testing.T) {
	target := newTestTargetRepo(t)
	source := newTestSourceBranch(t).onBranch("v2").
		withRalphSkill("ralph-write-spec", "[spec](docs/formats/specs.md)")
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills("v2")

	testRequireOK(t, err)
	testRequireSkillLink(t, target, "ralph-write-spec", testRawURL("v2", "docs/formats/specs.md"))
}

func TestSetSkills_DiscoveryFailureWritesNothing(t *testing.T) {
	target := newTestTargetRepo(t)
	source := newTestSourceBranch(t)
	source.failDiscover = true
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills(testDefaultBranch())

	testRequireError(t, err)
	testRequireEmpty(t, target)
}

func TestSetSkills_FetchFailureWritesNothing(t *testing.T) {
	target := newTestTargetRepo(t)
	source := newTestSourceBranch(t).
		withRalphSkill("ralph-write-spec", "spec content")
	source.failFetchFor["ralph-write-spec"] = true
	source.startServer()
	defer source.close()

	oldClient := skills.HTTPClient()
	skills.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = source.server.Listener.Addr().String()
		req.URL.Scheme = "http"
		return source.server.Client().Transport.RoundTrip(req)
	})})
	defer skills.SetHTTPClient(oldClient)

	err := SetSkills(testDefaultBranch())

	testRequireError(t, err)
	testRequireEmpty(t, target)
}

func TestSetSkills_OutsideGitRepoErrors(t *testing.T) {
	testOutsideAnyRepo(t)

	err := SetSkills(testDefaultBranch())

	testRequireError(t, err)
}