package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name:       "mode local parses",
			args:       []string{"run", "--mode", "local", "test.yaml"},
			expectMode: "local",
		},
		{
			name:       "mode worktree parses",
			args:       []string{"run", "--mode", "worktree", "test.yaml"},
			expectMode: "worktree",
		},
		{
			name:       "mode remote parses",
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
			name: "default values",
			args: []string{"run", "test.yaml"},
		},
		{
			name:       "default command - mode local parses",
			args:       []string{"--mode", "local", "test.yaml"},
			expectMode: "local",
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

func TestCommandCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"command", "--help"})
	assert.Contains(t, output, "Run a command in the ralph environment")
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

func TestSetConfigCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"set", "config", "--help"})
	assert.Contains(t, output, "Configure credentials for remote execution")
}

func TestSetHelpDoesNotListSkills(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"set", "--help"})
	assert.NotContains(t, output, "skills")
	assert.Contains(t, output, "config")
}

func TestWorkflowRunCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "run", "--help"})
	assert.Contains(t, output, "Run a project via the workflow engine")
}

func TestWorkflowCommentCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "comment", "--help"})
	assert.Contains(t, output, "Run a comment-triggered workflow iteration")
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
		{name: "workflow comment", args: []string{"workflow", "comment", "--repo", "owner/repo", "--project-branch", "feature", "--comment-body", "test", "--pr", "1", "--repo-owner", "owner", "--repo-name", "repo"}},
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
	})
	require.NoError(t, err)
	require.Equal(t, ".spec.tasks", cmd.Workflow.Run.Items)
	require.True(t, cmd.Workflow.Run.Cleanup)

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
			args:          []string{"run", "--extra-iterations", "3", "test.yaml"},
			expectedValue: 3,
		},
		{
			name:          "default command with explicit value",
			args:          []string{"--extra-iterations", "5", "test.yaml"},
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
