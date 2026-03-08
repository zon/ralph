package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
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

	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir}
	err = cmd.Run()
	require.NoError(t, err, "SetupWorkspaceCmd.Run should not fail")

	expectedSymlinks := []string{
		filepath.Join(tmpDir, "configs"),
		filepath.Join(tmpDir, "secrets"),
	}
	unexpectedSymlinks := []string{
		filepath.Join(tmpDir, "other-config.yaml"),
		filepath.Join(tmpDir, "other-secret.txt"),
	}

	for _, symlink := range expectedSymlinks {
		_, err := os.Lstat(symlink)
		assert.NoError(t, err, "Expected symlink %s to exist", symlink)
	}

	for _, symlink := range unexpectedSymlinks {
		_, err := os.Lstat(symlink)
		assert.Error(t, err, "Expected symlink %s to not exist", symlink)
	}

	configsLink, err := os.Readlink(filepath.Join(tmpDir, "configs"))
	require.NoError(t, err, "Failed to read configs symlink")
	assert.Equal(t, filepath.Join(workspaceDir, "configs"), configsLink, "configs symlink should point to correct location")

	secretsLink, err := os.Readlink(filepath.Join(tmpDir, "secrets"))
	require.NoError(t, err, "Failed to read secrets symlink")
	assert.Equal(t, filepath.Join(workspaceDir, "secrets"), secretsLink, "secrets symlink should point to correct location")
}

func TestSetupWorkspaceCmd_NoDestNoLink(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	err := os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err, "Failed to create .ralph directory")

	configContent := `workflow:
  configMaps:
    - name: my-config
  secrets:
    - name: my-secret
`
	err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err, "Failed to create config file")

	t.Chdir(tmpDir)

	cmd := &SetupWorkspaceCmd{WorkspaceDir: "/workspace"}
	err = cmd.Run()
	require.NoError(t, err, "SetupWorkspaceCmd.Run should not fail")
}

func TestSetupWorkspaceCmd_LinkMethod(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(workspaceDir, 0755)
	require.NoError(t, err, "Failed to create workspace directory")

	srcFile := filepath.Join(workspaceDir, "source.txt")
	err = os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create source file")

	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err, "Failed to create destination directory")

	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir}

	err = cmd.link(destDir, "source.txt", "")
	require.NoError(t, err, "link with destFile should not fail")

	linkPath := filepath.Join(destDir, "source.txt")
	_, err = os.Lstat(linkPath)
	assert.NoError(t, err, "Symlink should be created")

	target, err := os.Readlink(linkPath)
	require.NoError(t, err, "Failed to read symlink")
	assert.Equal(t, srcFile, target, "Symlink should point to correct location")

	os.Remove(linkPath)

	err = cmd.link(destDir, "", "source.txt")
	require.NoError(t, err, "link with destDir should not fail")

	linkPath = filepath.Join(destDir, "source.txt")
	_, err = os.Lstat(linkPath)
	assert.NoError(t, err, "Symlink should be created")

	err = cmd.link(destDir, "", "")
	require.NoError(t, err, "link with no destination should not fail")

	err = cmd.link(destDir, "nonexistent.txt", "")
	assert.Error(t, err, "Should return error for non-existent source")
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
