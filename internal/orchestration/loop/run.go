package loop

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
)

// LoopFlags carries the parsed `ralph loop` command line into the
// orchestration.
type LoopFlags struct {
	Slug   string
	Steps  []string
	Max    int
	Mode   string
	Follow bool
}

// Validate rejects flag combinations that have no valid meaning for the
// resolved execution mode. --follow is a workflow-only flag and is rejected
// for local and worktree modes.
func (f LoopFlags) Validate(mode string) error {
	if f.Follow && (mode == config.ModeLocal || mode == config.ModeWorktree) {
		return fmt.Errorf("--follow flag is not applicable with --mode %s", mode)
	}
	return nil
}

// WorktreeClient creates, detects, and removes git worktrees. Worktree mode
// uses it to run the loop in a sibling directory worktree while leaving the
// current checkout untouched.
type WorktreeClient interface {
	CreateWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error)
	BranchCheckedOutInWorktree(branch string, dryRun bool) (*git.WorktreeCommand, bool, error)
	RemoveWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error)
}

// WorkspaceClient changes the process working directory.
type WorkspaceClient interface {
	ChangeDirectory(path string) error
}

// RunCmd orchestrates the ralph loop command. It resolves the execution mode
// as the --mode flag, then the mode field in .ralph/config.yaml, then local,
// the same resolution `ralph run` uses, validates the flags against it, and
// dispatches between local, worktree, and remote execution. It returns
// the resolved slug and steps for the in-process modes so the caller can retain
// them.
type RunCmd struct {
	config    config.Loader
	newLoop   func() (*Cmd, error)
	worktree  WorktreeClient
	workspace WorkspaceClient
	remote    RemoteRunnerClient
}

func NewRunCmd(config config.Loader, newLoop func() (*Cmd, error), worktree WorktreeClient, workspace WorkspaceClient, remote RemoteRunnerClient) *RunCmd {
	return &RunCmd{config: config, newLoop: newLoop, worktree: worktree, workspace: workspace, remote: remote}
}

// Run resolves the execution mode, validates the flags, and dispatches to the
// matching execution path. Remote mode submits a loop workflow. Local mode runs
// the loop in-process in the current checkout. Worktree mode runs the loop
// in-process inside a sibling directory worktree. For the in-process modes it
// returns the resolved slug and steps; for remote mode it returns no result.
func (r *RunCmd) Run(flags LoopFlags) (*Result, error) {
	cfg, err := r.config.Load()
	if err != nil {
		return nil, err
	}
	mode, err := cfg.ResolveMode(flags.Mode)
	if err != nil {
		return nil, err
	}
	if err := flags.Validate(mode); err != nil {
		return nil, err
	}
	switch mode {
	case config.ModeRemote:
		if err := r.remote.Run(flags.Slug, flags.Steps, flags.Max, flags.Follow); err != nil {
			return nil, err
		}
		return nil, nil
	case config.ModeWorktree:
		return r.runWorktree(flags)
	default:
		loopCmd, err := r.newLoop()
		if err != nil {
			return nil, err
		}
		return loopCmd.Run(flags.Slug, flags.Steps, flags.Max)
	}
}

// runWorktree creates a git worktree on the loop-<slug> branch in a sibling
// directory, runs the loop in-process inside it, and removes the worktree when
// the loop ends, whether it succeeds or fails, leaving the current checkout
// untouched. When the loop branch is already checked out in a worktree, it
// returns an error and creates no worktree.
func (r *RunCmd) runWorktree(flags LoopFlags) (result *Result, retErr error) {
	loopCmd, err := r.newLoop()
	if err != nil {
		return nil, err
	}
	result, err = loopCmd.Resolve(flags.Slug, flags.Steps)
	if err != nil {
		return nil, err
	}
	branch := git.LoopBranch(result.Slug)
	_, checkedOut, err := r.worktree.BranchCheckedOutInWorktree(branch, false)
	if err != nil {
		return nil, err
	}
	if checkedOut {
		return nil, fmt.Errorf("branch '%s' is already checked out in another worktree", branch)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	worktree, err := r.worktree.CreateWorktree(branch, false)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if err := loopCmd.RunResolvedInWorktree(result, flags.Max); err != nil {
		return nil, err
	}
	return result, nil
}
