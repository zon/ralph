package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Loader = (*Client)(nil)

func TestClientLoad(t *testing.T) {
	tmpDir := t.TempDir()

	t.Chdir(tmpDir)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	// Write config file with known, distinctive content
	configContent := `maxIterations: 5
defaultBranch: develop
services:
  - name: test-service
    command: echo
    args: [hello]
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Write custom instructions file
	instructionsContent := "Custom instructions for testing"
	instructionsPath := filepath.Join(ralphDir, "instructions.md")
	require.NoError(t, os.WriteFile(instructionsPath, []byte(instructionsContent), 0644))

	client := &Client{}
	config, err := client.Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "develop", config.DefaultBranch)
	assert.Len(t, config.Services, 1)
	assert.Equal(t, "test-service", config.Services[0].Name)
	assert.Equal(t, instructionsContent, config.Instructions)
}
