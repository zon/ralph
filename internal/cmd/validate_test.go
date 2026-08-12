package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
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

// TestValidateCmdItemsFlagTakesPrecedence covers the "`--items` flag takes
// precedence" scenario: with `items: .requirements` in the config, the flag
// `.spec.tasks` wins, so the file validates against `.spec.tasks`.
func TestValidateCmdItemsFlagTakesPrecedence(t *testing.T) {
	// GIVEN `items: .requirements` is set in `.ralph/config.yaml`
	tmpDir := writeValidateConfig(t, "items: .requirements\n")
	// AND the file has no `.requirements`, so only the flag query resolves
	filePath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("spec:\n  tasks:\n    - slug: one\n    - slug: two\n"), 0644))

	// AND the user runs `ralph validate <file> --items '.spec.tasks'`
	cmd := &ValidateCmd{ProjectFile: filePath, Items: ".spec.tasks"}

	// WHEN the item query is resolved
	err := cmd.Run()

	// THEN `.spec.tasks` is evaluated against the parsed file
	require.NoError(t, err)
}

// TestValidateCmdConfigQueryUsedWhenNoFlag covers the "Config query used when no
// flag is passed" scenario: the config `items` field resolves the query, so a
// file that only yields items under the config query validates and one that
// does not fails naming the config query.
func TestValidateCmdConfigQueryUsedWhenNoFlag(t *testing.T) {
	// GIVEN `items: .requirements` is set in `.ralph/config.yaml`
	tmpDir := writeValidateConfig(t, "items: .requirements\n")
	// AND the file has no `.requirements`, so the config query yields nothing
	filePath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("slug: x\n"), 0644))

	// AND no `--items` flag is passed
	cmd := &ValidateCmd{ProjectFile: filePath}

	// WHEN the item query is resolved
	err := cmd.Run()

	// THEN `.requirements` is evaluated against the parsed file, yielding no items
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: .requirements", err.Error())
}

// TestValidateCmdDefaultQueryWhenUnset covers the "Default query when flag and
// config are unset" scenario: with no `items` in the config and no flag, the
// query `.` is evaluated, so a file whose top level is an array validates.
func TestValidateCmdDefaultQueryWhenUnset(t *testing.T) {
	// GIVEN `items` is not set in `.ralph/config.yaml`
	tmpDir := writeValidateConfig(t, "model: deepseek/deepseek-chat\n")
	// AND a file whose top level is an array
	filePath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("- one\n- two\n- three\n"), 0644))

	// AND no `--items` flag is passed
	cmd := &ValidateCmd{ProjectFile: filePath}

	// WHEN the item query is resolved
	err := cmd.Run()

	// THEN the query `.` is evaluated, so the file validates with no configuration
	require.NoError(t, err)
}

// TestValidateCmdItemsFlagParsing covers the item that `ralph validate` accepts
// an `--items` flag holding the jq query that selects the item array.
func TestValidateCmdItemsFlagParsing(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		expectFlag string
	}{
		{
			name:       "items flag is parsed",
			args:       []string{"validate", "test.yaml", "--items", ".spec.tasks"},
			expectFlag: ".spec.tasks",
		},
		{
			name:       "items flag defaults to empty",
			args:       []string{"validate", "test.yaml"},
			expectFlag: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.expectFlag, cmd.Validate.Items)
		})
	}
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
