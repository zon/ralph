# Local Mode Design

Local mode runs a [run](run.md) or [loop](loop.md) in the current checkout. It is the default mode, and it is the mode a [workflow container](workflow.md) uses.

## Setup

Before the first iteration, a local run runs the configured [before commands](services.md), then switches to the project branch, creating it from the current branch when it does not exist. The run then synchronizes the [base branch](base-branch-sync.md) and resolves the item array once.

A local run does not write the project file during the loop. Completion comes from commit trailers. See [Project Design](project.md) and [Completion Design](completion.md).

## Iterations

Each iteration runs in the checkout. [Services](services.md) start before the picker and stop after the commit. Each iteration commits and pushes to the project branch. A failed iteration is not committed and still consumes one iteration from the limit.

## End

After the loop, the run synchronizes the base branch, removes the project file when cleanup is on, opens the pull request, and restores the starting branch. Restoring happens only when a pull request opens and the run started on another branch.

A local run resumes from the branch. Completion is read from the commit log at every iteration, so a run stopped after some items restarts with the remaining items.

## Related

* [Run Design](run.md)
* [Loop Design](loop.md)
* [Workflow Container Design](workflow.md)
* [Services Design](services.md)
* [Base Branch Synchronization Design](base-branch-sync.md)
