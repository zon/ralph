package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowLoopCmdParsing covers the `ralph workflow loop` command surface
// that runs inside the workflow container: the required repo, the optional
// clone branch, bot identity defaults, and the slug, repeatable --step, and
// --max flags.
func TestWorkflowLoopCmdParsing(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		"workflow", "loop",
		"--repo", "owner/repo",
		"--clone-branch", "main",
		"--slug", "fmt",
		"--step", "run gofmt",
		"--step", "run go vet",
		"--max", "3",
	})
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", cmd.Workflow.Loop.Repo)
	assert.Equal(t, "main", cmd.Workflow.Loop.CloneBranch)
	assert.Equal(t, "fmt", cmd.Workflow.Loop.Slug)
	assert.Equal(t, []string{"run gofmt", "run go vet"}, cmd.Workflow.Loop.Steps)
	assert.Equal(t, 3, cmd.Workflow.Loop.Max)
}

// TestWorkflowLoopCmdDefaults asserts the container-side command defaults match
// `ralph workflow run`: bot identity, max iterations, and no slug or steps.
func TestWorkflowLoopCmdDefaults(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"workflow", "loop", "--repo", "owner/repo"})
	require.NoError(t, err)
	assert.Equal(t, "ralph-zon[bot]", cmd.Workflow.Loop.BotName, "the bot name defaults to the app bot")
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", cmd.Workflow.Loop.BotEmail, "the bot email defaults to the app bot email")
	assert.Equal(t, 10, cmd.Workflow.Loop.Max, "the max iterations default to 10")
	assert.Empty(t, cmd.Workflow.Loop.Slug, "the slug defaults to empty")
	assert.Empty(t, cmd.Workflow.Loop.Steps, "the steps default to empty")
}

// TestWorkflowLoopCmdHelpText asserts the workflow loop subcommand appears in
// the workflow group help.
func TestWorkflowLoopCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "--help"})
	assert.Contains(t, output, "Run a loop via the workflow engine")
}
