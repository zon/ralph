package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FileNotExists_ReturnsEmptyMap(t *testing.T) {
	tmpDir := t.TempDir()

	keys, err := Load(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestLoad_ReadsValidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.MkdirAll(tmpDir+"/.ralph", 0755))
	require.NoError(t, os.WriteFile(tmpDir+"/.ralph/auth.yaml", []byte("anthropic: sk-ant-123\ngoogle: AIza-existing\n"), 0644))

	keys, err := Load(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-123", keys["anthropic"])
	assert.Equal(t, "AIza-existing", keys["google"])
}

func TestWrite_CreatesDotRalphDir(t *testing.T) {
	tmpDir := t.TempDir()

	err := Write(tmpDir, map[string]string{"anthropic": "sk-ant-123"})
	require.NoError(t, err)

	_, err = os.Stat(tmpDir + "/.ralph/auth.yaml")
	require.NoError(t, err)
}

func TestWrite_ThenLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	input := map[string]string{
		"anthropic": "sk-ant-123",
		"google":    "AIza-existing",
	}

	require.NoError(t, Write(tmpDir, input))

	loaded, err := Load(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, input, loaded)
}

func TestWrite_PreservesKeysFromPriorWrite(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, Write(tmpDir, map[string]string{"google": "AIza-existing"}))

	keys, err := Load(tmpDir)
	require.NoError(t, err)
	keys["anthropic"] = "sk-ant-123"
	require.NoError(t, Write(tmpDir, keys))

	loaded, err := Load(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-123", loaded["anthropic"])
	assert.Equal(t, "AIza-existing", loaded["google"])
}
