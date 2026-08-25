package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/orchestration/loop"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workspace"
)

// LoopCmd is the `ralph loop` command. It resolves the loop config steps by
// slug or uses the passed --step values. When steps are given without a slug,
// it asks the AI for a branch slug. It builds the prompt embedding the steps
// and runs it as an iteration loop. It retains the resolved slug and steps on
// the command for the later loop phases. The execution mode resolves the same
// way `ralph run` resolves it: the --mode flag, then the mode field in
// .ralph/config.yaml, then worktree. --mode local runs the loop in-process on
// the local machine, --mode worktree runs it in-process inside a sibling git
// worktree on the loop-<slug> branch, and --mode remote submits an Argo
// Workflow so the loop runs inside the workflow container. --follow is
// rejected for the local and worktree modes and streams the submitted workflow
// logs for remote mode, sending a success or error desktop notification for
// the slug on completion, suppressed by --no-notify. The --model and --context
// flags resolve the same way `ralph run` resolves them: --model overrides the
// top-level model field in .ralph/config.yaml, which defaults to
// deepseek/deepseek-chat when unset, and --context overrides the Kubernetes
// context used for workflow submission.
type LoopCmd struct {
	Slug     string   `arg:"" optional:"" help:"Slug of the loop configuration in .ralph/config.yaml"`
	Steps    []string `help:"Step to run in the loop (repeatable)" name:"step"`
	Max      int      `help:"Maximum number of iterations" name:"max" default:"20"`
	Verbose  bool     `help:"Enable verbose logging" default:"false"`
	Mode     string   `help:"Execution mode: local, worktree, or remote (default: worktree)" name:"mode" optional:""`
	Follow   bool     `help:"Follow workflow logs after submission (only applicable with --mode remote)" short:"f" default:"false"`
	NoNotify bool     `help:"Disable desktop notifications" default:"false"`
	Model    string   `help:"Override the AI model from config" name:"model" optional:""`
	Context  string   `help:"Kubernetes context to use" name:"context" optional:""`

	// slugProposer proposes a branch slug from steps. Tests inject a fake. When
	// nil, buildLoopCmd builds the real adapter that consults the AI.
	slugProposer loop.SlugProposer `kong:"-"`

	// aiClient runs the loop prompt as an AI agent pass. Tests inject a fake.
	// When nil, buildLoopCmd builds the real adapter.
	aiClient loop.AIClient `kong:"-"`

	// reportReader reads the agent's report from report.md. Tests inject a
	// fake. When nil, buildLoopCmd builds the real adapter.
	reportReader loop.ReportReader `kong:"-"`

	// gitClient commits each iteration to the loop branch and pushes it.
	// Tests inject a fake. When nil, buildLoopCmd builds the real adapter.
	gitClient loop.GitClient `kong:"-"`

	// prClient opens the loop branch's pull request when the loop ends.
	// Tests inject a fake. When nil, buildLoopCmd builds the real adapter.
	prClient loop.PullRequestOpener `kong:"-"`

	// remoteRunner submits the loop workflow when the command runs in remote
	// mode. Tests inject a fake. When nil, newOrchestrationLoopCmd builds the
	// real adapter.
	remoteRunner loop.RemoteRunnerClient `kong:"-"`

	// worktree creates, detects, and removes the loop branch's git worktree.
	// Tests inject a fake. When nil, newOrchestrationLoopCmd builds the real
	// adapter.
	worktree loop.WorktreeClient `kong:"-"`

	// workspace changes the process working directory into and out of the
	// worktree. Tests inject a fake. When nil, newOrchestrationLoopCmd builds
	// the real adapter.
	workspace loop.WorkspaceClient `kong:"-"`

	// resolvedSlug and resolvedSteps retain the resolution of the last Run call
	// so the later loop phases (branch commit, pull request) can use them.
	resolvedSlug  string   `kong:"-"`
	resolvedSteps []string `kong:"-"`
}

// Validate checks the parsed command line before the command runs. Kong wraps
// its error in a usage error, printing the full usage alongside it.
func (c *LoopCmd) Validate() error {
	if c.Slug == "" && len(c.Steps) == 0 {
		return errors.New("a slug or at least one --step is required")
	}
	if c.Max < 1 {
		return fmt.Errorf("--max must be positive, got %d", c.Max)
	}
	return nil
}

// Run resolves the execution mode and dispatches to the matching execution
// path. Remote mode submits a loop workflow and the loop runs inside the
// workflow container. Local mode runs the loop in-process on the local machine.
// Worktree mode runs it in-process inside a sibling directory worktree on the
// loop-<slug> branch.
func (c *LoopCmd) Run() error {
	ctx := createExecutionContext()
	c.applyToContext(ctx)
	result, err := newOrchestrationLoopCmd(ctx, c).Run(loop.LoopFlags{
		Slug:   c.Slug,
		Steps:  c.Steps,
		Max:    c.Max,
		Mode:   c.Mode,
		Follow: c.Follow,
	})
	if err != nil {
		return err
	}
	if result != nil {
		c.resolvedSlug = result.Slug
		c.resolvedSteps = result.Steps
	}
	return nil
}

