# Local Mode Design

Local mode runs a [run](run.md), [loop](loop.md), or [prompt](prompt.md) in the current checkout. It is the default mode, and it is the mode a [workflow container](workflow.md) uses.

## Prompt

A prompt runs the [Prompt Design](prompt.md) flow in the current checkout. Ralph restores the starting branch when the pass leaves commits and the command started on another branch.

## Setup

Before the first iteration, a local run runs the configured [before commands](services.md), then switches to the project branch, creating it from the current branch when it does not exist. The run then synchronizes the [base branch](base-branch-sync.md) and resolves the item array once.

## Iterations

Each iteration runs in the checkout. [Services](services.md) start before the picker and stop after the commit. Each iteration commits and pushes to the project branch.

## End

After the loop, the run synchronizes the base branch, removes the project file when cleanup is on, opens the pull request, and restores the starting branch when it started on another branch.

A local run resumes from the branch. Ralph reads completion from the commit log at every iteration, so a run stopped after some items restarts with the remaining items.
