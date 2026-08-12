package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestNewLocalRunnerIsNotNil(t *testing.T) {
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	require.NotNil(t, runner)
}

func TestNewLocalRunner_WiresSystemEnvClient(t *testing.T) {
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	_, ok := runner.Env().(*SystemEnvClient)
	require.True(t, ok, "expected runner.env to be *SystemEnvClient")
}

func TestNewLocalRunner_EnvNotInWorkflowByDefault(t *testing.T) {
	os.Unsetenv("RALPH_WORKFLOW_EXECUTION")
	ctx := context.NewContext()
	runner := NewLocalRunner(ctx, "main")
	require.False(t, runner.Env().InWorkflow())
}

// The project client the runner iterates with must carry its commit log and
// warning output, since the first iteration reads completion from the branch
// log before doing anything else.
func TestNewLocalRunner_ProjectClientReadsCompletion(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	testutil.MakeInitialCommit(t, dir)
	t.Chdir(dir)

	base, err := git.GetCurrentBranch()
	require.NoError(t, err)
	require.NoError(t, git.CheckoutOrCreateBranch("completion-branch"))

	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	runner := NewLocalRunner(ctx, "main")

	proj := &project.Project{Items: project.NewItems([]any{"one", "two"})}
	incomplete, err := runner.Project().Incomplete(proj, base)
	require.NoError(t, err)
	require.Len(t, incomplete, 2)
}
