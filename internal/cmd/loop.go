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
)

// LoopCmd is the `ralph loop` command. It resolves the loop config steps by
// slug or uses the passed --step values. When steps are given without a slug,
// it asks the AI for a branch slug. It builds the prompt embedding the steps
// and runs it as an iteration loop. It retains the resolved slug and steps on
// the command for the later loop phases. By default it submits an Argo Workflow
// and the loop runs inside the workflow container; with --local it runs the
// loop in-process on the local machine without submitting a workflow. With
// --follow it streams the submitted workflow logs and waits for the workflow
// to finish, sending a success or error desktop notification for the slug on
// completion, suppressed by --no-notify. The --model and --context flags
// resolve the same way `ralph run` resolves them: --model overrides the
// top-level model field in .ralph/config.yaml, which defaults to
// deepseek/deepseek-chat when unset, and --context overrides the Kubernetes
// context used for workflow submission.
type LoopCmd struct {
	Slug     string   `arg:"" optional:"" help:"Slug of the loop configuration in .ralph/config.yaml"`
	Steps    []string `help:"Step to run in the loop (repeatable)" name:"step"`
	Max      int      `help:"Maximum number of iterations" name:"max" default:"20"`
	Verbose  bool     `help:"Enable verbose logging" default:"false"`
	Local    bool     `help:"Run on this machine instead of in Argo Workflows" default:"false"`
	Follow   bool     `help:"Follow workflow logs after submission (only applicable without --local)" short:"f" default:"false"`
	NoNotify bool     `help:"Disable desktop notifications" default:"false"`
	Model    string   `help:"Override the AI model from config" name:"model" optional:""`
	Context  string   `help:"Kubernetes context to use" name:"context" optional:""`

	// slugProposer proposes a branch slug from steps. Tests inject a fake. When
	// nil, runLocal builds the real adapter that consults the AI.
	slugProposer loop.SlugProposer `kong:"-"`

	// aiClient runs the loop prompt as an AI agent pass. Tests inject a fake.
	// When nil, runLocal builds the real adapter.
	aiClient loop.AIClient `kong:"-"`

	// reportReader reads the agent's report from report.md. Tests inject a
	// fake. When nil, runLocal builds the real adapter.
	reportReader loop.ReportReader `kong:"-"`

	// gitClient commits each iteration to the loop branch and pushes it.
	// Tests inject a fake. When nil, runLocal builds the real adapter.
	gitClient loop.GitClient `kong:"-"`

	// prClient opens the loop branch's pull request when the loop ends.
	// Tests inject a fake. When nil, runLocal builds the real adapter.
	prClient loop.PullRequestOpener `kong:"-"`

	// remoteRunner submits the loop workflow when the command runs without
	// --local. Tests inject a fake. When nil, Run builds the real adapter.
	remoteRunner loop.RemoteRunnerClient `kong:"-"`

	// resolvedSlug and resolvedSteps retain the resolution of the last Run call
	// so the later loop phases (branch commit, pull request) can use them.
	resolvedSlug  string   `kong:"-"`
	resolvedSteps []string `kong:"-"`
}

// Validate checks the parsed command line before the command runs. Kong wraps
// its error in a usage error, printing the full usage alongside it.
func (c *LoopCmd) Validate() error {
	if c.Follow && c.Local {
		return errors.New("--follow flag is not applicable with --local flag")
	}
	if c.Slug == "" && len(c.Steps) == 0 {
		return errors.New("a slug or at least one --step is required")
	}
	if c.Max < 1 {
		return fmt.Errorf("--max must be positive, got %d", c.Max)
	}
	return nil
}

// Run dispatches between the two execution modes. Without --local it submits a
// loop workflow and the loop runs inside the workflow container. With --local it
// runs the loop in-process on the local machine without submitting a workflow.
func (c *LoopCmd) Run() error {
	ctx := createExecutionContext()
	c.applyToContext(ctx)
	if c.Local {
		return c.runLocal(ctx)
	}
	return c.runRemote(ctx)
}

// applyToContext resolves the command flags into the execution context. The
// --model override and the --context override resolve the same way `ralph run`
// resolves them: the flag wins, otherwise the value from .ralph/config.yaml
// (the model defaulting to deepseek/deepseek-chat) is used downstream.
func (c *LoopCmd) applyToContext(ctx *execcontext.Context) {
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, c.Verbose))
	ctx.SetLocal(c.Local)
	ctx.SetFollow(c.Follow)
	ctx.SetNoNotify(c.NoNotify)
	ctx.SetModel(c.Model)
	ctx.SetKubeContext(c.Context)
}

// runLocal wires the orchestration, resolving the slug and steps, running the
// prompt as an iteration loop, and retaining the resolution on the command so
// the later loop phases can use it.
func (c *LoopCmd) runLocal(ctx *execcontext.Context) error {
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
			return err
		}
		prClient = &loopPullRequestOpener{client: github.NewClient(ctx, baseBranch, github.NewGH(ctx.Output()), opencode.New())}
	}

	result, err := loop.NewCmd(&config.Client{}, &loopPromptBuilder{}, propose, aiClient, reportReader, gitClient, prClient, &SystemEnvClient{}).Run(c.Slug, c.Steps, c.Max)
	if err != nil {
		return err
	}
	c.resolvedSlug = result.Slug
	c.resolvedSteps = result.Steps
	return nil
}

// runRemote submits a loop workflow that runs the loop inside the workflow
// container. With --follow it streams the submitted workflow logs and waits for
// the workflow to finish instead of printing the argo logs command.
func (c *LoopCmd) runRemote(ctx *execcontext.Context) error {
	runner := c.remoteRunner
	if runner == nil {
		runner = NewLoopRemoteRunner(ctx)
	}
	return runner.Run(c.Slug, c.Steps, c.Max, c.Follow)
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
