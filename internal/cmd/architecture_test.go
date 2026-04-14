package cmd

import (
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validArchitectureYAML = `apps:
  - name: test-app
    description: A test application
    main:
      file: cmd/test-app/main.go
      function: main
    features:
      - name: test-feature
        description: A test feature
        functions:
          - file: internal/test/feature.go
            name: TestFunction
modules:
  - path: internal/test
    description: Test module
    type: domain
`

func TestArchitectureCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedOut   string
		expectedVerb  bool
		expectedModel string
		wantParseErr  bool
	}{
		{
			name:          "review architecture default values",
			args:          []string{"review", "architecture"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  false,
			expectedModel: "",
		},
		{
			name:          "review architecture custom output path",
			args:          []string{"review", "architecture", "--output", "custom.yaml"},
			expectedOut:   "custom.yaml",
			expectedVerb:  false,
			expectedModel: "",
		},
		{
			name:          "review architecture verbose flag",
			args:          []string{"review", "architecture", "--verbose"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  true,
			expectedModel: "",
		},
		{
			name:          "review architecture model override",
			args:          []string{"review", "architecture", "--model", "deepseek/deepseek-chat"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  false,
			expectedModel: "deepseek/deepseek-chat",
		},
		{
			name:          "review architecture all flags",
			args:          []string{"review", "architecture", "--output", "out.yaml", "--verbose", "--model", "gpt-4"},
			expectedOut:   "out.yaml",
			expectedVerb:  true,
			expectedModel: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			_, err = parser.Parse(tt.args)
			if tt.wantParseErr {
				if err == nil {
					t.Error("expected parse error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Review.Architecture.Output != tt.expectedOut {
				t.Errorf("expected Output=%q, got %q", tt.expectedOut, cmd.Review.Architecture.Output)
			}
			if cmd.Review.Architecture.Verbose != tt.expectedVerb {
				t.Errorf("expected Verbose=%v, got %v", tt.expectedVerb, cmd.Review.Architecture.Verbose)
			}
			if cmd.Review.Architecture.Model != tt.expectedModel {
				t.Errorf("expected Model=%q, got %q", tt.expectedModel, cmd.Review.Architecture.Model)
			}
		})
	}
}

func TestArchitectureCmd_Run(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	require.NoError(t, os.WriteFile(outputPath, []byte(validArchitectureYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should not error with mock AI")
}

func TestArchitectureCmd_Run_Verbose(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	require.NoError(t, os.WriteFile(outputPath, []byte(validArchitectureYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: true,
		Model:   "",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should not error with verbose and mock AI")
}

func TestArchitectureCmd_Run_ModelOverride(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	require.NoError(t, os.WriteFile(outputPath, []byte(validArchitectureYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "gpt-4",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should not error with model override and mock AI")
}

func TestArchitectureCmd_Run_PromptBuildError(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	r := &ArchitectureCmd{
		Output:  "/invalid/path/that/will/fail/architecture.yaml",
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.Error(t, err, "ArchitectureCmd.Run should error when validation fails for invalid path")
	assert.Contains(t, err.Error(), "failed to load architecture file after 3 attempts")
}

func TestArchitectureCmd_ValidationLoop_FailsAfterMaxAttempts(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	invalidYAML := `apps:
  - name: ""
    description: ""
    main:
      file: ""
      function: ""
`
	require.NoError(t, os.WriteFile(outputPath, []byte(invalidYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.Error(t, err, "ArchitectureCmd.Run should error when validation fails after max attempts")
	assert.Contains(t, err.Error(), "architecture validation failed after 3 attempts")
}

func TestArchitectureCmd_ValidationLoop_SuccessOnValidFile(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	require.NoError(t, os.WriteFile(outputPath, []byte(validArchitectureYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should succeed with valid architecture file")
}

func TestArchitectureCmd_CommitAndPR_SkipsWhenNotWorkflowExecution(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	require.NoError(t, os.WriteFile(outputPath, []byte(validArchitectureYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should succeed when not in workflow execution")
}

func TestArchitectureCmd_CommitAndPR_SkipsWhenFileNotModified(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "true")
	t.Setenv("GITHUB_REPO_OWNER", "test-owner")
	t.Setenv("GITHUB_REPO_NAME", "test-repo")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := ".ralph"
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
`
	configPath := ralphDir + "/config.yaml"
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	outputPath := tmpDir + "/test-architecture.yaml"
	require.NoError(t, os.WriteFile(outputPath, []byte(validArchitectureYAML), 0644))

	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should succeed when file is not modified")
}