// applyToContext resolves the command flags into the execution context. The
// --model override and the --context override resolve the same way `ralph run`
// resolves them: the flag wins, otherwise the value from .ralph/config.yaml
// (the model defaulting to deepseek/deepseek-chat) is used downstream.
func (c *LoopCmd) applyToContext(ctx *execcontext.Context) {
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, c.Verbose))
	ctx.SetFollow(c.Follow)
	ctx.SetNoNotify(c.NoNotify)
	ctx.SetModel(c.Model)
	ctx.SetKubeContext(c.Context)
}

// newOrchestrationLoopCmd wires the loop orchestration with the real git,
// workspace, and remote clients, building the in-process loop command lazily
// so remote mode never touches local-only dependencies.
func newOrchestrationLoopCmd(ctx *execcontext.Context, c *LoopCmd) *loop.RunCmd {
	worktreeClient := c.worktree
	if worktreeClient == nil {
		worktreeClient = git.NewClient(ctx)
	}
	workspaceClient := c.workspace
	if workspaceClient == nil {
		workspaceClient = &workspace.Client{}
	}
	remoteRunner := c.remoteRunner
	if remoteRunner == nil {
		remoteRunner = NewLoopRemoteRunner(ctx)
	}
	return loop.NewRunCmd(
		&config.Client{},
		func() (*loop.Cmd, error) { return c.buildLoopCmd(ctx) },
		worktreeClient,
		workspaceClient,
		remoteRunner,
	)
}

// buildLoopCmd wires the in-process loop orchestration, resolving the slug and
// steps, running the prompt as an iteration loop, and opening the loop branch's
// pull request when the loop ends.
func (c *LoopCmd) buildLoopCmd(ctx *execcontext.Context) (*loop.Cmd, error) {
	propose := c.slugProposer
	if propose == nil {
		propose = &loopSlugProposer{ctx: ctx}
	}

	aiClient := c.aiClient
	if aiClient == nil {
		aiClient = &loopAIClient{ctx: ctx}
	}
	reportReader := c.reportReader
	if reportReader == nil {
		reportReader = &loopReportReader{}
	}
	gitClient := c.gitClient
	if gitClient == nil {
		gitClient = git.NewClient(ctx)
	}
	prClient := c.prClient
	if prClient == nil {
		baseBranch, err := git.GetCurrentBranch()
		if err != nil {
			return nil, err
		}
		prClient = &loopPullRequestOpener{client: github.NewClient(ctx, baseBranch, github.NewGH(ctx.Output()), opencode.New())}
	}

	return loop.NewCmd(&config.Client{}, &loopPromptBuilder{}, propose, aiClient, reportReader, gitClient, prClient, &SystemEnvClient{}), nil
}

// loopSlugProposer adapts ai.ProposeLoopSlug to the orchestration's
// SlugProposer interface.
type loopSlugProposer struct {
	ctx *execcontext.Context
}

// ProposeSlug asks the AI to propose a branch slug for the given steps.
func (p *loopSlugProposer) ProposeSlug(steps []string) (string, error) {
	return ai.ProposeLoopSlug(p.ctx, opencode.New(), steps)
}

// loopPromptBuilder adapts ai.BuildLoopPrompt to the orchestration's
// PromptBuilder interface.
type loopPromptBuilder struct{}

// BuildLoopPrompt builds the loop prompt embedding the given steps.
func (b *loopPromptBuilder) BuildLoopPrompt(steps []string) (string, error) {
	return ai.BuildLoopPrompt(steps)
}

// loopAIClient adapts ai.RunAgent to the orchestration's AIClient interface.
type loopAIClient struct {
	ctx *execcontext.Context
}

// RunAgent runs the loop prompt with opencode's configured agent.
func (a *loopAIClient) RunAgent(prompt string) error {
	return ai.RunAgent(a.ctx, opencode.New(), prompt)
}

// PrintStats prints the accumulated AI token usage and cost statistics, using
// the same formatting as `ralph run`.
func (a *loopAIClient) PrintStats() {
	NewAgentClient(a.ctx, opencode.New()).PrintStats()
}

// loopReportReader adapts ai.ReadReport to the orchestration's ReportReader
// interface.
type loopReportReader struct{}

// ReadReport reads the agent's report from report.md.
func (r *loopReportReader) ReadReport() (ai.Report, error) {
	return ai.ReadReport()
}

// loopPullRequestOpener opens the loop branch's pull request through the shared
// github client, so the loop PR is generated exactly like `ralph run`: an AI
// summary of the commits on the head branch.
type loopPullRequestOpener struct {
	client *github.Client
}

// OpenLoopPullRequest opens the loop-<slug> pull request.
func (o *loopPullRequestOpener) OpenLoopPullRequest(slug string) error {
	return o.client.CreatePR(&project.Project{Slug: slug}, git.LoopBranch(slug))
}
