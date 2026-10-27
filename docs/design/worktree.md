# Worktree Mode Design

Worktree mode runs a [run](run.md) or [loop](loop.md) in a Git worktree beside the current checkout. The user's checkout and working tree stay untouched.

## Creation

The command creates the worktree in a sibling directory named from the working branch, `<repo>-<branch>`. The worktree shares the repository's commits, remotes, and configuration.

The worktree starts from the tip `origin` holds for the branch. When the remote has moved ahead and the local branch has no commits the remote lacks, the local branch moves to the remote tip.

When both sides have moved ahead, Ralph stops, because the worktree would build on commits the remote does not share. When origin has no such branch, the worktree starts from the local branch, or from the current HEAD when neither has one.

The command refuses a branch already checked out in a worktree, including the current checkout.

## Inside the Worktree

The run uses the [local mode](local.md) body. It resolves the project file copy inside the worktree, so commits, cleanup, and Git commands stay in the worktree. Before the pull request, [base branch synchronization](base-branch-sync.md) merges `origin/<base>`, because the main checkout may hold the base branch.

## Removal

The command removes the worktree when the run ends, whether it succeeds, fails, or finds nothing to do. The starting checkout keeps its branch and working tree.

## Related

* [Run Design](run.md)
* [Loop Design](loop.md)
* [Local Mode Design](local.md)
* [Base Branch Synchronization Design](base-branch-sync.md)
