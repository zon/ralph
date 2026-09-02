package cmd

import (
	"os"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/opencode"
	orchestrationLoop "github.com/zon/ralph/internal/orchestration/loop"
	"github.com/zon/ralph/internal/output"
)

// WorkflowLoopCmd is the `ralph workflow loop` command that runs inside the
// workflow container. It prepares the workspace and runs the submitted loop
// in-process, so the loop body behaves identically to `ralph loop --mode
// local`.
type WorkflowLoopCmd struct {
	Repo        string   `help:"GitHub repository (owner/repo)" required:""`
	CloneBranch string   `help:"Branch to clone"`
	BotName     string   `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail    string   `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	Slug        string   `help:"Slug of the loop configuration in .ralph/config.yaml"`
	Steps       []string `help:"Step to run in the loop (repeatable)" name:"step"`
	Max         int      `help:"Maximum number of iterations" name:"max" default:"20"`
	Verbose     bool     `help:"Enable verbose logging" default:"false"`
	Model       string   `help:"The model to use in format of provider/model" name:"model" short:"m"`
	Agent       string   `help:"Override the opencode agent from config" name:"agent"`
	Variant     string   `help:"The model variant (provider-specific reasoning effort, e.g., high, max, minimal)" name:"variant"`

	// workspaceSetup prepares the container workspace before the loop runs.
	// Tests inject a fake. When nil, Run builds the real adapter.
	workspaceSetup orchestrationLoop.WorkspaceSetupClient `kong:"-"`

	// loopRunner runs the loop in-process after the workspace is prepared.
	// Tests inject a fake. When nil, Run builds the real adapter.
	loopRunner orchestrationLoop.LoopRunnerClient `kong:"-"`
}

func (w *WorkflowLoopCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, w.Verbose))
	ctx.SetVerbose(w.Verbose)
	if parts := strings.SplitN(w.Repo, "/", 2); len(parts) == 2 {
		ctx.SetRepoOwner(parts[0])
		ctx.SetRepoName(parts[1])
	}
	ctx.SetBranch(w.CloneBranch)
	ctx.SetBotName(w.BotName)
	ctx.SetBotEmail(w.BotEmail)
	ctx.SetModel(w.Model)
	ctx.SetAgent(w.Agent)
	ctx.SetVariant(w.Variant)
	ctx.SetLocal(true)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	flags := orchestrationLoop.WorkflowLoopFlags{
		Repo:        w.Repo,
		CloneBranch: w.CloneBranch,
		BotName:     w.BotName,
		BotEmail:    w.BotEmail,
		Slug:        w.Slug,
		Steps:       w.Steps,
		Max:         w.Max,
	}
	return w.newOrchestrationWorkflowLoopCmd(ctx).Run(flags)
}

// newOrchestrationWorkflowLoopCmd wires the container-side workflow loop
// command. Tests inject fakes for the workspace setup and loop runner.
func (w *WorkflowLoopCmd) newOrchestrationWorkflowLoopCmd(ctx *execcontext.Context) *orchestrationLoop.WorkflowLoopCmd {
	workspace := w.workspaceSetup
	if workspace == nil {
		workspace = &workspaceSetupAdapter{ctx: ctx}
	}
	runner := w.loopRunner
	if runner == nil {
		runner = &loopWorkflowRunnerAdapter{ctx: ctx}
	}
	return orchestrationLoop.NewWorkflowLoopCmd(workspace, runner)
}

// loopWorkflowRunnerAdapter implements orchestration/loop.LoopRunnerClient and
// runs the loop in-process after the workspace is prepared.
type loopWorkflowRunnerAdapter struct {
	ctx *execcontext.Context
}

func (a *loopWorkflowRunnerAdapter) Run(slug string, steps []string, max int) error {
	baseBranch, err := git.GetCurrentBranch()
	if err != nil {
		return err
	}
	cmd := orchestrationLoop.NewCmd(
		&config.Client{},
		&loopPromptBuilder{},
		&loopSlugProposer{ctx: a.ctx},
		&loopAIClient{ctx: a.ctx},
		&loopReportReader{},
		git.NewClient(a.ctx),
		&loopPullRequestOpener{client: github.NewClient(a.ctx, baseBranch, github.NewGH(a.ctx.Output()), opencode.New())},
		&SystemEnvClient{},
	)
	_, err = cmd.Run(slug, steps, max)
	return err
}
