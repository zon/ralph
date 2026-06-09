package github

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFakeGHScript writes a fake gh executable script to a temp directory
// and prepends that directory to PATH so that exec.LookPath resolves it
// instead of the real gh CLI.
func writeFakeGHScript(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755))
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)
}

func TestGH_IsReady(t *testing.T) {
	t.Run("both version and auth succeed", func(t *testing.T) {
		writeFakeGHScript(t, `case "$1" in --version) exit 0;; auth) exit 0;; *) exit 1;; esac`)
		g := NewGH(nil)
		assert.True(t, g.IsReady())
	})

	t.Run("version fails", func(t *testing.T) {
		writeFakeGHScript(t, `exit 1`)
		g := NewGH(nil)
		assert.False(t, g.IsReady())
	})

	t.Run("auth fails", func(t *testing.T) {
		writeFakeGHScript(t, `case "$1" in --version) exit 0;; *) exit 1;; esac`)
		g := NewGH(nil)
		assert.False(t, g.IsReady())
	})
}

func TestGH_FindExistingPR(t *testing.T) {
	t.Run("returns URL when present", func(t *testing.T) {
		writeFakeGHScript(t, `printf '[{"url":"https://github.com/owner/repo/pull/123"}]\nhttps://github.com/owner/repo/pull/123\n'`)
		g := NewGH(nil)
		url, err := g.FindExistingPR("my-branch")
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/owner/repo/pull/123", url)
	})

	t.Run("returns empty when url field missing", func(t *testing.T) {
		writeFakeGHScript(t, `echo '[{"number":1}]'`)
		g := NewGH(nil)
		url, err := g.FindExistingPR("my-branch")
		require.NoError(t, err)
		assert.Empty(t, url)
	})

	t.Run("returns empty when no http line", func(t *testing.T) {
		writeFakeGHScript(t, `echo '[{"url":"not-http-value"}]'`)
		g := NewGH(nil)
		url, err := g.FindExistingPR("my-branch")
		require.NoError(t, err)
		assert.Empty(t, url)
	})
}

func TestGH_ListCollaborators(t *testing.T) {
	t.Run("returns logins", func(t *testing.T) {
		writeFakeGHScript(t, `printf 'alice\nbob\ncharlie\n'`)
		g := NewGH(nil)
		logins, err := g.ListCollaborators(context.Background(), "owner", "repo")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice", "bob", "charlie"}, logins)
	})

	t.Run("filters blank lines", func(t *testing.T) {
		writeFakeGHScript(t, `printf 'alice\n\nbob\n'`)
		g := NewGH(nil)
		logins, err := g.ListCollaborators(context.Background(), "owner", "repo")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice", "bob"}, logins)
	})

	t.Run("returns error on non-zero exit", func(t *testing.T) {
		writeFakeGHScript(t, `exit 1`)
		g := NewGH(nil)
		_, err := g.ListCollaborators(context.Background(), "owner", "repo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list collaborators")
	})
}

func TestGH_RegisterWebhook(t *testing.T) {
	t.Run("creates new webhook when no existing hook", func(t *testing.T) {
		writeFakeGHScript(t, `
			case "$*" in
				*--method*POST*) exit 0;;
				*) echo ""; exit 0;;
			esac
			exit 1
		`)
		g := NewGH(nil)
		err := g.RegisterWebhook(context.Background(), "owner", "repo", "https://example.com/hook", "secret")
		assert.NoError(t, err)
	})

	t.Run("updates existing webhook when hook ID found", func(t *testing.T) {
		writeFakeGHScript(t, `
			case "$*" in
				*--method*PATCH*) exit 0;;
				*) echo "42"; exit 0;;
			esac
			exit 1
		`)
		g := NewGH(nil)
		err := g.RegisterWebhook(context.Background(), "owner", "repo", "https://example.com/hook", "secret")
		assert.NoError(t, err)
	})

	t.Run("returns error when create fails", func(t *testing.T) {
		writeFakeGHScript(t, `
			case "$*" in
				*--method*POST*) exit 1;;
				*) echo ""; exit 0;;
			esac
			exit 1
		`)
		g := NewGH(nil)
		err := g.RegisterWebhook(context.Background(), "owner", "repo", "https://example.com/hook", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create webhook")
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		writeFakeGHScript(t, `
			case "$*" in
				*--method*PATCH*) exit 1;;
				*) echo "42"; exit 0;;
			esac
			exit 1
		`)
		g := NewGH(nil)
		err := g.RegisterWebhook(context.Background(), "owner", "repo", "https://example.com/hook", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update webhook")
	})
}
