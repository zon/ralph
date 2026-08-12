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

func TestValidateCmdWiring(t *testing.T) {
	tmpDir := t.TempDir()
	content := `slug: wiring-test
title: Wiring Test
requirements:
  - slug: wiring-req
    description: Test requirement
    items:
      - Test item
    passing: true`
	filePath := filepath.Join(tmpDir, "wiring-test.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cmd := &ValidateCmd{
		ProjectFile: filePath,
	}

	err := cmd.Run()
	require.NoError(t, err)
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

// writeValidateConfig writes a .ralph/config.yaml into a temp working directory
// and changes the working directory to it, so the validate command resolves the
// item query from the config.
func writeValidateConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(content), 0644))
	t.Chdir(tmpDir)
	return tmpDir
}

func TestValidateCmdQueryYieldsNoItems(t *testing.T) {
	// GIVEN a config whose item query resolves to nothing for the parsed file
	tmpDir := writeValidateConfig(t, "items: .missing\n")
	filePath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("slug: x\n"), 0644))

	// WHEN `ralph validate <file>` is run
	cmd := &ValidateCmd{ProjectFile: filePath}
	err := cmd.Run()

	// THEN the command exits with an error naming the query
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: .missing", err.Error())
}

func TestValidateCmdQueryEvaluationError(t *testing.T) {
	// GIVEN a config whose item query cannot be evaluated against the file
	tmpDir := writeValidateConfig(t, "items: .slug.name\n")
	filePath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("slug: x\n"), 0644))

	// WHEN `ralph validate <file>` is run
	cmd := &ValidateCmd{ProjectFile: filePath}
	err := cmd.Run()

	// THEN the command exits with an error reporting the query error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item query failed")
	assert.Contains(t, err.Error(), ".slug.name")
}
