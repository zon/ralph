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

	"github.com/zon/ralph/internal/config"
)

func TestModeFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "follow with local should fail",
			args:        []string{"run", "--follow", "--mode", "local", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --mode local",
		},
		{
			name:        "follow with worktree should fail",
			args:        []string{"run", "--follow", "--mode", "worktree", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --mode worktree",
		},
		{
			name:        "local alone should succeed validation",
			args:        []string{"run", "--mode", "local", "test.yaml"},
			expectError: false,
		},
		{
			name:        "follow with remote should succeed validation",
			args:        []string{"run", "--follow", "--mode", "remote", "test.yaml"},
			expectError: false,
		},
		{
			name:        "no flags should succeed validation",
			args:        []string{"run", "test.yaml"},
			expectError: false,
		},
		{
			name:        "follow alone should fail with the worktree default",
			args:        []string{"run", "--follow", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --mode worktree",
		},
		{
			name:        "debug with local should fail",
			args:        []string{"run", "--debug", "my-branch", "--mode", "local", "test.yaml"},
			expectError: true,
			errorMsg:    "--debug flag is not applicable with --mode local",
		},
		{
			name:        "debug with worktree should fail",
			args:        []string{"run", "--debug", "my-branch", "--mode", "worktree", "test.yaml"},
			expectError: true,
			errorMsg:    "--debug flag is not applicable with --mode worktree",
		},
		{
			name:        "default command - follow with local should fail",
			args:        []string{"--follow", "--mode", "local", "test.yaml"},
			expectError: true,
			errorMsg:    "--follow flag is not applicable with --mode local",
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
				if tt.expectError {
					// This is ok - the parser caught an error
					return
				}
				t.Fatalf("failed to parse args: %v", err)
			}

			if cmd.Run.InputFile == "" {
				cmd.Run.InputFile = "test.yaml"
			}

			err = validateRunFlags(&cmd.Run)
			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectMode        string
		expectFollow      bool
		expectNoNotify    bool
		expectDebugBranch string
	}{
		{
			name:       "mode local flag sets Mode",
			args:       []string{"run", "--mode", "local", "test.yaml"},
			expectMode: "local",
		},
		{
			name:       "mode remote flag sets Mode",
			args:       []string{"run", "--mode", "remote", "test.yaml"},
			expectMode: "remote",
		},
		{
			name:         "follow flag sets Follow to true",
			args:         []string{"run", "--follow", "test.yaml"},
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
			expectFollow:   false,
			expectNoNotify: true,
		},
		{
			name:         "default values",
			args:         []string{"run", "test.yaml"},
			expectFollow: false,
		},
		{
			name:         "default command - mode local flag sets Mode",
			args:         []string{"--mode", "local", "test.yaml"},
			expectMode:   "local",
			expectFollow: false,
		},
		{
			name:         "default command - follow flag sets Follow to true",
			args:         []string{"--follow", "test.yaml"},
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

			if cmd.Run.Mode != tt.expectMode {
				t.Errorf("expected Mode=%q, got %q", tt.expectMode, cmd.Run.Mode)
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

// validateRunFlags extracts the validation logic for testing. The mode
// resolves as the --mode flag when passed, otherwise the worktree default.
func validateRunFlags(r *RunCmd) error {
	mode := r.Mode
	if mode == "" {
		mode = config.ModeWorktree
	}
	if r.Follow && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--follow flag is not applicable with --mode %s", mode)
	}
	if r.Debug != "" && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--debug flag is not applicable with --mode %s", mode)
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

func TestSetRemoteCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"set", "remote", "--help"})
	assert.Contains(t, output, "Configure credentials for remote execution")
}

func TestSetHelpDoesNotListSkills(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"set", "--help"})
	assert.NotContains(t, output, "skills")
	assert.Contains(t, output, "remote")
}

func TestWorkflowRunCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "run", "--help"})
	assert.Contains(t, output, "Run a project via the workflow engine")
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
		{name: "validate", args: []string{"validate", "test.yaml"}},
		{name: "list", args: []string{"list"}},
		{name: "stop", args: []string{"stop", "test-workflow"}},
		{name: "set remote", args: []string{"set", "remote"}},
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

