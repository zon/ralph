# Base Branch Synchronization Specification

## Purpose

The shared behavior that `ralph run` and `ralph loop` use in every execution mode to keep their working branch current with its base branch: merge the base branch in before the first iteration and again before the pull request is opened.

## Requirements

### Requirement: Synchronization points

The command SHALL synchronize the working branch with its base branch before the first iteration and again immediately before the pull request is opened. Synchronization SHALL run in every execution mode: `local` in the current checkout, `worktree` inside the worktree, and `remote` inside the workflow container.

The base branch is the one resolved by the calling command: the value passed to `ralph run` (see [run.md](run.md)) or the branch the loop branch was created from (see [loop.md](loop.md)). The command SHALL NOT recompute the base branch during synchronization.

#### Scenario: Synchronized before the first iteration

- GIVEN the working branch is checked out and the base branch has commits the working branch does not contain
- WHEN execution is about to run the first iteration
- THEN the base branch is fetched and merged into the working branch
- AND the first iteration runs against the merged state

#### Scenario: Synchronized again before the pull request

- GIVEN the work is complete and the base branch has advanced since the working branch last merged it
- WHEN the command is about to open the pull request
- THEN the base branch is fetched and merged into the working branch
- AND the merge is pushed to the remote before the pull request is opened
- AND the pull request contains the merged base changes

#### Scenario: Synchronization in a worktree

- GIVEN the command runs in `worktree` mode
- WHEN synchronization runs
- THEN the fetch and merge happen inside the worktree
- AND the current checkout is left untouched

#### Scenario: Synchronization in a workflow container

- GIVEN the command runs in `remote` mode
- WHEN the container runs the project
- THEN synchronization happens inside the container using the base branch delivered to it

---

### Requirement: Fetch and merge

The command SHALL fetch the base branch and merge it into the working branch when the base branch is not already contained in it.

#### Scenario: Branch is up-to-date

- GIVEN the working branch's merge-base with the base branch equals the base branch tip
- WHEN synchronization runs
- THEN no merge is performed and execution continues

#### Scenario: Clean merge

- GIVEN the working branch is behind the base branch with no conflicts
- WHEN synchronization runs
- THEN the base branch is merged into the working branch (fast-forward or auto-merge)
- AND execution continues

#### Scenario: Base branch fetch failure

- GIVEN the base branch cannot be fetched (e.g., network error)
- WHEN synchronization runs
- THEN a warning is logged and execution continues without merging

---

### Requirement: Merge conflicts resolved by AI

When a merge produces conflicts, the command SHALL abort the merge and invoke an AI agent to resolve the conflicts, run the tests, and stage the resolved files. Because resolving conflicts writes repository code, the invocation SHALL follow the calling command's agent rules (see [run.md](run.md) and [loop.md](loop.md)). If resolution fails, the command SHALL return an error, and when the merge happened before the pull request, no pull request SHALL be opened.

#### Scenario: Conflicts resolved by AI

- GIVEN a merge attempt produces conflicts
- WHEN the merge fails
- THEN the merge is aborted
- AND the configured AI agent is invoked with instructions to resolve all conflicts, run tests, and stage the resolved files
- AND execution continues after resolution

#### Scenario: Conflict resolution failure aborts the run

- GIVEN a merge attempt produces conflicts
- AND the AI conflict resolution fails
- WHEN synchronization finishes
- THEN an error is returned
- AND when the merge happened before the pull request, no pull request is opened

#### Scenario: Failed merge leaves no half-finished state

- GIVEN a merge attempt produces conflicts
- WHEN the AI conflict resolution fails
- THEN the repository is not left in a conflicted, mid-merge state
