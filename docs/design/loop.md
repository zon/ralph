# Loop Design

`ralph loop` runs a bounded AI iteration loop over a list of steps. It is the sibling of `ralph run`, which iterates over project items instead of steps.

## Invocation

The command takes an optional slug argument and any number of `--step` flags. At least one of them is required.

* A slug looks up a `loops:` entry in the [config](app/ralph-config.md). The entry supplies the steps.
* `--step` flags replace the entry's steps entirely.
* Steps without a slug make Ralph ask the AI for a slug, used only to name the branch.

A slug that matches no entry fails with `loop config not found: <slug>`.

## Iterations

The loop runs on the `loop-<slug>` branch, created from the current branch when it does not exist. The prompt embeds the resolved steps in order and asks for a brief summary in `report.md`. When nothing was necessary the agent writes exactly `NOTHING_TO_DO` instead.

Each iteration runs the prompt, then reads `report.md`:

* `NOTHING_TO_DO` stops the loop and creates no commit.
* Any other report is committed with the report as the message, pushed to `loop-<slug>`, and `report.md` is deleted.

The cap is `--max`, else the config entry's `max`, else 20. A cap that is not positive is rejected.

The base branch is the branch the loop branch was created from. Ralph synchronizes it before the first iteration and again before the pull request, following [Base Branch Synchronization Design](base-branch-sync.md). A non-fatal iteration failure follows [Iteration Recovery Design](iteration-recovery.md).

## Modes

`--mode` resolves and behaves as it does for `ralph run`: `local`, `worktree`, or `remote`. See [Run Design](run.md) and [Remote Mode Design](remote.md).

`--follow` applies only to `remote` mode. Ralph warns and ignores it otherwise. The loop body is the same in every mode.

## Finishing

When the loop ends with at least one commit on `loop-<slug>`, Ralph generates a pull request description from the commit log and opens a pull request to the starting branch. When no commit was made, it opens no pull request and exits successfully.

After opening the pull request, a local loop checks the starting branch back out. A worktree loop leaves the starting checkout untouched.

## Related

* [Run Design](run.md)
* [Base Branch Synchronization Design](base-branch-sync.md)
* [Iteration Recovery Design](iteration-recovery.md)
* [Remote Mode Design](remote.md)
* [Config Design](app/ralph-config.md)
