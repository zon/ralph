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

		value, err := ConfigGet("user.testkey")
		require.NoError(t, err)
		assert.Equal(t, "testvalue", value)
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := Config(false, "user.overwritekey", "first")
		require.NoError(t, err)

		err = Config(false, "user.overwritekey", "second")
		require.NoError(t, err)

		value, err := ConfigGet("user.overwritekey")
		require.NoError(t, err)
		assert.Equal(t, "second", value)
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

		value, err := ConfigGet("user.unsettest")
		require.NoError(t, err)
		assert.Equal(t, "unsetvalue", value)

		err = ConfigUnset(false, "user.unsettest")
		require.NoError(t, err)

		_, err = ConfigGet("user.unsettest")
		assert.Error(t, err)
	})

}

func TestConfigGet(t *testing.T) {
	t.Run("retrieves existing config", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		err := Config(false, "user.gettest", "getvalue")
		require.NoError(t, err)

		value, err := ConfigGet("user.gettest")
		require.NoError(t, err)
		assert.Equal(t, "getvalue", value)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		_, err := ConfigGet("user.nonexistent")
		assert.Error(t, err)
	})
}
