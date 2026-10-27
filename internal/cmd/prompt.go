package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/basesync"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workspace"
)

// promptSlugProposer proposes a branch slug from the prompt.
type promptSlugProposer interface {
	ProposeSlug(prompt string) (string, error)
}

// promptGitClient reports the branch the prompt starts from, switches the
// current checkout to the prompt branch, records the branch tip, synchronizes
// the prompt branch with its base branch, and pushes the branch after commits.
type promptGitClient interface {
	CurrentBranch() (string, error)
	SwitchToBranch(slug string) error
	FetchBranch(branch string) error
	NeedsMerge(branch string) (bool, error)
	Merge(branch string) error
	AbortMerge() error
	HeadCommit() (string, error)
	Push() error
	CheckoutBranch(name string) error
}

// promptPRClient generates a pull request summary from the prompt and opens a
// pull request to the resolved base branch.
type promptPRClient interface {
	CreatePromptPR(prompt, slug, head string) error
}

// promptServicesClient runs the before commands and starts, stops, and cleans
// up the configured services around the prompt pass.
type promptServicesClient interface {
	RunBeforeCommands(cfg *config.RalphConfig) error
	Start(cfg *config.RalphConfig) (*services.Manager, error)
	Stop(svc *services.Manager)
	RemoveLogs(cfg *config.RalphConfig)
}

// promptAIClient runs the prompt pass, fixes a service that fails to start,
// and resolves a conflicting base-branch merge. All three may write repository
// code, so they run with the configured agent.
type promptAIClient interface {
	RunPrompt(prompt string) error
	FixServiceStartup(cfg *config.RalphConfig, err error) error
	ResolveMergeConflicts(baseBranch, promptBranch string) error
}

// promptOutputClient logs the warning emitted when the base branch cannot be
// fetched.
type promptOutputClient interface {
	Warnf(format string, a ...any)
}

// promptWorktreeClient creates, detects, and removes git worktrees. Worktree
// mode runs the prompt in a sibling directory worktree while leaving the
// starting checkout untouched.
type promptWorktreeClient interface {
	CreateWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error)
	BranchCheckedOutInWorktree(branch string, dryRun bool) (*git.WorktreeCommand, bool, error)
	RemoveWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error)
}

// promptWorkspaceClient changes the process working directory.
type promptWorkspaceClient interface {
	ChangeDirectory(path string) error
}

// promptResolution carries the mode and branches resolved once for a prompt
// invocation. Base synchronization and the pull request read the same
// BaseBranch value.
type promptResolution struct {
	Config        *config.RalphConfig
	Mode          string
	Slug          string
	CurrentBranch string
	PromptBranch  string
	BaseBranch    string
}

// PromptCmd runs one user-supplied prompt as a single OpenCode pass. The prompt
// comes from positional text, --file, or standard input. The command has no
// project or item options.
type PromptCmd struct {
	Prompt     []string `arg:"" optional:"" help:"Prompt text, joined with single spaces"`
	File       string   `help:"Read the prompt from a file, or - for standard input" name:"file" optional:""`
	Mode       string   `help:"Execution mode: local, worktree, or remote (default: local)" name:"mode" optional:""`
	Base       string   `help:"Override the base branch for PR creation (default: detects from current branch)" name:"base" optional:"" short:"B"`
	Model      string   `help:"The model to use in format of provider/model" name:"model" optional:""`
	Variant    string   `help:"The model variant (provider-specific reasoning effort, e.g., high, max, minimal)" name:"variant" optional:""`
	Agent      string   `help:"Override the opencode agent from config" name:"agent" optional:""`
	Context    string   `help:"The name of the Kubernetes context to use" name:"context" optional:""`
	Namespace  string   `help:"The name of the Kubernetes namespace to use" name:"namespace" short:"n" optional:""`
	NoServices bool     `help:"Skip service startup" default:"false"`
	NoFollow   bool     `help:"Do not follow workflow logs after submission (only applicable with --mode remote)" default:"false"`
	NoNotify   bool     `help:"Disable desktop notifications" default:"false"`
	Debug      string   `help:"Checkout the given Ralph repo branch in the workflow container and invoke Ralph via 'go run' instead of the built binary (only applicable with --mode remote)" name:"debug" optional:""`
	Verbose    bool     `help:"Enable verbose logging" default:"false"`

	stdin           io.Reader             `kong:"-"`
	stdinIsTerminal func() bool           `kong:"-"`
	ctx             *execcontext.Context  `kong:"-"`
	config          config.Loader         `kong:"-"`
	git             promptGitClient       `kong:"-"`
	slugProposer    promptSlugProposer    `kong:"-"`
	worktree        promptWorktreeClient  `kong:"-"`
	workspace       promptWorkspaceClient `kong:"-"`
	services        promptServicesClient  `kong:"-"`
	ai              promptAIClient        `kong:"-"`
	pr              promptPRClient        `kong:"-"`
	output          promptOutputClient    `kong:"-"`

	// promptBranch is the branch the prompt runs on. The slug step supplies
	// it; base resolution needs it to decide when to fall back to the
	// configured default branch.
	promptBranch string `kong:"-"`

	// resolved retains the mode and branches resolved for this invocation so
	// later steps reuse them instead of recomputing.
	resolved *promptResolution `kong:"-"`
}

