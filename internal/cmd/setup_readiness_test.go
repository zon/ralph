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
		assert.Contains(t, buf.String(), "git is ready")
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
		assert.Contains(t, buf.String(), "GitHub CLI is ready")
	})

	t.Run("succeeds with GITHUB_TOKEN set", func(t *testing.T) {
		writeFakeGHCLIScript(t, `exit 1`)
		t.Setenv("GITHUB_TOKEN", "ghp_env_token")
		client, buf := newSetupReadinessClient(t)
		err := client.ConfirmGitHubCLIReady()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "GitHub CLI is ready")
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
		assert.Contains(t, buf.String(), "OpenCode is ready")
	})
}
