# Specs Index

## Ralph

- [argo](ralph/argo.md) — Convenience CLI commands for inspecting and managing Argo Workflows created by Ralph.
- [command](ralph/command.md) — Submit an arbitrary command as an Argo Workflow and stream its logs without AI iteration.
- [completion](ralph/completion.md) — Read-only commands (`ralph complete` and `ralph incomplete`) that report which project items are complete and which are left, from the branch's commit trailers.
- [kube-options](ralph/kube-options.md) — Shared contract for every Ralph CLI command that interacts with a Kubernetes cluster through kubectl: all such commands support `--context` and `--namespace` to target a cluster and namespace.
- [logs](ralph/logs.md) — Print the pod logs of a Ralph-owned Argo Workflow, defaulting to the workflow at the top of the list and streaming with `--follow`.
- [model-options](ralph/model-options.md) — Shared contract for every Ralph CLI command that prompts an AI model: `ralph run`, `ralph loop`, and `ralph workflow run` support `--model` and `--variant` to override the model and its reasoning-effort variant from `.ralph/config.yaml`.
- [loop](ralph/loop.md) — Bounded AI iteration loop over a configured or flag-supplied list of steps, committing to a `loop-<slug>` branch and opening a pull request when the loop ends, running with `--mode` in a Git worktree by default, in-process locally, or in an Argo Workflow.
- [run](ralph/run.md) — Primary entry point that drives an AI coding agent through iterative development cycles until every project item is complete, selecting the execution mode with `--mode`.
- [run-local](ralph/run-local.md) — Runs the development loop in-process in the current checkout (the `local` execution mode).
- [run-remote](ralph/run-remote.md) — Submits an Argo Workflow to a Kubernetes cluster and returns after submission for remote execution (the `remote` execution mode).
- [run-worktree](ralph/run-worktree.md) — Runs the development loop in-process in a Git worktree created in a sibling directory, leaving the current checkout untouched (the default `worktree` execution mode).
- [set-remote](ralph/set-remote.md) — One-shot setup of all Kubernetes credentials required for Ralph remote execution on Argo Workflows, accepting a GitHub App key or a personal access token.
- [validate](ralph/validate.md) — Checks that a project file parses and that the item query resolves, repairs it via a local agent if not, and rewrites it in canonical format.
- [workflow-command](ralph/workflow-command.md) — Container entrypoint that clones the current branch and runs supplied command tokens in the Ralph environment.
- [workflow-token](ralph/workflow-token.md) — Configure git HTTPS authentication inside Argo Workflow containers from GitHub App credentials or a stored token, preferring the App credentials when both are present.
- [workflow-run](ralph/workflow-run.md) — Executes the project loop after workspace setup by synchronizing the base branch and delegating to run-local.
- [workflow-workspace](ralph/workflow-workspace.md) — Shared container bootstrap for all workflow subcommands: auth, credentials, git setup, clone, and checkout.
