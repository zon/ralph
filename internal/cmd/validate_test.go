package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCmd(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(tmpDir string) string
		wantErr        bool
		errContains    string
		outputContains string
	}{
		{
			name: "valid project file",
			setupFile: func(tmpDir string) string {
				content := `name: test-project
description: A test project
requirements:
  - category: cli
    description: New validate subcommand
    items:
      - Test item
    passing: false`
				filePath := filepath.Join(tmpDir, "valid-project.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:        false,
			outputContains: "test-project",
		},
		{
			name: "missing name field",
			setupFile: func(tmpDir string) string {
				content := `description: A test project
requirements:
  - category: cli
    description: New validate subcommand
    items:
      - Test item
    passing: false`
				filePath := filepath.Join(tmpDir, "no-name.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "project name is required",
		},
		{
			name: "missing requirements",
			setupFile: func(tmpDir string) string {
				content := `name: no-reqs-project
description: A test project`
				filePath := filepath.Join(tmpDir, "no-requirements.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "at least one requirement",
		},
		{
			name: "invalid YAML syntax",
			setupFile: func(tmpDir string) string {
				content := `name: invalid-yaml
  this: is: not: valid: yaml
requirements:
  - category: cli`
				filePath := filepath.Join(tmpDir, "invalid-yaml.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "failed to parse project YAML",
		},
		{
			name: "file not found",
			setupFile: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.yaml")
			},
			wantErr:     true,
			errContains: "failed to read project file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			projectFile := tt.setupFile(tmpDir)

			cmd := &ValidateCmd{
				ProjectFile: projectFile,
			}

			err := cmd.Run()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCmdOutput(t *testing.T) {
	tmpDir := t.TempDir()
	content := `name: output-test
description: Test output
requirements:
  - category: cli
    description: Feature 1
    passing: false
  - category: cli
    description: Feature 2
    passing: true`
	filePath := filepath.Join(tmpDir, "output-test.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cmd := &ValidateCmd{
		ProjectFile: filePath,
	}

	err := cmd.Run()
	require.NoError(t, err)
}
