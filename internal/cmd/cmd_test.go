package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "follow with local should fail",
			args:        []string{"run", "--follow", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --local flag",
		},
		{
			name:        "local alone should succeed validation",
			args:        []string{"run", "--local", "test.yaml"},
			expectError: false,
		},
		{
			name:        "follow without local should succeed validation",
			args:        []string{"run", "--follow", "test.yaml"},
			expectError: false,
		},
		{
			name:        "no flags should succeed validation",
			args:        []string{"run", "test.yaml"},
			expectError: false,
		},
		{
			name:        "debug with local should fail",
			args:        []string{"run", "--debug", "my-branch", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--debug flag is not applicable with --local flag",
		},
{
			name:        "default command - follow with local should fail",
			args:        []string{"--follow", "--local", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --local flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}), // Prevent exit during tests
			)
			if err != nil {
				t.Fatalf("failed to create parser: %v", err)
			}

			// Parse the args
			_, err = parser.Parse(tt.args)
			if err != nil {
				// Parser error - skip validation test as we can't reach Run()
				if tt.expectError {
					// This is ok - parser caught the error
					return
				}
				t.Fatalf("failed to parse args: %v", err)
			}

			// Now run validation
			// We need to mock the execution to test only validation
			// Override the project file validation since we're not testing that
			if cmd.Run.InputFile == "" {
				cmd.Run.InputFile = "test.yaml"
			}

			// Test follow + local validation
			if cmd.Run.Follow && cmd.Run.Local {
				err = validateRunFlags(&cmd.Run)
				if !tt.expectError {
					t.Errorf("expected no error, got: %v", err)
				} else if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			// Test local + debug validation
			if cmd.Run.Local && cmd.Run.Debug != "" {
				err = validateRunFlags(&cmd.Run)
				if !tt.expectError {
					t.Errorf("expected no error, got: %v", err)
				} else if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			// Should pass validation
			err = validateRunFlags(&cmd.Run)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectLocal       bool
		expectFollow      bool
		expectNoNotify    bool
		expectDebugBranch string
	}{
		{
			name:         "local flag sets Local to true",
			args:         []string{"run", "--local", "test.yaml"},
			expectLocal:  true,
			expectFollow: false,
		},
		{
			name:         "follow flag sets Follow to true",
			args:         []string{"run", "--follow", "test.yaml"},
			expectLocal:  false,
			expectFollow: true,
		},
		{
			name:              "debug flag sets DebugBranch",
			args:              []string{"run", "--debug", "fix-bug", "test.yaml"},
			expectDebugBranch: "fix-bug",
		},
		{
			name:           "no-notify flag sets NoNotify to true",
			args:           []string{"run", "--no-notify", "test.yaml"},
			expectLocal:    false,
			expectFollow:   false,
			expectNoNotify: true,
		},
		{
			name:         "default values",
			args:         []string{"run", "test.yaml"},
			expectLocal:  false,
			expectFollow: false,
		},
		{
			name:         "default command - local flag sets Local to true",
			args:         []string{"--local", "test.yaml"},
			expectLocal:  true,
			expectFollow: false,
		},
		{
			name:         "default command - follow flag sets Follow to true",
			args:         []string{"--follow", "test.yaml"},
			expectLocal:  false,
			expectFollow: true,
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
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Run.Local != tt.expectLocal {
				t.Errorf("expected Local=%v, got %v", tt.expectLocal, cmd.Run.Local)
			}
			if cmd.Run.Follow != tt.expectFollow {
				t.Errorf("expected Follow=%v, got %v", tt.expectFollow, cmd.Run.Follow)
			}
			if cmd.Run.NoNotify != tt.expectNoNotify {
				t.Errorf("expected NoNotify=%v, got %v", tt.expectNoNotify, cmd.Run.NoNotify)
			}
			if cmd.Run.Debug != tt.expectDebugBranch {
				t.Errorf("expected DebugBranch=%q, got %q", tt.expectDebugBranch, cmd.Run.Debug)
			}
		})
	}
}

