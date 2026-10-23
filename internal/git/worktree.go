package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
// named from the branch. The worktree starts from the branch tip origin holds,
// so a run never builds on work the remote has moved past. When origin has no
// such branch the local branch is used, and when neither has one the branch is
// created from the current HEAD. In dry-run mode it returns the
// branch-creating command it would run without executing it.
func CreateWorktree(branch string, dryRun bool) (*WorktreeCommand, error) {
	path, err := worktreePath(branch, dryRun)
	if err != nil {
		return nil, err
	}
	if dryRun {
		return &WorktreeCommand{Args: []string{"worktree", "add", "-b", branch, path}, Path: path}, nil
	}
	onRemote, err := alignBranchWithRemote(branch)
	if err != nil {
		return nil, err
	}
	args := worktreeAddArgs(path, branch, onRemote)
	cmd := &WorktreeCommand{Args: args, Path: path}
	if _, err := runGit(args...); err != nil {
		return nil, fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
	}
	return cmd, nil
}

// worktreeAddArgs returns the arguments that add the worktree at path: an
// existing local branch is attached, a branch only origin has is created from
// its remote tip, and a branch neither has is created from the current HEAD.
func worktreeAddArgs(path, branch string, onRemote bool) []string {
	if branchExists(branch) {
		return []string{"worktree", "add", path, branch}
	}
	if onRemote {
		return []string{"worktree", "add", "-b", branch, path, "origin/" + branch}
	}
	return []string{"worktree", "add", "-b", branch, path}
}

// alignBranchWithRemote fetches branch from origin and reports whether origin
// has it. It moves the local branch onto the fetched tip when the local branch
// has no commits the remote is missing. History that diverged is an error: the
// run would build on commits the remote does not share, and choosing which side
// to keep is the user's decision.
func alignBranchWithRemote(branch string) (bool, error) {
	if !remoteBranchExists(branch) {
		return false, nil
	}
	if _, err := runGit("fetch", "origin", branch); err != nil {
		return false, fmt.Errorf("failed to fetch branch '%s' from origin: %w", branch, err)
	}
	if !branchExists(branch) {
		return true, nil
	}
	localOnly, remoteOnly, err := commitsAheadBehind(branch, "origin/"+branch)
	if err != nil {
		return false, err
	}
	if localOnly > 0 && remoteOnly > 0 {
		return false, fmt.Errorf("branch '%s' has diverged from 'origin/%s': the local branch has %d commits the remote is missing - push or delete it before running in a worktree", branch, branch, localOnly)
	}
	if remoteOnly == 0 {
		return true, nil
	}
	if _, err := runGit("branch", "-f", branch, "origin/"+branch); err != nil {
		return false, fmt.Errorf("failed to move branch '%s' to its remote tip: %w", branch, err)
	}
	return true, nil
}

// commitsAheadBehind reports how many commits left has that right lacks and how
// many right has that left lacks.
func commitsAheadBehind(left, right string) (ahead, behind int, err error) {
	output, err := runGit("rev-list", "--left-right", "--count", left+"..."+right)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to compare '%s' with '%s': %w", left, right, err)
	}
	counts := strings.Fields(output)
	if len(counts) != 2 {
		return 0, 0, fmt.Errorf("unexpected commit counts comparing '%s' with '%s': %q", left, right, output)
	}
	ahead, err = strconv.Atoi(counts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read commit count %q: %w", counts[0], err)
	}
	behind, err = strconv.Atoi(counts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read commit count %q: %w", counts[1], err)
	}
	return ahead, behind, nil
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
