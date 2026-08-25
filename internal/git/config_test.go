package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