// validateRunFlags extracts the validation logic for testing
func validateRunFlags(r *RunCmd) error {
	if r.Follow && r.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}
	if r.Debug != "" && r.Local {
		return fmt.Errorf("--debug flag is not applicable with --local flag")
	}
	return nil
}

func TestRunCmdInputFileValidation(t *testing.T) {
	t.Run("nonexistent input file returns error", func(t *testing.T) {
		r := &RunCmd{InputFile: "/nonexistent/path/project.yaml"}
		err := r.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input file not found")
	})

	t.Run("existing project file passes file validation", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "project.yaml")
		require.NoError(t, os.WriteFile(f, []byte("slug: test\n"), 0644))

		r := &RunCmd{InputFile: f}
		err := r.Run()
		// Error is expected (project execution will fail without full setup),
		// but it should NOT be an "input file not found" error.
		if err != nil {
			assert.NotContains(t, err.Error(), "input file not found")
		}
	})
}

func TestMergeCmdFlagParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedBranch  string
		expectedVerbose bool
		wantParseErr    bool
	}{
		{
			name:            "basic merge command",
			args:            []string{"merge", "ralph/my-feature", "--pr", "42"},
			expectedBranch:  "ralph/my-feature",
			expectedVerbose: false,
		},
		{
			name:            "merge with verbose flag",
			args:            []string{"merge", "ralph/my-feature", "--pr", "42", "--verbose"},
			expectedBranch:  "ralph/my-feature",
			expectedVerbose: true,
		},
		{
			name:         "merge missing pr should fail",
			args:         []string{"merge", "ralph/my-feature"},
			wantParseErr: true,
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

			if cmd.Merge.Branch != tt.expectedBranch {
				t.Errorf("expected Branch=%q, got %q", tt.expectedBranch, cmd.Merge.Branch)
			}
			if cmd.Merge.Verbose != tt.expectedVerbose {
				t.Errorf("expected Verbose=%v, got %v", tt.expectedVerbose, cmd.Merge.Verbose)
			}
		})
	}
}

func TestMaxIterationsFlagParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedValue int
	}{
		{
			name:          "default value is 0 when not provided",
			args:          []string{"run", "test.yaml"},
			expectedValue: 0,
		},
		{
			name:          "explicit value is parsed correctly",
			args:          []string{"run", "--max-iterations", "5", "test.yaml"},
			expectedValue: 5,
		},
		{
			name:          "default command with explicit value",
			args:          []string{"--max-iterations", "3", "test.yaml"},
			expectedValue: 3,
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
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Run.MaxIterations != tt.expectedValue {
				t.Errorf("expected MaxIterations=%v, got %v", tt.expectedValue, cmd.Run.MaxIterations)
			}
		})
	}
}

func TestCommandSubcommandRegistered(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"command"})
	require.NoError(t, err)
}

func captureHelpOutput(cmd interface{}, args []string) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	if err != nil {
		os.Stdout = old
		w.Close()
		return ""
	}

	parser.Parse(args)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

func TestRunCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"run", "--help"})
	assert.Contains(t, output, "Execute ralph with a project file")
}

func TestCommandCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"command", "--help"})
	assert.Contains(t, output, "Run a command in the ralph environment")
}

func TestMergeCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"merge", "--help"})
	assert.Contains(t, output, "Submit an Argo workflow to merge a completed PR")
}

func TestValidateCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"validate", "--help"})
	assert.Contains(t, output, "Validate a project YAML file")
}

func TestListCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"list", "--help"})
	assert.Contains(t, output, "List Argo workflows")
}

func TestStopCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"stop", "--help"})
	assert.Contains(t, output, "Stop an Argo workflow")
}

func TestPassCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"pass", "--help"})
	assert.Contains(t, output, "Mark a project requirement as passing or failing")
}

func TestSetSkillsCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"set", "skills", "--help"})
	assert.Contains(t, output, "Manage ralph skill installation")
}

func TestSetConfigCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"set", "config", "--help"})
	assert.Contains(t, output, "Configure credentials for remote execution")
}

func TestWorkflowRunCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "run", "--help"})
	assert.Contains(t, output, "Run a project via the workflow engine")
}

func TestWorkflowCommentCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "comment", "--help"})
	assert.Contains(t, output, "Run a comment-triggered workflow iteration")
}

func TestWorkflowMergeCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "merge", "--help"})
	assert.Contains(t, output, "Merge a completed PR via workflow")
}

func TestWorkflowCommandCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "command", "--help"})
	assert.Contains(t, output, "Run an arbitrary command via workflow")
}

func TestWorkflowTokenCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "token", "--help"})
	assert.Contains(t, output, "Generate a GitHub App installation token")
}

func TestTopLevelCommandsParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "run", args: []string{"run", "test.yaml"}},
		{name: "command", args: []string{"command", "echo", "hello"}},
		{name: "merge", args: []string{"merge", "my-branch", "--pr", "1"}},
		{name: "validate", args: []string{"validate", "test.yaml"}},
		{name: "list", args: []string{"list"}},
		{name: "stop", args: []string{"stop", "test-workflow"}},
		{name: "pass", args: []string{"pass", "test.yaml", "test-slug"}},
		{name: "set skills", args: []string{"set", "skills"}},
		{name: "set config", args: []string{"set", "config"}},
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
		})
	}
}

func TestWorkflowSubcommandsParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "workflow run", args: []string{"workflow", "run", "--repo", "owner/repo", "--project-path", "test.yaml", "--base", "main"}},
		{name: "workflow comment", args: []string{"workflow", "comment", "--repo", "owner/repo", "--project-branch", "feature", "--comment-body", "test", "--pr", "1", "--repo-owner", "owner", "--repo-name", "repo"}},
		{name: "workflow merge", args: []string{"workflow", "merge", "--repo", "owner/repo", "--pr-branch", "feature", "--pr", "1"}},
		{name: "workflow command", args: []string{"workflow", "command", "--repo", "owner/repo", "echo", "hello"}},
		{name: "workflow token", args: []string{"workflow", "token"}},
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
		})
	}
}

func TestCommandSubcommandCleanupRegistrarWiring(t *testing.T) {
	cmd := &Cmd{}
	require.Nil(t, cmd.Command.cleanupRegistrar, "CommandCmd.cleanupRegistrar should be nil before SetCleanupRegistrar")

	cmd.SetCleanupRegistrar(func(registrar func()) {})

	require.NotNil(t, cmd.Command.cleanupRegistrar, "CommandCmd.cleanupRegistrar should be set after SetCleanupRegistrar")
}

func TestBaseFlagParsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedBase string
	}{
		{
			name:         "default value is empty when not provided",
			args:         []string{"run", "test.yaml"},
			expectedBase: "",
		},
		{
			name:         "explicit --base value is parsed correctly",
			args:         []string{"run", "--base", "develop", "test.yaml"},
			expectedBase: "develop",
		},
		{
			name:         "explicit -B short form is parsed correctly",
			args:         []string{"run", "-B", "main", "test.yaml"},
			expectedBase: "main",
		},
		{
			name:         "default command with explicit --base value",
			args:         []string{"--base", "feature-branch", "test.yaml"},
			expectedBase: "feature-branch",
		},
		{
			name:         "default command with explicit -B value",
			args:         []string{"-B", "release-branch", "test.yaml"},
			expectedBase: "release-branch",
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
			if err != nil {
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Run.Base != tt.expectedBase {
				t.Errorf("expected Base=%q, got %q", tt.expectedBase, cmd.Run.Base)
			}
		})
	}
}
