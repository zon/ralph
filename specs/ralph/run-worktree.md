# Run Worktree Mode Specification

## Purpose

Behavior of the `worktree` execution mode (`ralph run --mode worktree`): runs the full development loop in-process in a Git worktree created for the project branch, leaving the current checkout untouched.

## Requirements

### Requirement: Worktree creation

Before running the loop, the command SHALL create a Git worktree for the project branch in a sibling directory of the current repository, named from the project branch (e.g. `../<repo>-<branch>`). The worktree SHALL be created from the current repository, so it shares its commit history, remotes, and configuration. The command SHALL run the development loop inside that worktree and SHALL NOT switch the current checkout to the project branch.

#### Scenario: Worktree created for the project branch

- GIVEN the project slug is `my-feature` and the current branch is `main`
- WHEN worktree setup runs
- THEN a Git worktree on the branch `my-feature` is created in a sibling directory named from that branch
- AND the current checkout stays on `main`

#### Scenario: Current checkout left untouched

- GIVEN the command runs in `worktree` mode
- WHEN the run executes
- THEN the current working tree and its checked-out branch are not changed by the run

#### Scenario: Worktree carries the committed state only

- GIVEN the current checkout has uncommitted changes
- WHEN the worktree is created
- THEN the worktree reflects only the committed state of the repository
- AND the uncommitted changes are not carried into the worktree

#### Scenario: Branch already checked out elsewhere

- GIVEN the project branch is already checked out in another worktree, including the current one
- WHEN worktree setup runs
- THEN an error is returned indicating the branch is already checked out
- AND no execution begins

---

### Requirement: Development loop runs in the worktree

The development loop SHALL behave identically to the `local` mode described in [run-local.md](run-local.md): item array resolution, per-iteration service management, completion read from the commit log, item selection and development, commit after each iteration, cleanup, and PR creation. All work SHALL happen inside the worktree.

#### Scenario: Loop runs inside the worktree

- GIVEN a worktree has been created for the project branch
- WHEN the iteration loop runs
- THEN every iteration, commit, and branch operation happens inside the worktree

#### Scenario: Artifact generation happens in the worktree

- GIVEN the input is an `orchestration.md` or `spec.md` file
- WHEN just-in-time artifact generation runs
- THEN the generated artifacts are committed on the project branch inside the worktree, as described in [run-local.md](run-local.md)

#### Scenario: Pull request opened from the worktree's branch

- GIVEN all items are complete
- WHEN the PR creation step runs
- THEN the project branch is pushed to the remote and a pull request is opened to the base branch, exactly as in `local` mode

---

### Requirement: Worktree removal

After the run ends, whether every item completed and a PR was opened, the iteration limit was reached with items incomplete, or the run failed, the command SHALL remove the worktree.

#### Scenario: Worktree removed after a successful run

- GIVEN a run completes with a pull request opened
- WHEN the run finishes
- THEN the worktree is removed
- AND the current checkout remains on its original branch

#### Scenario: Worktree removed after a failed run

- GIVEN the run fails or exits with items still incomplete
- WHEN the run finishes
- THEN the worktree is removed

#### Scenario: Worktree removed when nothing was committed

- GIVEN every item was already complete before the first iteration and no commits were added
- WHEN the run finishes
- THEN the worktree is removed and the command exits successfully
