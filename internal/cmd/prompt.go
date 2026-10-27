package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/workspace"
)

// promptSlugProposer proposes a branch slug from the prompt.
type promptSlugProposer interface {
	ProposeSlug(prompt string) (string, error)
}

// promptGitClient reports the branch the prompt starts from and switches the
// current checkout to the prompt branch.
type promptGitClient interface {
	CurrentBranch() (string, error)
	SwitchToBranch(slug string) error
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
	config          config.Loader         `kong:"-"`
	git             promptGitClient       `kong:"-"`
	slugProposer    promptSlugProposer    `kong:"-"`
	worktree        promptWorktreeClient  `kong:"-"`
	workspace       promptWorkspaceClient `kong:"-"`

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
// worktree work starts. The command then enters the prompt branch according to
// the resolved mode.
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
	return c.enterPromptBranch()
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

// enterPromptBranch enters the prompt branch according to the resolved mode.
// Local mode switches the current checkout. Worktree mode creates a sibling
// worktree on the branch and removes it when the command ends. Remote mode
// creates the branch inside the workflow container, so the client does no
// local branch work.
func (c *PromptCmd) enterPromptBranch() error {
	switch c.resolved.Mode {
	case config.ModeWorktree:
		return c.runWorktree()
	case config.ModeLocal:
		return c.gitClient().SwitchToBranch(c.resolved.PromptBranch)
	default:
		return nil
	}
}

// runWorktree creates a worktree on the prompt branch in a sibling directory,
// changes into it for the rest of the command, and removes it when the command
// ends. The starting checkout stays on its original branch. A prompt branch
// already checked out in another worktree is refused before one is created.
func (c *PromptCmd) runWorktree() (retErr error) {
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

	return c.workspaceClient().ChangeDirectory(created.Path)
}

// proposeSlug asks the AI for a branch slug from the prompt. The proposal is
// returned before it is cleaned, so the caller controls the branch name.
func (c *PromptCmd) proposeSlug(prompt string) (string, error) {
	proposer := c.slugProposer
	if proposer == nil {
		proposer = &promptAISlugProposer{ctx: createExecutionContext()}
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
	return git.NewClient(createExecutionContext())
}

// worktreeClient returns the wired worktree client, building the real one
// lazily.
func (c *PromptCmd) worktreeClient() promptWorktreeClient {
	if c.worktree != nil {
		return c.worktree
	}
	return git.NewClient(createExecutionContext())
}

// workspaceClient returns the wired workspace client, building the real one
// lazily.
func (c *PromptCmd) workspaceClient() promptWorkspaceClient {
	if c.workspace != nil {
		return c.workspace
	}
	return &workspace.Client{}
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
}

// ProposeSlug asks the AI to propose a branch slug for the prompt.
func (p *promptAISlugProposer) ProposeSlug(prompt string) (string, error) {
	return ai.ProposePromptSlug(p.ctx, opencode.New(), prompt)
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
