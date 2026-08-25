package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorktreeCommand describes the git worktree command a worktree operation
// would run. Worktree operations return it as inspectable state in dry-run
// mode, so callers and tests can assert on the command without running git.
type WorktreeCommand struct {
	Args []string
	Path string
}

// worktreePathFor returns the sibling directory path a worktree for branch is
// created at, named from the branch: <parent>/<repo>-<branch>.
func worktreePathFor(repoRoot, branch string) string {
	return filepath.Join(filepath.Dir(repoRoot), filepath.Base(repoRoot)+"-"+branch)
}

// worktreePath resolves the repository root and returns the sibling directory
// path for the worktree of branch. In dry-run mode it uses the current working
// directory as the repo root so no git command runs.
func worktreePath(branch string, dryRun bool) (string, error) {
	var repoRoot string
	var err error
	if dryRun {
		repoRoot, err = os.Getwd()
	} else {
		repoRoot, err = FindRepoRoot()
	}
	if err != nil {
		return "", fmt.Errorf("failed to determine repo root: %w", err)
	}
	return worktreePathFor(repoRoot, branch), nil
}

// CreateWorktree creates a Git worktree for branch in a sibling directory
// named from the branch. When the branch does not exist it is created from the
// current HEAD. In dry-run mode it returns the git command it would run
// without executing it, reporting the branch-creating command.
func CreateWorktree(branch string, dryRun bool) (*WorktreeCommand, error) {
	path, err := worktreePath(branch, dryRun)
	if err != nil {
		return nil, err
	}
	var args []string
	if !dryRun && branchExists(branch) {
		args = []string{"worktree", "add", path, branch}
	} else {
		args = []string{"worktree", "add", "-b", branch, path}
	}
	cmd := &WorktreeCommand{Args: args, Path: path}
	if dryRun {
		return cmd, nil
	}
	if _, err := runGit(args...); err != nil {
		return nil, fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
	}
	return cmd, nil
}

// branchExists reports whether branch exists as a local branch.
func branchExists(branch string) bool {
	_, err := runGit("rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// BranchCheckedOutInWorktree reports whether branch is checked out in a
// worktree, including the current one. In dry-run mode it returns the git
// command it would run without executing it, and the branch is reported as not
// checked out.
func BranchCheckedOutInWorktree(branch string, dryRun bool) (*WorktreeCommand, bool, error) {
	cmd := &WorktreeCommand{Args: []string{"worktree", "list", "--porcelain"}}
	if dryRun {
		return cmd, false, nil
	}
	output, err := runGit(cmd.Args...)
	if err != nil {
		return cmd, false, fmt.Errorf("failed to list worktrees: %w", err)
	}
	return cmd, worktreeListHasBranch(output, branch), nil
}

// worktreeListHasBranch reports whether the porcelain worktree list output
// shows branch as a checked-out branch.
func worktreeListHasBranch(output, branch string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "branch refs/heads/"+branch {
			return true
		}
	}
	return false
}

// RemoveWorktree removes the worktree created for branch. In dry-run mode it
// returns the git command it would run without executing it.
func RemoveWorktree(branch string, dryRun bool) (*WorktreeCommand, error) {
	path, err := worktreePath(branch, dryRun)
	if err != nil {
		return nil, err
	}
	cmd := &WorktreeCommand{Args: []string{"worktree", "remove", "--force", path}, Path: path}
	if dryRun {
		return cmd, nil
	}
	if _, err := runGit(cmd.Args...); err != nil {
		return nil, fmt.Errorf("failed to remove worktree at '%s': %w", path, err)
	}
	return cmd, nil
}

func (a *Client) CreateWorktree(branch string, dryRun bool) (*WorktreeCommand, error) {
	return CreateWorktree(branch, dryRun)
}

func (a *Client) BranchCheckedOutInWorktree(branch string, dryRun bool) (*WorktreeCommand, bool, error) {
	return BranchCheckedOutInWorktree(branch, dryRun)
}

func (a *Client) RemoveWorktree(branch string, dryRun bool) (*WorktreeCommand, error) {
	return RemoveWorktree(branch, dryRun)
}