// Validate rejects the conflicting prompt sources during command-line parsing.
// The remaining prompt checks need standard input or file access and run in
// Run before any mode work starts.
func (c *PromptCmd) Validate() error {
	if len(c.Prompt) > 0 && c.File != "" {
		return errors.New("cannot use positional prompt text with --file")
	}
	return nil
}

// Run validates the prompt source, resolves the mode, proposes the slug, and
// resolves the branches. A failed slug proposal returns before any branch or
// worktree work starts. The command then enters the prompt branch, synchronizes
// the base branch, starts the services, and runs the pass.
func (c *PromptCmd) Run() error {
	prompt, err := c.readPrompt()
	if err != nil {
		return err
	}
	cfg, mode, err := c.resolveMode()
	if err != nil {
		return err
	}
	if err := validatePromptDebug(c.Debug, mode); err != nil {
		return err
	}
	c.executionContext()
	slug, err := c.proposeSlug(prompt)
	if err != nil {
		return err
	}
	c.promptBranch = git.SanitizeBranchName(slug)
	resolved, err := c.resolve(cfg, mode, slug)
	if err != nil {
		return err
	}
	c.resolved = resolved
	return c.enterPromptBranch(prompt)
}

// readPrompt reads the prompt from the source selected by the flags. Tests
// inject standard input and terminal detection through the command fields.
func (c *PromptCmd) readPrompt() (string, error) {
	stdin := c.stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	isTerminal := c.stdinIsTerminal
	if isTerminal == nil {
		isTerminal = stdinIsTerminal
	}
	return ReadPrompt(c.Prompt, c.File, stdin, isTerminal())
}

// resolve resolves both branches once from the already resolved mode and
// prompt branch. The current branch is read only after the slug proposal, so a
// failed proposal starts no git work.
func (c *PromptCmd) resolve(cfg *config.RalphConfig, mode, slug string) (*promptResolution, error) {
	currentBranch, err := c.currentBranch()
	if err != nil {
		return nil, err
	}
	baseBranch := orchestrationRun.ResolveBaseBranch(c.Base, currentBranch, c.promptBranch, cfg.DefaultBranch)
	cfg.Mode = mode
	cfg.Base = baseBranch
	cfg.StartingBranch = currentBranch
	return &promptResolution{
		Config:        cfg,
		Mode:          mode,
		Slug:          slug,
		CurrentBranch: currentBranch,
		PromptBranch:  c.promptBranch,
		BaseBranch:    baseBranch,
	}, nil
}

// enterPromptBranch enters the prompt branch according to the resolved mode
// and runs the setup and pass. Local mode runs the before commands before
// switching the current checkout, then synchronizes, starts services, and runs
// the pass. Worktree mode creates a sibling worktree, runs the before commands
// inside it, and removes it when the command ends. Remote mode creates the
// branch and runs the setup inside the workflow container, so the client does
// no local branch or setup work.
func (c *PromptCmd) enterPromptBranch(prompt string) error {
	switch c.resolved.Mode {
	case config.ModeWorktree:
		return c.runWorktree(prompt)
	case config.ModeLocal:
		if err := c.runBeforeCommands(); err != nil {
			return err
		}
		if err := c.gitClient().SwitchToBranch(c.resolved.PromptBranch); err != nil {
			return err
		}
		return c.runSetupAndPass(prompt, false)
	default:
		return nil
	}
}

