# Base Branch Synchronization Design

`ralph run`, `ralph loop`, and `ralph prompt` keep their working branch current with its base branch. The behavior is shared so all three merge the same way in every execution mode.

All three synchronize before their first work and again before the pull request. See [Prompt Design](prompt.md).

## Synchronization Points

The synchronization points are the same for all three:

* Before the first iteration or pass, so the work starts from the base branch's latest commits.
* Immediately before the pull request, so the pull request contains those commits.

A prompt that leaves no commits opens no pull request and therefore has no second point.

The base branch is resolved by the calling command. A run and a prompt resolve it from `--base`, the current branch, or the config. A loop uses the branch its loop branch was created from. Synchronization never recomputes it.

Synchronization runs in every mode. In `local` mode it runs in the current checkout, in `worktree` mode inside the worktree, and in `remote` mode inside the workflow container.

## Fetch and Merge

Ralph fetches the base branch and merges it only when the base branch is not already contained in the working branch. An up-to-date branch is left alone.

When the fetch fails, Ralph logs a warning and continues without merging, so a network problem does not end a run, loop, or prompt.

After the pre-pull-request merge, Ralph pushes the merge so the pull request sees it.

## Conflicts

When the merge conflicts, Ralph aborts it and asks the configured AI agent to resolve the conflicts, run the tests, and stage the result. The agent is the one that writes repository code, so this follows the calling command's agent rules. See [Run Design](run.md) and [AI Design](ai.md).

If the agent fails, the command returns an error and, before a pull request, opens none. Ralph does not leave the repository in a conflicted, mid-merge state.

## Related

* [Run Design](run.md)
* [Loop Design](loop.md)
* [Prompt Design](prompt.md)
* [Remote Mode Design](remote.md)
