package loop

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/basesync"
	"github.com/zon/ralph/internal/git"
)

// LoopConfigClient resolves the steps of the loop config entry matching the
// slug.
type LoopConfigClient interface {
	LoopSteps(slug string) ([]string, error)
}

// PromptBuilder builds the loop prompt that embeds the resolved steps and,
// when non-nil, the error the previous iteration's AI pass failed with.
type PromptBuilder interface {
	BuildLoopPrompt(steps []string, previousErr error) (string, error)
}

// SlugProposer proposes a branch slug for the given steps.
type SlugProposer interface {
	ProposeSlug(steps []string) (string, error)
}

// AIClient runs the loop prompt as one AI agent pass and resolves a
// conflicting base-branch merge.
type AIClient interface {
	RunAgent(prompt string) error
	// IsFatal reports whether an AI pass failure is fatal and must stop the
	// loop immediately rather than be carried forward.
	IsFatal(err error) bool
	// PrintStats prints the accumulated AI token usage and cost statistics.
	PrintStats()
	// ResolveMergeConflicts resolves a base-branch merge conflict, runs the
	// tests, and stages the resolved files.
	ResolveMergeConflicts(baseBranch, projectBranch string) error
}

// OutputClient logs the warning emitted when the base branch cannot be fetched.
type OutputClient interface {
	Warnf(format string, a ...any)
}

// EnvClient reports whether the command is executing inside a workflow
// container, where token usage and cost statistics are printed on completion.
type EnvClient interface {
	InWorkflow() bool
}

// ReportReader reads the agent's report from report.md.
type ReportReader interface {
	ReadReport() (ai.Report, error)
}

// GitClient switches the loop to its branch before the agent runs, commits
// each iteration to the loop branch, pushing it, synchronizes the loop branch
// with the branch it was created from, and checks the starting checkout back
// out after the pull request opens.
type GitClient interface {
	CurrentBranch() (string, error)
	SwitchToLoopBranch(slug string) error
	CommitIterationAndPush(slug string) error
	CheckoutBranch(name string) error
	HasCommitsAhead(base string) (bool, error)
	FetchBranch(branch string) error
	NeedsMerge(branch string) (bool, error)
	Merge(branch string) error
	AbortMerge() error
	Push() error
}

// PullRequestOpener opens a pull request for the loop branch after the
// loop ends. It opens nothing when no commits were made on the loop
// branch, and succeeds.
type PullRequestOpener interface {
	OpenLoopPullRequest(slug string) error
}

// Cmd orchestrates the ralph loop command, from resolving the steps and slug
// through running the iteration loop and opening the loop branch's pull
// request.
type Cmd struct {
	cfg     LoopConfigClient
	prompt  PromptBuilder
	propose SlugProposer
	ai      AIClient
	report  ReportReader
	git     GitClient
	pr      PullRequestOpener
	env     EnvClient
	output  OutputClient
	base    string
}

// Option configures optional dependencies of the loop command.
type Option func(*Cmd)

// WithOutput sets the client that logs synchronization warnings.
func WithOutput(output OutputClient) Option {
	return func(c *Cmd) { c.output = output }
}

