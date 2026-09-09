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

func TestClientLoopSteps_MatchingSlugReturnsSteps(t *testing.T) {
	loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
`)

	client := &Client{}
	steps, err := client.LoopSteps("fmt")
	require.NoError(t, err)
	assert.Equal(t, []string{"run gofmt", "run go vet"}, steps)
}

func TestClientLoopSteps_NotFoundReturnsError(t *testing.T) {
	loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	client := &Client{}
	_, err := client.LoopSteps("missing")
	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: missing")
}

func TestClientLoad_NoRALPHDirectoryAppliesDefaults(t *testing.T) {
	// GIVEN no .ralph directory in the working directory
	t.Chdir(t.TempDir())

	config, err := (&Client{}).Load()
	require.NoError(t, err, "Load() must succeed without a .ralph directory")
	require.NotNil(t, config)

	assert.Equal(t, "main", config.DefaultBranch)
	assert.Equal(t, "local", config.Mode)
	assert.Equal(t, ".", config.Items)
}

func TestClientLoopSteps_MissingSlugWithoutConfig(t *testing.T) {
	// GIVEN no .ralph directory in the working directory, so loading falls back
	// to a config of defaults with no loop entries
	t.Chdir(t.TempDir())

	client := &Client{}
	_, err := client.LoopSteps("fmt")
	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: fmt")
}
