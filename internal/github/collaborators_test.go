package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFakeGH(t *testing.T, stdoutLines []string, stderr string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()

	stdoutContent := strings.Join(stdoutLines, "\n")
	stdoutFile := filepath.Join(dir, "stdout.txt")
	err := os.WriteFile(stdoutFile, []byte(stdoutContent), 0644)
	require.NoError(t, err)

	var script string
	script += "#!/bin/sh\n"
	script += fmt.Sprintf("cat %s\n", shQuote(stdoutFile))
	if stderr != "" {
		script += fmt.Sprintf("echo %s >&2\n", shQuote(stderr))
	}
	script += fmt.Sprintf("exit %d\n", exitCode)

	ghBin := filepath.Join(dir, "gh")
	err = os.WriteFile(ghBin, []byte(script), 0755)
	require.NoError(t, err)
	return dir
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func withFakeGH(t *testing.T, dir string) {
	t.Helper()
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)
}

func TestListCollaborators_Multiple(t *testing.T) {
	dir := writeFakeGH(t, []string{"alice", "bob", "charlie"}, "", 0)
	withFakeGH(t, dir)

	ctx := context.Background()
	logins, err := ListCollaborators(ctx, "test-owner", "test-repo")
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob", "charlie"}, logins)
}

func TestListCollaborators_Single(t *testing.T) {
	dir := writeFakeGH(t, []string{"alice"}, "", 0)
	withFakeGH(t, dir)

	ctx := context.Background()
	logins, err := ListCollaborators(ctx, "test-owner", "test-repo")
	require.NoError(t, err)
	assert.Equal(t, []string{"alice"}, logins)
}

func TestListCollaborators_Empty(t *testing.T) {
	dir := writeFakeGH(t, nil, "", 0)
	withFakeGH(t, dir)

	ctx := context.Background()
	logins, err := ListCollaborators(ctx, "test-owner", "test-repo")
	require.NoError(t, err)
	assert.Empty(t, logins)
}

func TestListCollaborators_Error(t *testing.T) {
	dir := writeFakeGH(t, nil, "not found", 1)
	withFakeGH(t, dir)

	ctx := context.Background()
	_, err := ListCollaborators(ctx, "test-owner", "test-repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test-owner")
	assert.Contains(t, err.Error(), "test-repo")
	assert.Contains(t, err.Error(), "not found")
}
