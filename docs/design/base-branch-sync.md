# Base Branch Synchronization Design

`ralph run` and `ralph loop` keep their working branch current with its base branch. The behavior is shared so a run and a loop merge the same way in every execution mode.

## Synchronization Points

Ralph synchronizes the working branch twice:

* Before the first iteration, so the work starts from the base branch's latest commits.
* Immediately before the pull request, so the pull request contains those commits.

The base branch is resolved by the calling command. A run resolves it from `--base`, the current branch, or the config. A loop uses the branch its loop branch was created from. Synchronization never recomputes it.

Synchronization runs in every mode. In `local` mode it runs in the current checkout, in `worktree` mode inside the worktree, and in `remote` mode inside the workflow container.

## Fetch and Merge

Ralph fetches the base branch and merges it only when the base branch is not already contained in the working branch. An up-to-date branch is left alone.

When the fetch fails, Ralph logs a warning and continues without merging, so a network problem does not end a run.

After the pre-pull-request merge, Ralph pushes the merge so the pull request sees it.

## Conflicts

When the merge conflicts, Ralph aborts it and asks the configured AI agent to resolve the conflicts, run the tests, and stage the result. The agent is the one that writes repository code, so this follows the calling command's agent rules. See [Run Design](run.md) and [AI Design](ai.md).

If the agent fails, the command returns an error and, before a pull request, opens none. Ralph does not leave the repository in a conflicted, mid-merge state.

## Related

* [Run Design](run.md)
* [Loop Design](loop.md)
* [Remote Mode Design](remote.md)
