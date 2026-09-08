package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/output"
)

func writeFakeTool(t *testing.T, name, body string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755))
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)
}

func initRepoOnly(t *testing.T, dir string) {
	t.Helper()
	c := exec.Command("git", "init", "-b", "main")
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git init: %s", out)
}

func newSetupReadinessClient(t *testing.T) (*setupLocalReadinessClient, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	return &setupLocalReadinessClient{out: output.NewClient(buf, io.Discard, false)}, buf
}

func TestSetupLocalReadinessClientConfirmGitReady(t *testing.T) {
	t.Run("fails when git is not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmGitReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git is not installed")
	})

	t.Run("fails when current directory is not inside a git repository", func(t *testing.T) {
		t.Chdir(t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmGitReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not inside a git repository")
	})

	t.Run("fails when git identity is not configured", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
		t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
		initRepoOnly(t, dir)
		t.Chdir(dir)
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmGitReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configure a git identity")
	})

	t.Run("succeeds when git, repository, and identity are ready", func(t *testing.T) {
		dir := t.TempDir()
		initRepoOnly(t, dir)
		for _, args := range [][]string{
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
		} {
			c := exec.Command("git", args...)
			c.Dir = dir
			require.NoError(t, c.Run())
		}
		t.Chdir(dir)
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmGitReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "\u2713 git ready")
	})
}

func TestSetupLocalReadinessClientConfirmGitHubCLIReady(t *testing.T) {
	t.Run("fails when gh is not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmGitHubCLIReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gh is not installed")
	})

	t.Run("fails when no token is available", func(t *testing.T) {
		writeFakeGHCLIScript(t, `exit 1`)
		t.Setenv("GITHUB_TOKEN", "")
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmGitHubCLIReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no token available")
	})

	t.Run("succeeds with a gh login token", func(t *testing.T) {
		writeFakeGHCLIScript(t, `printf 'ghp_cli_token\n'`)
		t.Setenv("GITHUB_TOKEN", "")
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmGitHubCLIReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "\u2713 gh ready")
	})

	t.Run("succeeds with GITHUB_TOKEN set", func(t *testing.T) {
		writeFakeGHCLIScript(t, `exit 1`)
		t.Setenv("GITHUB_TOKEN", "ghp_env_token")
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmGitHubCLIReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "\u2713 gh ready")
	})
}

func TestSetupLocalReadinessClientConfirmOpenCodeReady(t *testing.T) {
	t.Run("fails when opencode is not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmOpenCodeReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "opencode is not installed")
	})

	t.Run("fails when auth.json is missing", func(t *testing.T) {
		writeFakeTool(t, "opencode", `exit 0`)
		t.Setenv("HOME", t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmOpenCodeReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.json")
	})

	t.Run("fails when auth.json holds no credentials", func(t *testing.T) {
		writeFakeTool(t, "opencode", `exit 0`)
		home := t.TempDir()
		t.Setenv("HOME", home)
		authDir := filepath.Join(home, ".local", "share", "opencode")
		require.NoError(t, os.MkdirAll(authDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(authDir, "auth.json"), nil, 0644))
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmOpenCodeReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.json is empty")
	})

	t.Run("succeeds when auth.json holds credentials", func(t *testing.T) {
		writeFakeTool(t, "opencode", `exit 0`)
		home := t.TempDir()
		t.Setenv("HOME", home)
		authDir := filepath.Join(home, ".local", "share", "opencode")
		require.NoError(t, os.MkdirAll(authDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(authDir, "auth.json"), []byte(`{"tokens":{}}`), 0644))
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmOpenCodeReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "\u2713 opencode ready")
	})
}

func TestSetupLocalReadinessClientConfirmKubectlReady(t *testing.T) {
	t.Run("fails when kubectl is not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmKubectlReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kubectl is not installed")
	})

	t.Run("succeeds when kubectl is installed", func(t *testing.T) {
		writeFakeTool(t, "kubectl", `exit 0`)
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmKubectlReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "\u2713 kubectl ready")
	})
}

func TestSetupLocalReadinessClientConfirmArgoReady(t *testing.T) {
	t.Run("fails when argo is not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		client, _ := newSetupReadinessClient(t)
		err := client.ConfirmArgoReady()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "argo CLI is not installed")
	})

	t.Run("succeeds when argo is installed", func(t *testing.T) {
		writeFakeTool(t, "argo", `exit 0`)
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmArgoReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "\u2713 argo ready")
	})
}

func TestSetupLocalReadinessClientPrintsFailedChecks(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	tests := []struct {
		name    string
		cmd     string
		confirm func(client *setupLocalReadinessClient) error
	}{
		{
			name: "git not installed",
			cmd:  "git",
			confirm: func(client *setupLocalReadinessClient) error {
				return client.ConfirmGitReady()
			},
		},
		{
			name: "gh not installed",
			cmd:  "gh",
			confirm: func(client *setupLocalReadinessClient) error {
				return client.ConfirmGitHubCLIReady()
			},
		},
		{
			name: "opencode not installed",
			cmd:  "opencode",
			confirm: func(client *setupLocalReadinessClient) error {
				return client.ConfirmOpenCodeReady()
			},
		},
		{
			name: "kubectl not installed",
			cmd:  "kubectl",
			confirm: func(client *setupLocalReadinessClient) error {
				return client.ConfirmKubectlReady()
			},
		},
		{
			name: "argo not installed",
			cmd:  "argo",
			confirm: func(client *setupLocalReadinessClient) error {
				return client.ConfirmArgoReady()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outBuf := &bytes.Buffer{}
			errBuf := &bytes.Buffer{}
			client := &setupLocalReadinessClient{out: output.NewClient(outBuf, errBuf, false)}
			err := tt.confirm(client)
			require.Error(t, err)
			assert.Contains(t, errBuf.String(), "\u2717 "+tt.cmd+" not ready")
			assert.Empty(t, outBuf.String())
		})
	}
}

func TestSetupLocalReadinessClientFailureReportsReason(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	client := &setupLocalReadinessClient{out: output.NewClient(outBuf, errBuf, false)}
	err := client.ConfirmGitReady()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git is not installed")
}