func TestSetSkillsNotRegistered(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"set", "skills"})
	require.Error(t, err, "set skills should not parse once the skills command is removed")
}

func TestWorkflowSubcommandsParsed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "workflow run", args: []string{"workflow", "run", "--repo", "owner/repo", "--project-path", "test.yaml", "--base", "main"}},
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

// TestWorkflowRunItemFlagsParsed covers the items that `ralph workflow run`
// accepts an `--items` flag holding the item query and a `--cleanup` flag for
// the cleanup setting, with `--items` defaulting to empty when absent.
func TestWorkflowRunItemFlagsParsed(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		"workflow", "run",
		"--repo", "owner/repo",
		"--project-path", "test.yaml",
		"--base", "main",
		"--items", ".spec.tasks",
		"--cleanup",
		"--model", "gpt-4",
		"--variant", "high",
	})
	require.NoError(t, err)
	require.Equal(t, ".spec.tasks", cmd.Workflow.Run.Items)
	require.True(t, cmd.Workflow.Run.Cleanup)
	require.Equal(t, "gpt-4", cmd.Workflow.Run.Model)
	require.Equal(t, "high", cmd.Workflow.Run.Variant)

	cmd2 := &Cmd{}
	parser2, err := kong.New(cmd2,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser2.Parse([]string{
		"workflow", "run",
		"--repo", "owner/repo",
		"--project-path", "test.yaml",
		"--base", "main",
	})
	require.NoError(t, err)
	require.Equal(t, "", cmd2.Workflow.Run.Items)
	require.False(t, cmd2.Workflow.Run.Cleanup)
}

// TestTopLevelHelpListsNoMergeCommand asserts the top-level help lists no merge
// command now that the merge feature is removed.
func TestTopLevelHelpListsNoMergeCommand(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"--help"})
	assert.NotContains(t, output, "merge")
}

// TestWorkflowHelpListsNoMergeSubcommand asserts the workflow group help lists
// no merge subcommand now that the merge feature is removed.
func TestWorkflowHelpListsNoMergeSubcommand(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "--help"})
	assert.NotContains(t, output, "merge")
}

// TestWorkflowHelpListsNoCommentSubcommand asserts the workflow group help
// lists no comment subcommand now that the workflow comment feature is removed.
func TestWorkflowHelpListsNoCommentSubcommand(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "--help"})
	assert.NotContains(t, output, "comment")
}

func TestExtraIterationsFlagParsing(t *testing.T) {
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
			args:          []string{"run", "--extra", "3", "test.yaml"},
			expectedValue: 3,
		},
		{
			name:          "default command with explicit value",
			args:          []string{"--extra", "5", "test.yaml"},
			expectedValue: 5,
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

			if cmd.Run.ExtraIterations != tt.expectedValue {
				t.Errorf("expected ExtraIterations=%v, got %v", tt.expectedValue, cmd.Run.ExtraIterations)
			}
		})
	}
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

func TestNamespaceFlagParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedContext string
		expectedNs      string
	}{
		{
			name:            "default value is empty when not provided",
			args:            []string{"run", "test.yaml"},
			expectedContext: "",
			expectedNs:      "",
		},
		{
			name:            "explicit --namespace value is parsed correctly",
			args:            []string{"run", "--namespace", "argo", "test.yaml"},
			expectedContext: "",
			expectedNs:      "argo",
		},
		{
			name:            "explicit -n short form is parsed correctly",
			args:            []string{"run", "-n", "staging", "test.yaml"},
			expectedContext: "",
			expectedNs:      "staging",
		},
		{
			name:            "--namespace parses alongside --context",
			args:            []string{"run", "--context", "prod", "--namespace", "argo", "test.yaml"},
			expectedContext: "prod",
			expectedNs:      "argo",
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

			if cmd.Run.Context != tt.expectedContext {
				t.Errorf("expected Context=%q, got %q", tt.expectedContext, cmd.Run.Context)
			}
			if cmd.Run.Namespace != tt.expectedNs {
				t.Errorf("expected Namespace=%q, got %q", tt.expectedNs, cmd.Run.Namespace)
			}
		})
	}
}
