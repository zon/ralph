package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/output"
)

func TestSetupWorkspaceCmd_LinkField(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err, "Failed to create workspace directory")

	configMapDir := filepath.Join(workspaceDir, "configs")
	err = os.MkdirAll(configMapDir, 0755)
	require.NoError(t, err, "Failed to create configMap directory")
	configMapFile := filepath.Join(configMapDir, "app-config.yaml")
	err = os.WriteFile(configMapFile, []byte("config: value"), 0644)
	require.NoError(t, err, "Failed to create configMap file")

	secretDir := filepath.Join(workspaceDir, "secrets")
	err = os.MkdirAll(secretDir, 0755)
	require.NoError(t, err, "Failed to create secret directory")
	secretFile := filepath.Join(secretDir, "api-key.txt")
	err = os.WriteFile(secretFile, []byte("secret-key"), 0644)
	require.NoError(t, err, "Failed to create secret file")

	ralphDir := filepath.Join(tmpDir, ".ralph")
	err = os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err, "Failed to create .ralph directory")

	configContent := `workflow:
  configMaps:
    - name: my-config
      destDir: configs
      link: true
    - name: other-config
      destFile: other-config.yaml
      link: false
  secrets:
    - name: my-secret
      destDir: secrets
      link: true
    - name: other-secret
      destFile: other-secret.txt
`
	err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err, "Failed to create config file")

	t.Chdir(tmpDir)

	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir, out: output.NewClient(os.Stdout, os.Stderr, false)}
	require.NotNil(t, cmd)

	err = cmd.Run()
	require.NoError(t, err, "SetupWorkspaceCmd.Run should not fail")

	// Verify symlinks were created
	configLink := filepath.Join(tmpDir, "configs", "app-config.yaml")
	_, err = os.Lstat(configLink)
	assert.NoError(t, err, "Config symlink should exist")

	secretLink := filepath.Join(tmpDir, "secrets", "api-key.txt")
	_, err = os.Lstat(secretLink)
	assert.NoError(t, err, "Secret symlink should exist")

	// Verify non-linked entries don't create symlinks
	_, err = os.Lstat(filepath.Join(tmpDir, "other-config.yaml"))
	assert.True(t, os.IsNotExist(err), "Non-linked config should not create symlink")

	_, err = os.Lstat(filepath.Join(tmpDir, "other-secret.txt"))
	assert.True(t, os.IsNotExist(err), "Non-linked secret should not create symlink")
}

func TestSetupWorkspaceCmd_SkipOnExistingLink(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	srcFile := filepath.Join(workspaceDir, "source.txt")
	err = os.WriteFile(srcFile, []byte("content"), 0644)
	require.NoError(t, err)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	err = os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err)

	configContent := `workflow:
  configMaps:
    - name: my-config
      destFile: source.txt
      link: true
`
	err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Create the target symlink already
	err = os.Symlink(srcFile, filepath.Join(tmpDir, "source.txt"))
	require.NoError(t, err)

	t.Chdir(tmpDir)

	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir, out: output.NewClient(os.Stdout, os.Stderr, false)}
	err = cmd.Run()
	require.NoError(t, err, "Should succeed even if link already exists")
}

func TestSetupWorkspaceCmd_StatSourceFailure(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	err = os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err)

	configContent := `workflow:
  configMaps:
    - name: my-config
      destFile: nonexistent.txt
      link: true
`
	err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	t.Chdir(tmpDir)

	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir, out: output.NewClient(os.Stdout, os.Stderr, false)}
	err = cmd.Run()
	require.Error(t, err, "Should fail when source does not exist")
}

func TestConfigMapMountAndSecretMountYAML(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	err := os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err, "Failed to create .ralph directory")

	yamlContent := `workflow:
  configMaps:
    - name: config1
      destFile: config.yaml
      link: true
    - name: config2
      destDir: configs
    - name: config3
      destFile: other.yaml
      link: false
  secrets:
    - name: secret1
      destFile: secret.txt
      link: true
    - name: secret2
      destDir: secrets
`
	err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(yamlContent), 0644)
	require.NoError(t, err, "Failed to create config file")

	t.Chdir(tmpDir)

	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	assert.Len(t, cfg.Workflow.ConfigMaps, 3, "Should have 3 configMaps")

	expectedConfigMaps := []struct {
		name     string
		destFile string
		destDir  string
		link     bool
	}{
		{"config1", "config.yaml", "", true},
		{"config2", "", "configs", false},
		{"config3", "other.yaml", "", false},
	}

	for i, expected := range expectedConfigMaps {
		cm := cfg.Workflow.ConfigMaps[i]
		assert.Equal(t, expected.name, cm.Name, "ConfigMap[%d].Name should match", i)
		assert.Equal(t, expected.destFile, cm.DestFile, "ConfigMap[%d].DestFile should match", i)
		assert.Equal(t, expected.destDir, cm.DestDir, "ConfigMap[%d].DestDir should match", i)
		assert.Equal(t, expected.link, cm.Link, "ConfigMap[%d].Link should match", i)
	}

	assert.Len(t, cfg.Workflow.Secrets, 2, "Should have 2 secrets")

	expectedSecrets := []struct {
		name     string
		destFile string
		destDir  string
		link     bool
	}{
		{"secret1", "secret.txt", "", true},
		{"secret2", "", "secrets", false},
	}

	for i, expected := range expectedSecrets {
		secret := cfg.Workflow.Secrets[i]
		assert.Equal(t, expected.name, secret.Name, "Secret[%d].Name should match", i)
		assert.Equal(t, expected.destFile, secret.DestFile, "Secret[%d].DestFile should match", i)
		assert.Equal(t, expected.destDir, secret.DestDir, "Secret[%d].DestDir should match", i)
		assert.Equal(t, expected.link, secret.Link, "Secret[%d].Link should match", i)
	}
}
