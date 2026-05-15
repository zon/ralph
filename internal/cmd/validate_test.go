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
				content := `slug: test-project
title: A test project
requirements:
  - slug: validate-subcommand
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
			name: "missing slug field",
			setupFile: func(tmpDir string) string {
				content := `title: A test project
requirements:
  - slug: validate-subcommand
    description: New validate subcommand
    items:
      - Test item
    passing: false`
				filePath := filepath.Join(tmpDir, "no-slug.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "project slug is required",
		},
		{
			name: "missing requirements",
			setupFile: func(tmpDir string) string {
				content := `slug: no-reqs-project
title: A test project`
				filePath := filepath.Join(tmpDir, "no-requirements.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "at least one requirement",
		},
		{
			name: "requirement missing slug",
			setupFile: func(tmpDir string) string {
				content := `slug: missing-req-slug
title: Missing requirement slug
requirements:
  - description: New validate subcommand
    items:
      - Test item
    passing: false`
				filePath := filepath.Join(tmpDir, "missing-req-slug.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "slug is required",
		},
		{
			name: "requirement with no items/scenarios/code/tests",
			setupFile: func(tmpDir string) string {
				content := `slug: empty-requirement
title: Requirement with no work defined
requirements:
  - slug: empty-req
    description: nothing to do
    passing: false`
				filePath := filepath.Join(tmpDir, "empty-requirement.yaml")
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
				return filePath
			},
			wantErr:     true,
			errContains: "at least one of items, scenarios, code, or tests",
		},
		{
			name: "invalid YAML syntax",
			setupFile: func(tmpDir string) string {
				content := `slug: invalid-yaml
  this: is: not: valid: yaml
requirements:
  - slug: anything`
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
	content := `slug: output-test
title: Test output
requirements:
  - slug: feature-1
    description: Feature 1
    items:
      - Item 1
    passing: false
  - slug: feature-2
    description: Feature 2
    items:
      - Item 2
    passing: true`
	filePath := filepath.Join(tmpDir, "output-test.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cmd := &ValidateCmd{
		ProjectFile: filePath,
	}

	err := cmd.Run()
	require.NoError(t, err)
}
