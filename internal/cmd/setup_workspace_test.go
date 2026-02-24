package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/config"
)

func TestSetupWorkspaceCmd_LinkField(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}

	// Create source files that would be mounted
	configMapDir := filepath.Join(workspaceDir, "configs")
	if err := os.MkdirAll(configMapDir, 0755); err != nil {
		t.Fatalf("Failed to create configMap directory: %v", err)
	}
	configMapFile := filepath.Join(configMapDir, "app-config.yaml")
	if err := os.WriteFile(configMapFile, []byte("config: value"), 0644); err != nil {
		t.Fatalf("Failed to create configMap file: %v", err)
	}

	secretDir := filepath.Join(workspaceDir, "secrets")
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		t.Fatalf("Failed to create secret directory: %v", err)
	}
	secretFile := filepath.Join(secretDir, "api-key.txt")
	if err := os.WriteFile(secretFile, []byte("secret-key"), 0644); err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}

	// Create .ralph directory and config.yaml
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

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
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Change to test directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Run setup-workspace command
	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir}
	if err := cmd.Run(); err != nil {
		t.Fatalf("SetupWorkspaceCmd.Run failed: %v", err)
	}

	// Check that symlinks were created only for entries with link: true
	expectedSymlinks := []string{
		filepath.Join(tmpDir, "configs"),
		filepath.Join(tmpDir, "secrets"),
	}
	unexpectedSymlinks := []string{
		filepath.Join(tmpDir, "other-config.yaml"),
		filepath.Join(tmpDir, "other-secret.txt"),
	}

	for _, symlink := range expectedSymlinks {
		if _, err := os.Lstat(symlink); err != nil {
			t.Errorf("Expected symlink %s to exist, got error: %v", symlink, err)
		}
	}

	for _, symlink := range unexpectedSymlinks {
		if _, err := os.Lstat(symlink); err == nil {
			t.Errorf("Expected symlink %s to not exist, but it does", symlink)
		}
	}

	// Verify the symlinks point to the correct locations
	configsLink, err := os.Readlink(filepath.Join(tmpDir, "configs"))
	if err != nil {
		t.Errorf("Failed to read configs symlink: %v", err)
	}
	if configsLink != filepath.Join(workspaceDir, "configs") {
		t.Errorf("configs symlink points to %s, expected %s", configsLink, filepath.Join(workspaceDir, "configs"))
	}

	secretsLink, err := os.Readlink(filepath.Join(tmpDir, "secrets"))
	if err != nil {
		t.Errorf("Failed to read secrets symlink: %v", err)
	}
	if secretsLink != filepath.Join(workspaceDir, "secrets") {
		t.Errorf("secrets symlink points to %s, expected %s", secretsLink, filepath.Join(workspaceDir, "secrets"))
	}
}

func TestSetupWorkspaceCmd_NoDestNoLink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ralph directory and config.yaml with configMaps/secrets without destFile/destDir
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	configContent := `workflow:
  configMaps:
    - name: my-config
  secrets:
    - name: my-secret
`
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Change to test directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Run setup-workspace command - should not error even though there's nothing to link
	cmd := &SetupWorkspaceCmd{WorkspaceDir: "/workspace"}
	if err := cmd.Run(); err != nil {
		t.Fatalf("SetupWorkspaceCmd.Run failed: %v", err)
	}
}

func TestSetupWorkspaceCmd_LinkMethod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace directory structure
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}

	// Create source file in workspace
	srcFile := filepath.Join(workspaceDir, "source.txt")
	if err := os.WriteFile(srcFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination directory (current working directory)
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	cmd := &SetupWorkspaceCmd{WorkspaceDir: workspaceDir}

	// Test with destFile
	if err := cmd.link(destDir, "source.txt", ""); err != nil {
		t.Fatalf("link with destFile failed: %v", err)
	}

	linkPath := filepath.Join(destDir, "source.txt")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Errorf("Symlink not created: %v", err)
	}

	// Verify symlink points to correct location
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Errorf("Failed to read symlink: %v", err)
	}
	if target != srcFile {
		t.Errorf("Symlink points to %s, expected %s", target, srcFile)
	}

	// Clean up for next test
	os.Remove(linkPath)

	// Test with destDir (creates symlink named after the source file)
	if err := cmd.link(destDir, "", "source.txt"); err != nil {
		t.Fatalf("link with destDir failed: %v", err)
	}

	linkPath = filepath.Join(destDir, "source.txt")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Errorf("Symlink not created: %v", err)
	}

	// Test with neither destFile nor destDir (should do nothing)
	if err := cmd.link(destDir, "", ""); err != nil {
		t.Fatalf("link with no destination failed: %v", err)
	}

	// Test with non-existent source (should error)
	if err := cmd.link(destDir, "nonexistent.txt", ""); err == nil {
		t.Error("Expected error for non-existent source, got nil")
	}
}

func TestConfigMapMountAndSecretMountYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ralph directory and config.yaml
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	// Test YAML marshaling/unmarshaling with link field
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
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Change to test directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configMaps
	if len(cfg.Workflow.ConfigMaps) != 3 {
		t.Fatalf("Expected 3 configMaps, got %d", len(cfg.Workflow.ConfigMaps))
	}

	expectedConfigMaps := []struct {
		name     string
		destFile string
		destDir  string
		link     bool
	}{
		{"config1", "config.yaml", "", true},
		{"config2", "", "configs", false}, // link omitted, should default to false
		{"config3", "other.yaml", "", false},
	}

	for i, expected := range expectedConfigMaps {
		cm := cfg.Workflow.ConfigMaps[i]
		if cm.Name != expected.name {
			t.Errorf("ConfigMap[%d].Name = %s, want %s", i, cm.Name, expected.name)
		}
		if cm.DestFile != expected.destFile {
			t.Errorf("ConfigMap[%d].DestFile = %s, want %s", i, cm.DestFile, expected.destFile)
		}
		if cm.DestDir != expected.destDir {
			t.Errorf("ConfigMap[%d].DestDir = %s, want %s", i, cm.DestDir, expected.destDir)
		}
		if cm.Link != expected.link {
			t.Errorf("ConfigMap[%d].Link = %v, want %v", i, cm.Link, expected.link)
		}
	}

	// Verify secrets
	if len(cfg.Workflow.Secrets) != 2 {
		t.Fatalf("Expected 2 secrets, got %d", len(cfg.Workflow.Secrets))
	}

	expectedSecrets := []struct {
		name     string
		destFile string
		destDir  string
		link     bool
	}{
		{"secret1", "secret.txt", "", true},
		{"secret2", "", "secrets", false}, // link omitted, should default to false
	}

	for i, expected := range expectedSecrets {
		secret := cfg.Workflow.Secrets[i]
		if secret.Name != expected.name {
			t.Errorf("Secret[%d].Name = %s, want %s", i, secret.Name, expected.name)
		}
		if secret.DestFile != expected.destFile {
			t.Errorf("Secret[%d].DestFile = %s, want %s", i, secret.DestFile, expected.destFile)
		}
		if secret.DestDir != expected.destDir {
			t.Errorf("Secret[%d].DestDir = %s, want %s", i, secret.DestDir, expected.destDir)
		}
		if secret.Link != expected.link {
			t.Errorf("Secret[%d].Link = %v, want %v", i, secret.Link, expected.link)
		}
	}
}
