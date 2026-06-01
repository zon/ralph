package github

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFakeGHWebhook(t *testing.T, hookID string) (string, string) {
	t.Helper()
	dir := t.TempDir()

	invokedFile := filepath.Join(dir, "invoked")

	script := "#!/bin/sh\n"
	script += "echo \"$@\" >> " + shQuote(invokedFile) + "\n"
	script += "case \"$*\" in\n"
	script += "  *--jq*)\n"
	script += "    if [ -n \"" + hookID + "\" ]; then\n"
	script += "      echo \"" + hookID + "\"\n"
	script += "    fi\n"
	script += "    ;;\n"
	script += "  *PATCH*)\n"
	script += "    cat /dev/stdin > /dev/null 2>&1\n"
	script += "    ;;\n"
	script += "  *POST*)\n"
	script += "    cat /dev/stdin > /dev/null 2>&1\n"
	script += "    ;;\n"
	script += "  *)\n"
	script += "    exit 1\n"
	script += "    ;;\n"
	script += "esac\n"

	ghBin := filepath.Join(dir, "gh")
	err := os.WriteFile(ghBin, []byte(script), 0755)
	require.NoError(t, err)
	return dir, invokedFile
}

func TestRegisterWebhook_Create(t *testing.T) {
	// No hookID means list returns empty → should trigger create path
	dir, invokedFile := writeFakeGHWebhook(t, "")
	withFakeGH(t, dir)

	ctx := context.Background()
	err := RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	require.NoError(t, err)

	invoked, err := os.ReadFile(invokedFile)
	require.NoError(t, err)
	assert.Contains(t, string(invoked), "POST")
}

func TestRegisterWebhook_Update(t *testing.T) {
	// hookID "42" means list returns 42 → should trigger update path
	dir, invokedFile := writeFakeGHWebhook(t, "42")
	withFakeGH(t, dir)

	ctx := context.Background()
	err := RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	require.NoError(t, err)

	invoked, err := os.ReadFile(invokedFile)
	require.NoError(t, err)
	assert.Contains(t, string(invoked), "PATCH")
	assert.Contains(t, string(invoked), "42")
}

func TestRegisterWebhook_ListErrorProceedsToCreate(t *testing.T) {
	// Use a non-existent gh binary directory to trigger list error
	dir := t.TempDir()
	// Write a gh that fails on list (exit 1) to simulate error
	script := "#!/bin/sh\n"
	script += "case \"$*\" in\n"
	script += "  *--jq*) exit 1 ;;\n"
	script += "  *POST*) cat /dev/stdin > /dev/null 2>&1 ;;\n"
	script += "  *) exit 1 ;;\n"
	script += "esac\n"
	ghBin := filepath.Join(dir, "gh")
	err := os.WriteFile(ghBin, []byte(script), 0755)
	require.NoError(t, err)
	withFakeGH(t, dir)

	ctx := context.Background()
	err = RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	require.NoError(t, err)
}

func TestRegisterWebhook_UpdateError(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	script += "case \"$*\" in\n"
	script += "  *--jq*) echo \"42\" ;;\n"
	script += "  *PATCH*) echo \"update failed\" >&2; exit 1 ;;\n"
	script += "  *) exit 1 ;;\n"
	script += "esac\n"
	ghBin := filepath.Join(dir, "gh")
	err := os.WriteFile(ghBin, []byte(script), 0755)
	require.NoError(t, err)
	withFakeGH(t, dir)

	ctx := context.Background()
	err = RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test-owner")
	assert.Contains(t, err.Error(), "test-repo")
	assert.Contains(t, err.Error(), "update failed")
}

func TestRegisterWebhook_CreateError(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	script += "case \"$*\" in\n"
	script += "  *--jq*) ;;\n"
	script += "  *POST*) echo \"create failed\" >&2; exit 1 ;;\n"
	script += "  *) exit 1 ;;\n"
	script += "esac\n"
	ghBin := filepath.Join(dir, "gh")
	err := os.WriteFile(ghBin, []byte(script), 0755)
	require.NoError(t, err)
	withFakeGH(t, dir)

	ctx := context.Background()
	err = RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test-owner")
	assert.Contains(t, err.Error(), "test-repo")
	assert.Contains(t, err.Error(), "create failed")
}


