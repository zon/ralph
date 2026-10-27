# Run Design

A run drives an AI coding agent through the items of a [project file](project.md) until every item is complete, then opens a pull request. It is Ralph's default command. Progress lives in the branch's [completion trailers](completion.md), not in the project file.

## Inputs

A run reads one project file. The [item query](project.md#item-query) selects its item array. The run resolves the query once, before the first iteration, so an item's index and hash stay stable for the whole run.

The project branch comes from the project slug. The base branch comes from `--base`, else the current branch when that differs from the project branch, else `defaultBranch` from [config](app/ralph-config.md).

A run also resolves the execution mode, the extra iteration count, the model, the variant, and the agent. See [Execution Modes](#execution-modes), [AI Design](ai.md), and [Config Design](app/ralph-config.md).

## Iterations

Each iteration works on one item:

1. Read completion from the branch commit log.
2. Start configured [services](services.md).
3. Ask the picker agent to choose one incomplete item.
4. Ask the development agent to finish that item.
5. Commit the changes with the agent's report as the message.
6. Stop the services.

The development agent ends the report with the item's completion trailer when the item is finished. Any report produces a commit, even when no code changed. A report that ends with a trailer marks the item complete. A report without one leaves the item incomplete, even when it committed code. The report file is never committed. Without a report, a changelog prompt supplies the commit message and completes nothing.

A non-fatal agent failure leaves the loop running and passes the failure to the next iteration. A fatal failure stops the run. See [Iteration Recovery Design](iteration-recovery.md).

The loop stops when no item is incomplete, when it reaches the iteration limit, or when the agent leaves `blocked.md`. An iteration that leaves `blocked.md` is committed, then the run returns an error. A run that finds `blocked.md` at the start of an iteration stops without invoking the agent. The limit is the item count plus the extra iterations. The extra count defaults to 30% of the item count, rounded up.

## Execution Modes

`--mode` selects where the loop runs. The configured `mode` supplies the default, and `local` is the final default.

* [Local Mode](local.md) runs in the current checkout.
* [Worktree Mode](worktree.md) runs in a Git worktree beside the checkout.
* [Remote Mode](remote.md) submits an Argo Workflow and runs in a container.

Every mode runs the same iteration body. The mode decides where it runs and how the result reaches the original branch.

## End of Run

A run synchronizes the [base branch](base-branch-sync.md) again before the pull request. It deletes the project file in its own commit when cleanup is on, then generates a summary from the commit log and opens a pull request. No pull request opens when items remain incomplete or the branch has no commits ahead of the base branch.

The pull request title is the first H1 in the generated summary, falling back to the project title and then the slug. See [Project Design](project.md).

## Related

* [Project Design](project.md)
* [Completion Design](completion.md)
* [AI Design](ai.md)
* [Local Mode Design](local.md)
* [Worktree Mode Design](worktree.md)
* [Remote Mode Design](remote.md)
* [Loop Design](loop.md)
* [Base Branch Synchronization Design](base-branch-sync.md)
