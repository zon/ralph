package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/skills"
)

func TestSetSkillsCmd_Run_NotInGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	origCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origCwd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cmd := &SetSkillsCmd{}
	err = cmd.Run()

	require.Error(t, err)
	assert.ErrorIs(t, err, skills.ErrNotInGitRepo)
}

func TestSetSkillsCmd_Run_InGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	_, err = exec.Command("git", "init", tmpDir).CombinedOutput()
	require.NoError(t, err)

	origCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origCwd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	oldClient := skills.HTTPClient()
	defer skills.SetHTTPClient(oldClient)

	skills.SetHTTPClient(nil)

	cmd := &SetSkillsCmd{Branch: "main"}
	err = cmd.Run()

	assert.NoError(t, err)
}

func TestSetSkillsCmd_BranchFlag(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	_, err = exec.Command("git", "init", tmpDir).CombinedOutput()
	require.NoError(t, err)

	origCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origCwd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/zon/ralph/contents/.claude/skills" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"name": "ralph-write-spec", "type": "dir"}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/ralph-write-spec/SKILL.md") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# SKILL.md content"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := skills.HTTPClient()
	defer skills.SetHTTPClient(oldClient)

	skills.SetHTTPClient(&http.Client{Transport: &cmdTestTransport{old: http.DefaultTransport, serverURL: serverURL}})

	cmd := &SetSkillsCmd{Branch: "v2"}
	err = cmd.Run()

	assert.NoError(t, err)
}

type cmdTestTransport struct {
	old       http.RoundTripper
	serverURL string
}

func (ct *cmdTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.github.com" || strings.HasPrefix(req.URL.Host, "api.github.com") ||
		strings.HasPrefix(req.URL.Host, "raw.githubusercontent.com") {
		req = cloneTestRequest(req)
		req.URL.Scheme = "http"
		req.URL.Host = ct.serverURL
		if strings.HasPrefix(req.URL.Path, "/repos/") {
			req.URL.Path = req.URL.Path
		}
		req.URL.RawPath = ""
	}
	return ct.old.RoundTrip(req)
}

func cloneTestRequest(req *http.Request) *http.Request {
	newReq := *req
	newReq.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	return &newReq
}