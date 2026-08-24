package loop

import wksp "github.com/zon/ralph/internal/orchestration/workspace"

// WorkspaceSetupClient prepares the container environment before the loop runs.
type WorkspaceSetupClient interface {
	Setup(flags wksp.WorkspaceFlags) error
}

// LoopRunnerClient runs the loop in-process.
type LoopRunnerClient interface {
	Run(slug string, steps []string, max int) error
}

// WorkflowLoopFlags carries the workflow-side loop invocation: the repo and
// clone branch used to prepare the workspace, and the slug, steps, and maximum
// iterations to run.
type WorkflowLoopFlags struct {
	Repo        string
	CloneBranch string
	BotName     string
	BotEmail    string
	Slug        string
	Steps       []string
	Max         int
}

func (f WorkflowLoopFlags) WorkspaceFlags() wksp.WorkspaceFlags {
	return wksp.WorkspaceFlags{
		Repo:        f.Repo,
		CloneBranch: f.CloneBranch,
		BotName:     f.BotName,
		BotEmail:    f.BotEmail,
	}
}

// WorkflowLoopCmd orchestrates the ralph workflow loop command that runs inside
// the workflow container. It prepares the workspace and runs the loop in-process
// with the submitted slug and steps, so the loop body behaves identically to
// --local.
type WorkflowLoopCmd struct {
	workspace WorkspaceSetupClient
	runner    LoopRunnerClient
}

func NewWorkflowLoopCmd(workspace WorkspaceSetupClient, runner LoopRunnerClient) *WorkflowLoopCmd {
	return &WorkflowLoopCmd{workspace: workspace, runner: runner}
}

func (w *WorkflowLoopCmd) Run(flags WorkflowLoopFlags) error {
	if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
		return err
	}
	return w.runner.Run(flags.Slug, flags.Steps, flags.Max)
}