// runWorktree creates a worktree on the prompt branch in a sibling directory,
// changes into it for the rest of the command, runs the before commands and
// pass inside it, and removes it when the command ends. The starting checkout
// stays on its original branch. A prompt branch already checked out in another
// worktree is refused before one is created.
func (c *PromptCmd) runWorktree(prompt string) (retErr error) {
	branch := c.resolved.PromptBranch
	worktree := c.worktreeClient()
	_, checkedOut, err := worktree.BranchCheckedOutInWorktree(branch, false)
	if err != nil {
		return err
	}
	if checkedOut {
		return fmt.Errorf("branch '%s' is already checked out in another worktree", branch)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}
	created, err := worktree.CreateWorktree(branch, false)
	if err != nil {
		return err
	}
	defer func() {
		// git refuses to remove the worktree of the current directory, so change
		// back to the main checkout before removing it.
		_ = c.workspaceClient().ChangeDirectory(originalDir)
		if _, removeErr := worktree.RemoveWorktree(branch, false); removeErr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to remove worktree for branch '%s': %w", branch, removeErr)
		}
	}()

	if err := c.workspaceClient().ChangeDirectory(created.Path); err != nil {
		return err
	}
	if err := c.runBeforeCommands(); err != nil {
		return err
	}
	return c.runSetupAndPass(prompt, true)
}

// runBeforeCommands runs the configured before commands once. A required
// failure is returned and stops the command; an optional failure is warned
// about by the services client and execution continues.
func (c *PromptCmd) runBeforeCommands() error {
	return c.servicesClient().RunBeforeCommands(c.resolved.Config)
}

// runSetupAndPass synchronizes the base branch, records the prompt branch tip,
// starts the configured services, and runs the prompt pass. A service that
// fails to start or become healthy triggers the service-fix prompt; a
// successful fix continues the pass without services, while a failed fix fails
// the command. --no-services skips startup and the fix prompt without affecting
// the before commands. After a successful pass it publishes the branch only
// when the tip advanced, so a prompt that made no commits succeeds without a
// second synchronization or push.
func (c *PromptCmd) runSetupAndPass(prompt string, inWorktree bool) error {
	cfg := c.resolved.Config
	aiClient := c.aiClient()
	gitClient := c.gitClient()
	if _, err := basesync.Sync(gitClient, aiClient, c.outputClient(), c.resolved.BaseBranch, c.resolved.PromptBranch, inWorktree); err != nil {
		return err
	}
	tip, err := gitClient.HeadCommit()
	if err != nil {
		return err
	}
	defer c.servicesClient().RemoveLogs(cfg)

	svc, err := c.startServices()
	if err != nil {
		return err
	}
	if svc != nil {
		defer c.servicesClient().Stop(svc)
	}

	if err := aiClient.RunPrompt(prompt); err != nil {
		return err
	}
	return c.publishIfAdvanced(prompt, gitClient, aiClient, tip, inWorktree)
}

// publishIfAdvanced compares the prompt branch tip with the one recorded after
// the first synchronization. An unchanged tip means the prompt made no commits,
// so the command succeeds without a second synchronization or push. An advanced
// tip synchronizes the base branch again, pushes the prompt branch, opens a pull
// request, and restores the starting branch in local mode.
func (c *PromptCmd) publishIfAdvanced(prompt string, gitClient promptGitClient, aiClient promptAIClient, tip string, inWorktree bool) error {
	after, err := gitClient.HeadCommit()
	if err != nil {
		return err
	}
	if after == tip {
		return nil
	}
	if _, err := basesync.Sync(gitClient, aiClient, c.outputClient(), c.resolved.BaseBranch, c.resolved.PromptBranch, inWorktree); err != nil {
		return err
	}
	if err := gitClient.Push(); err != nil {
		return err
	}
	if err := c.createPullRequest(prompt); err != nil {
		return err
	}
	return c.restoreStartingBranch(gitClient, inWorktree)
}

// createPullRequest generates the pull request description from the prompt and
// the commit log and opens a pull request from the prompt branch to the
// resolved base branch. The prompt slug names the pull request when the summary
// has no H1 heading.
func (c *PromptCmd) createPullRequest(prompt string) error {
	return c.prClient().CreatePromptPR(prompt, c.resolved.PromptBranch, c.resolved.PromptBranch)
}

