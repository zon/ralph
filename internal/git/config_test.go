package git

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "redacts installation token in insteadOf key",
			input: "url.https://x-access-token:ghs_secret@github.com/.insteadOf",
			want:  "url.https://redacted:redacted@github.com/.insteadOf",
		},
		{
			name:  "redacts credentials in a remote URL",
			input: "https://user:pass@example.com/owner/repo.git",
			want:  "https://redacted:redacted@example.com/owner/repo.git",
		},
		{
			name:  "leaves URLs without credentials unchanged",
			input: "https://github.com/owner/repo.git",
			want:  "https://github.com/owner/repo.git",
		},
		{
			name:  "leaves scp style remotes unchanged",
			input: "git@github.com:owner/repo.git",
			want:  "git@github.com:owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RedactCredentials(tt.input))
		})
	}
}

func TestRedactedError(t *testing.T) {
	cause := errors.New("git [config --global url.https://x-access-token:ghs_secret@github.com/.insteadOf https://github.com/] failed")
	err := redactedError{err: cause}

	assert.NotContains(t, err.Error(), "ghs_secret")
	assert.Contains(t, err.Error(), "https://redacted:redacted@github.com/.insteadOf")
	assert.ErrorIs(t, err, cause)
}

func TestConfig(t *testing.T) {
	t.Run("sets and retrieves local config", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := Config(false, "user.testkey", "testvalue")
		require.NoError(t, err)

		output, err := ConfigList(false)
		require.NoError(t, err)
		assert.Contains(t, output, "user.testkey=testvalue")
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := Config(false, "user.overwritekey", "first")
		require.NoError(t, err)

		err = Config(false, "user.overwritekey", "second")
		require.NoError(t, err)

		output, err := ConfigList(false)
		require.NoError(t, err)
		assert.Contains(t, output, "user.overwritekey=second")
	})
}

func TestConfigList(t *testing.T) {
	t.Run("lists global config", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(tempDir, ".gitconfig"))

		err := Config(true, "user.isolatedtest", "isolatedvalue")
		require.NoError(t, err)

		output, err := ConfigList(true)
		require.NoError(t, err)
		assert.Contains(t, output, "user.isolatedtest=isolatedvalue")
	})

	t.Run("lists local config", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := Config(false, "user.listtest", "listvalue")
		require.NoError(t, err)

		output, err := ConfigList(false)
		require.NoError(t, err)
		assert.Contains(t, output, "user.listtest=listvalue")
	})
}

func TestConfigUnset(t *testing.T) {
	t.Run("unsets existing config", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := Config(false, "user.unsettest", "unsetvalue")
		require.NoError(t, err)

		output, err := ConfigList(false)
		require.NoError(t, err)
		assert.Contains(t, output, "user.unsettest=unsetvalue")

		err = ConfigUnset(false, "user.unsettest")
		require.NoError(t, err)

		output, err = ConfigList(false)
		require.NoError(t, err)
		assert.NotContains(t, output, "user.unsettest=unsetvalue")
	})

}
