package run

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

type RunCmd struct {
	workspace WorkspaceClient
	project   ProjectRepo
	git       GitClient
	worktree  WorktreeClient
	config    config.Loader
	local     LocalRunnerClient
	remote    RemoteRunnerClient
}

type WorkspaceClient interface {
	ChangeDirectory(path string) error
}

type ProjectRepo interface {
	ResolveInputFile(path string) (*project.InputFile, error)
}

type LocalRunnerClient interface {
	RunLocal(input *project.InputFile, cfg *config.RalphConfig) error
	RunLocalInWorktree(input *project.InputFile, cfg *config.RalphConfig) error
}

type RemoteRunnerClient interface {
	Run(input *project.InputFile, flags RunRemoteFlags) error
}

// WorktreeClient creates, detects, and removes git worktrees. Worktree mode
// uses it to run the development loop in a sibling directory worktree while
// leaving the current checkout untouched.
type WorktreeClient interface {
	CreateWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error)
	BranchCheckedOutInWorktree(branch string, dryRun bool) (*git.WorktreeCommand, bool, error)
	RemoveWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error)
}

type ExecutionSetup struct {
	Config        *config.RalphConfig
	Mode          string
	BranchName    string
	CurrentBranch string
	BaseBranch    string
	Model         string
	Agent         string
	Context       string
}

type RunFlags struct {
	WorkingDir      string
	InputFile       string
	ExtraIterations int
	Items           string
	Cleanup         *bool
	Mode            string
	Follow          bool
	Debug           string
	Base            string
	Model           string
	Agent           string
	Context         string
}

// Validate rejects flag combinations that have no valid meaning for the
// resolved execution mode. --follow and --debug are workflow-only flags and are
// rejected for local and worktree modes.
func (f RunFlags) Validate(mode string) error {
	if f.Follow && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--follow flag is not applicable with --mode %s", mode)
	}
	if f.Debug != "" && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--debug flag is not applicable with --mode %s", mode)
	}
	return nil
}

func NewRunCmd(workspace WorkspaceClient, project ProjectRepo, git GitClient, worktree WorktreeClient, config config.Loader, local LocalRunnerClient, remote RemoteRunnerClient) *RunCmd {
	return &RunCmd{
		workspace: workspace,
		project:   project,
		git:       git,
		worktree:  worktree,
		config:    config,
		local:     local,
		remote:    remote,
	}
}

func (r *RunCmd) Run(flags RunFlags) error {
	if err := r.workspace.ChangeDirectory(flags.WorkingDir); err != nil {
		return err
	}
	input, err := r.project.ResolveInputFile(flags.InputFile)
	if err != nil {
		return err
	}
	setup, err := r.prepareSetup(flags, input)
	if err != nil {
		return err
	}
	if err := flags.Validate(setup.Mode); err != nil {
		return err
	}
	switch setup.Mode {
	case config.ModeRemote:
		return r.remote.Run(input, RunRemoteFlags{
			Follow:     flags.Follow,
			Debug:      flags.Debug,
			BaseBranch: setup.BaseBranch,
			Items:      setup.Config.Items,
			Cleanup:    setup.Config.Cleanup,
		})
	case config.ModeWorktree:
		return r.runWorktree(input, setup)
	default:
		return r.local.RunLocal(input, setup.Config)
	}
}

// runWorktree runs the full development loop inside a git worktree created for
// the project branch in a sibling directory. The worktree is removed when the
// run ends, whether it succeeds or fails, and the current checkout stays
// untouched. When the project branch is already checked out in a worktree, it
// returns an error and creates no worktree.
func (r *RunCmd) runWorktree(input *project.InputFile, setup ExecutionSetup) (retErr error) {
	branch := setup.BranchName
	_, checkedOut, err := r.worktree.BranchCheckedOutInWorktree(branch, false)
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
	worktree, err := r.worktree.CreateWorktree(branch, false)
	if err != nil {
		return err
	}
	defer func() {
		// git refuses to remove the worktree of the current directory, so change
		// back to the main checkout before removing it.
		_ = r.workspace.ChangeDirectory(originalDir)
		if _, removeErr := r.worktree.RemoveWorktree(branch, false); removeErr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to remove worktree for branch '%s': %w", branch, removeErr)
		}
	}()

	if err := r.workspace.ChangeDirectory(worktree.Path); err != nil {
		return err
	}
	if err := r.local.RunLocalInWorktree(input, setup.Config); err != nil {
		return err
	}
	return nil
}

func (r *RunCmd) prepareSetup(flags RunFlags, input *project.InputFile) (ExecutionSetup, error) {
	cfg, err := r.config.Load()
	if err != nil {
		return ExecutionSetup{}, err
	}
	mode, err := cfg.ResolveMode(flags.Mode)
	if err != nil {
		return ExecutionSetup{}, err
	}
	cfg.Mode = mode
	currentBranch, err := r.git.CurrentBranch()
	if err != nil {
		return ExecutionSetup{}, err
	}
	projectBranch := git.SanitizeBranchName(input.Slug())
	baseBranch := resolveBaseBranch(flags.Base, currentBranch, projectBranch, cfg.DefaultBranch)
	if flags.ExtraIterations != 0 {
		v := flags.ExtraIterations
		cfg.ExtraIterations = &v
	}
	cfg.Items = cfg.ResolveItems(flags.Items)
	cfg.Cleanup = cfg.ResolveCleanup(flags.Cleanup)
	cfg.Base = baseBranch
	return ExecutionSetup{
		Config:        cfg,
		Mode:          mode,
		BranchName:    projectBranch,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
		Model:         flags.Model,
		Agent:         flags.Agent,
		Context:       flags.Context,
	}, nil
}