// restoreStartingBranch checks the starting checkout back out after a pull
// request is opened. Worktree mode leaves the starting checkout untouched, and
// the checkout stays put when the prompt began on the prompt branch.
func (c *PromptCmd) restoreStartingBranch(gitClient promptGitClient, inWorktree bool) error {
	if inWorktree {
		return nil
	}
	if c.resolved.CurrentBranch == "" || c.resolved.CurrentBranch == c.resolved.PromptBranch {
		return nil
	}
	return gitClient.CheckoutBranch(c.resolved.CurrentBranch)
}

// startServices starts the configured services unless --no-services is set.
// When startup or the health check fails, it runs the service-fix prompt and
// returns no services when the fix succeeds.
func (c *PromptCmd) startServices() (*services.Manager, error) {
	if c.NoServices {
		return nil, nil
	}
	cfg := c.resolved.Config
	svc, err := c.servicesClient().Start(cfg)
	if err != nil {
		if fixErr := c.aiClient().FixServiceStartup(cfg, err); fixErr != nil {
			return nil, fixErr
		}
		return nil, nil
	}
	return svc, nil
}

// proposeSlug asks the AI for a branch slug from the prompt. The proposal is
// returned before it is cleaned, so the caller controls the branch name.
func (c *PromptCmd) proposeSlug(prompt string) (string, error) {
	proposer := c.slugProposer
	if proposer == nil {
		proposer = &promptAISlugProposer{ctx: c.executionContext(), oc: opencode.New()}
	}
	return proposer.ProposeSlug(prompt)
}

// resolveMode resolves the execution mode from the --mode flag, then the
// config mode, then local, and returns the loaded config with the effective
// mode.
func (c *PromptCmd) resolveMode() (*config.RalphConfig, string, error) {
	loader := c.config
	if loader == nil {
		loader = &config.Client{}
	}
	cfg, err := loader.Load()
	if err != nil {
		return nil, "", err
	}
	mode, err := cfg.ResolveMode(c.Mode)
	if err != nil {
		return nil, "", err
	}
	return cfg, mode, nil
}

// currentBranch reads the branch the prompt starts from.
func (c *PromptCmd) currentBranch() (string, error) {
	return c.gitClient().CurrentBranch()
}

// gitClient returns the wired git client, building the real one lazily.
func (c *PromptCmd) gitClient() promptGitClient {
	if c.git != nil {
		return c.git
	}
	return git.NewClient(c.executionContext())
}

// worktreeClient returns the wired worktree client, building the real one
// lazily.
func (c *PromptCmd) worktreeClient() promptWorktreeClient {
	if c.worktree != nil {
		return c.worktree
	}
	return git.NewClient(c.executionContext())
}

// workspaceClient returns the wired workspace client, building the real one
// lazily.
func (c *PromptCmd) workspaceClient() promptWorkspaceClient {
	if c.workspace != nil {
		return c.workspace
	}
	return &workspace.Client{}
}

// servicesClient returns the wired services client, building the real one
// lazily.
func (c *PromptCmd) servicesClient() promptServicesClient {
	if c.services != nil {
		return c.services
	}
	return services.NewClient(c.realOutputClient())
}

// aiClient returns the wired AI client, building the real one lazily.
func (c *PromptCmd) aiClient() promptAIClient {
	if c.ai != nil {
		return c.ai
	}
	ctx := c.executionContext()
	oc := opencode.New()
	return &promptAIClientAdapter{ctx: ctx, oc: oc, agent: NewAgentClient(ctx, oc)}
}

// prClient returns the wired pull request client, building the real one lazily
// with the resolved base branch.
func (c *PromptCmd) prClient() promptPRClient {
	if c.pr != nil {
		return c.pr
	}
	ctx := c.executionContext()
	return github.NewClient(ctx, c.resolved.BaseBranch, github.NewGH(ctx.Output()), opencode.New())
}

// outputClient returns the wired output client, building the real one lazily.
func (c *PromptCmd) outputClient() promptOutputClient {
	if c.output != nil {
		return c.output
	}
	return c.realOutputClient()
}

// realOutputClient builds the output client backed by the process streams.
func (c *PromptCmd) realOutputClient() *output.Client {
	return output.NewClient(os.Stdout, os.Stderr, c.Verbose)
}

