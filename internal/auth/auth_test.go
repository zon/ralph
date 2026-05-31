package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FileNotExists_ReturnsEmptyMap(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(cwd) })

	keys, err := Load()
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestLoad_ReadsValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(cwd) })

	require.NoError(t, os.MkdirAll(".ralph", 0755))
	require.NoError(t, os.WriteFile(".ralph/auth.yaml", []byte("anthropic: sk-ant-123\ngoogle: AIza-existing\n"), 0644))

	keys, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-123", keys["anthropic"])
	assert.Equal(t, "AIza-existing", keys["google"])
}

func TestWrite_CreatesDotRalphDir(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(cwd) })

	err = Write(map[string]string{"anthropic": "sk-ant-123"})
	require.NoError(t, err)

	_, err = os.Stat(".ralph/auth.yaml")
	require.NoError(t, err)
}

func TestWrite_ThenLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(cwd) })

	input := map[string]string{
		"anthropic": "sk-ant-123",
		"google":    "AIza-existing",
	}

	require.NoError(t, Write(input))

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, input, loaded)
}
