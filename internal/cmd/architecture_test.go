package cmd

import (
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name:          "default values",
			args:          []string{"architecture"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  false,
			expectedModel: "",
		},
		{
			name:          "custom output path",
			args:          []string{"architecture", "--output", "custom.yaml"},
			expectedOut:   "custom.yaml",
			expectedVerb:  false,
			expectedModel: "",
		},
		{
			name:          "verbose flag",
			args:          []string{"architecture", "--verbose"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  true,
			expectedModel: "",
		},
		{
			name:          "model override",
			args:          []string{"architecture", "--model", "deepseek/deepseek-chat"},
			expectedOut:   "architecture.yaml",
			expectedVerb:  false,
			expectedModel: "deepseek/deepseek-chat",
		},
		{
			name:          "all flags",
			args:          []string{"architecture", "--output", "out.yaml", "--verbose", "--model", "gpt-4"},
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

			if cmd.Architecture.Output != tt.expectedOut {
				t.Errorf("expected Output=%q, got %q", tt.expectedOut, cmd.Architecture.Output)
			}
			if cmd.Architecture.Verbose != tt.expectedVerb {
				t.Errorf("expected Verbose=%v, got %v", tt.expectedVerb, cmd.Architecture.Verbose)
			}
			if cmd.Architecture.Model != tt.expectedModel {
				t.Errorf("expected Model=%q, got %q", tt.expectedModel, cmd.Architecture.Model)
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
	r := &ArchitectureCmd{
		Output:  outputPath,
		Verbose: false,
		Model:   "gpt-4",
	}

	err := r.Run()
	require.NoError(t, err, "ArchitectureCmd.Run should not error with model override and mock AI")
}

func TestArchitectureCmd_Run_PromptBuildError(t *testing.T) {
	r := &ArchitectureCmd{
		Output:  "/invalid/path/that/will/fail/architecture.yaml",
		Verbose: false,
		Model:   "",
	}

	err := r.Run()
	require.Error(t, err, "ArchitectureCmd.Run should error when prompt build fails")
	assert.Contains(t, err.Error(), "failed to build architecture prompt")
}