// executionContext returns the command's execution context, creating it once
// from the command flags. The same context reaches the slug, git, worktree, and
// AI clients, so the configured model, variant, and agent apply to every pass.
func (c *PromptCmd) executionContext() *execcontext.Context {
	if c.ctx == nil {
		c.ctx = createExecutionContext()
		c.applyToContext(c.ctx)
	}
	return c.ctx
}

// applyToContext resolves the command flags into the execution context. The
// model, variant, and agent flags override their config values, and the direct
// prompt's context is left without custom instructions so the prompt runs as
// written.
func (c *PromptCmd) applyToContext(ctx *execcontext.Context) {
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(c.realOutputClient())
	ctx.SetNoNotify(c.NoNotify)
	ctx.SetNoServices(c.NoServices)
	ctx.SetFollow(!c.NoFollow)
	ctx.SetDebugBranch(c.Debug)
	ctx.SetBaseBranch(c.Base)
	ctx.SetModel(c.Model)
	ctx.SetVariant(c.Variant)
	ctx.SetAgent(c.Agent)
	ctx.SetKubeContext(c.Context)
	ctx.SetKubeNamespace(c.Namespace)
}

// ReadPrompt returns the prompt from exactly one source: positional text
// joined with single spaces, a file, or standard input. Standard input is used
// when neither positional text nor --file is given and it is not a terminal,
// and also when --file is "-" regardless of the terminal. The prompt is
// returned as written; only its emptiness is checked after trimming space.
func ReadPrompt(text []string, file string, stdin io.Reader, terminal bool) (string, error) {
	if len(text) > 0 && file != "" {
		return "", errors.New("cannot use positional prompt text with --file")
	}
	if stdin == nil {
		stdin = os.Stdin
	}

	var prompt string
	switch {
	case file == "-":
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt from standard input: %w", err)
		}
		prompt = string(data)
	case file != "":
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file %q: %w", file, err)
		}
		prompt = string(data)
	case len(text) > 0:
		prompt = strings.Join(text, " ")
	case !terminal:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt from standard input: %w", err)
		}
		prompt = string(data)
	default:
		return "", errors.New("prompt is required: provide positional text, --file, or standard input")
	}

	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is empty")
	}
	return prompt, nil
}

// promptAISlugProposer adapts ai.ProposePromptSlug to the command's slug
// proposer. The slug proposal is a supporting prompt, so it runs with
// OpenCode's primary agent.
type promptAISlugProposer struct {
	ctx *execcontext.Context
	oc  opencode.OCClient
}

// ProposeSlug asks the AI to propose a branch slug for the prompt.
func (p *promptAISlugProposer) ProposeSlug(prompt string) (string, error) {
	oc := p.oc
	if oc == nil {
		oc = opencode.New()
	}
	return ai.ProposePromptSlug(p.ctx, oc, prompt)
}

// promptAIClientAdapter adapts the AI entry points to the command's AI client.
// The prompt pass, the service fix, and conflict resolution all run with the
// configured agent because they may write repository code.
type promptAIClientAdapter struct {
	ctx   *execcontext.Context
	oc    opencode.OCClient
	agent *AgentClient
}

// RunPrompt runs the prompt text as one non-interactive OpenCode pass.
func (a *promptAIClientAdapter) RunPrompt(prompt string) error {
	return ai.RunAgent(a.ctx, a.oc, prompt)
}

// FixServiceStartup runs the service-fix prompt with the configured agent.
func (a *promptAIClientAdapter) FixServiceStartup(cfg *config.RalphConfig, err error) error {
	return a.agent.FixServiceStartup(cfg, err)
}

// ResolveMergeConflicts runs the conflict-resolution prompt with the
// configured agent.
func (a *promptAIClientAdapter) ResolveMergeConflicts(baseBranch, promptBranch string) error {
	return a.agent.ResolveMergeConflicts(baseBranch, promptBranch)
}

// validatePromptDebug rejects --debug outside remote mode, where there is no
// workflow container to run from a source branch.
func validatePromptDebug(debug, mode string) error {
	if debug != "" && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--debug flag is not applicable with --mode %s", mode)
	}
	return nil
}

// stdinIsTerminal reports whether standard input is an interactive terminal.
func stdinIsTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