func NewCmd(cfg LoopConfigClient, prompt PromptBuilder, propose SlugProposer, ai AIClient, report ReportReader, git GitClient, pr PullRequestOpener, env EnvClient, opts ...Option) *Cmd {
	c := &Cmd{cfg: cfg, prompt: prompt, propose: propose, ai: ai, report: report, git: git, pr: pr, env: env, output: noopOutput{}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// noopOutput discards warnings when no output client is wired, so
// synchronization never fails on a missing logger.
type noopOutput struct{}

func (noopOutput) Warnf(string, ...any) {}

// Result carries the resolution of a loop invocation: the branch slug and the
// steps to run.
type Result struct {
	Slug  string
	Steps []string
}

// Run resolves the branch slug and the steps to run, switches to the loop
// branch so the agent works on its own state, synchronizes the loop branch
// with the branch it was created from, builds the loop prompt embedding the
// steps, and runs it as an iteration loop. The loop stops when the agent
// reports nothing to do or after max iterations, whichever comes first. An
// iteration whose agent pass leaves report.md missing or unreadable is not
// committed and the loop runs its next iteration. After the loop ends it opens
// the loop branch's pull request and, in local mode, checks the starting
// checkout back out. It returns the resolution so the caller can
// derive the branch name from the slug. Inside a workflow container the
// accumulated AI token usage and cost statistics are printed at the end of
// execution, whether the loop succeeded or failed.
func (c *Cmd) Run(slug string, steps []string, max int) (*Result, error) {
	result, err := c.Resolve(slug, steps)
	if err != nil {
		return nil, err
	}
	if err := c.runResolved(result, max, false); err != nil {
		return nil, err
	}
	return result, nil
}

// Resolve returns the branch slug and steps to run for the invocation without
// running the loop, and records the branch the loop branch is created from as
// the base branch synchronization merges. Worktree mode resolves before
// creating the loop branch's worktree, so the branch name is known ahead of
// the in-process run and the base is still the current checkout's branch.
func (c *Cmd) Resolve(slug string, steps []string) (*Result, error) {
	result, err := c.resolve(slug, steps)
	if err != nil {
		return nil, err
	}
	base, err := c.git.CurrentBranch()
	if err != nil {
		return nil, err
	}
	c.base = base
	return result, nil
}

// RunResolvedInWorktree runs the loop in-process inside an existing worktree
// that already has the loop branch checked out, without switching branches in
// the current checkout. It runs the same iteration loop as Run and opens the
// loop branch's pull request when the loop ends.
func (c *Cmd) RunResolvedInWorktree(result *Result, max int) error {
	return c.runResolved(result, max, true)
}

// runResolved synchronizes the loop branch with the branch it was created from,
// runs the resolved steps as an iteration loop, opening the loop branch's pull
// request afterwards and restoring the starting checkout once it is open. In
// worktree mode the branch switch is skipped because the worktree already has
// the loop branch checked out.
func (c *Cmd) runResolved(result *Result, max int, inWorktree bool) error {
	if c.env.InWorkflow() {
		defer c.ai.PrintStats()
	}
	if !inWorktree {
		if err := c.git.SwitchToLoopBranch(result.Slug); err != nil {
			return err
		}
	}
	if _, err := basesync.Sync(c.git, c.ai, c.output, c.base, git.LoopBranch(result.Slug), inWorktree); err != nil {
		return err
	}
	if err := c.iterate(result.Steps, max, result.Slug); err != nil {
		return err
	}
	if err := c.syncBaseBranchBeforePR(result, inWorktree); err != nil {
		return err
	}
	if err := c.pr.OpenLoopPullRequest(result.Slug); err != nil {
		return err
	}
	return c.restoreStartingBranch(result, inWorktree)
}

// restoreStartingBranch checks the starting checkout back out after a pull
// request is opened, so a local loop leaves the user on the branch the loop
// branch was created from rather than loop-<slug>. The checkout stays put in
// worktree mode, when the loop began on the loop branch, and when the loop
// branch has no commits ahead of the base branch so no pull request is opened.
func (c *Cmd) restoreStartingBranch(result *Result, inWorktree bool) error {
	if inWorktree {
		return nil
	}
	if c.base == "" || c.base == git.LoopBranch(result.Slug) {
		return nil
	}
	ahead, err := c.git.HasCommitsAhead(c.base)
	if err != nil {
		return err
	}
	if !ahead {
		return nil
	}
	return c.git.CheckoutBranch(c.base)
}

// syncBaseBranchBeforePR fetches and merges the branch the loop branch was
// created from immediately before the pull request is opened and pushes the
// merge so the pull request contains the base branch's latest changes. A fetch
// failure is warned about and skipped like the start-of-run synchronization.
func (c *Cmd) syncBaseBranchBeforePR(result *Result, inWorktree bool) error {
	merged, err := basesync.Sync(c.git, c.ai, c.output, c.base, git.LoopBranch(result.Slug), inWorktree)
	if err != nil {
		return err
	}
	if !merged {
		return nil
	}
	return c.git.Push()
}

// iterate runs the resolved steps as an iteration loop. Each iteration builds
// the loop prompt, invokes the AI, and reads the agent's report. A non-fatal AI
// pass failure is recorded and the loop moves on to the next iteration, whose
// prompt carries the recorded error as the failure of the previous attempt; a
// successful AI pass clears it. A fatal failure stops the loop immediately and
// is returned. An iteration whose agent pass leaves report.md missing or
// unreadable is not committed: the loop moves on to the next iteration instead
// of returning an error, still bounded by the iteration cap. Otherwise the
// iteration commits when the report says work was done. The loop stops when the
// report says nothing to do or after max iterations, whichever comes first.
func (c *Cmd) iterate(steps []string, max int, slug string) error {
	var previousErr error
	for i := 0; i < max; i++ {
		prompt, err := c.prompt.BuildLoopPrompt(steps, previousErr)
		if err != nil {
			return err
		}
		if err := c.ai.RunAgent(prompt); err != nil {
			if c.ai.IsFatal(err) {
				return err
			}
			previousErr = err
			continue
		}
		previousErr = nil
		report, err := c.report.ReadReport()
		if err != nil {
			continue
		}
		if report.IsNothingToDo() {
			return nil
		}
		if err := c.git.CommitIterationAndPush(slug); err != nil {
			return err
		}
	}
	return nil
}

// resolve returns the branch slug and steps to run. A given slug always
// consults the loop config. When steps are passed, they replace the entry's
// steps, otherwise the entry's steps are used. The given slug is returned
// unchanged. Without a slug, the slug proposer proposes one from the passed
// steps. With neither, no slug and no steps are resolved.
func (c *Cmd) resolve(slug string, steps []string) (*Result, error) {
	if slug != "" {
		return c.resolveConfig(slug, steps)
	}
	if len(steps) == 0 {
		return &Result{}, nil
	}
	proposed, err := c.propose.ProposeSlug(steps)
	if err != nil {
		return nil, err
	}
	return &Result{Slug: proposed, Steps: steps}, nil
}

// resolveConfig returns the resolution for the given slug. Passed steps
// replace the matching loop config entry's steps when present, otherwise the
// entry's steps are used.
func (c *Cmd) resolveConfig(slug string, steps []string) (*Result, error) {
	loopSteps, err := c.cfg.LoopSteps(slug)
	if err != nil {
		return nil, err
	}
	if len(steps) > 0 {
		return &Result{Slug: slug, Steps: steps}, nil
	}
	return &Result{Slug: slug, Steps: loopSteps}, nil
}
