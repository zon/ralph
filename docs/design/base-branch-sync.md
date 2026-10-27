# Base Branch Synchronization Design

`ralph run`, `ralph loop`, and `ralph prompt` keep their working branch current with its base branch. The behavior is shared so all three merge the same way in every execution mode.

## Synchronization Points

The synchronization points are the same for all three:

* Before the first iteration or pass, so the work starts from the base branch's latest commits.
* Immediately before the pull request, so the pull request contains those commits.

A prompt that leaves no commits opens no pull request and therefore has no second point.

The calling command resolves the base branch. A run and a prompt resolve it from `--base`, the current branch, or the config. A loop uses the branch its loop branch was created from. Synchronization never resolves it again.

Synchronization runs in every mode. In `local` mode it runs in the current checkout, in `worktree` mode inside the worktree, and in `remote` mode inside the workflow container.

## Fetch and Merge

Ralph fetches the base branch and merges it only when the working branch does not already contain it. Ralph leaves an up-to-date branch alone.

When the fetch fails, Ralph logs a warning and continues without merging, so a network problem does not end a run, loop, or prompt.

After the pre-pull-request merge, Ralph pushes the merge so the pull request sees it.

## Conflicts

When the merge conflicts, Ralph aborts it and asks the configured AI agent to resolve the conflicts, run the tests, and stage the result. The agent writes repository code, so this follows the calling command's agent rules. See [Run Design](run.md) and [AI Design](ai.md).

If the agent fails, the command returns an error and does not open a pull request. Ralph never leaves the repository in a conflicted, mid-merge state.
