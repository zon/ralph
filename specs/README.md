# Specs Index

## ralph

- [argo](features/ralph/argo/spec.md) — Convenience CLI commands for inspecting and managing Argo Workflows created by ralph.
- [command](features/ralph/command/spec.md) — Submit an arbitrary command as an Argo Workflow and stream its logs without AI iteration.
- [get](features/ralph/get/spec.md) — Read-only commands that report which project items are complete and which are left, from the branch's commit trailers.
- [logs](features/ralph/logs/spec.md) — Print the pod logs of a ralph-owned Argo Workflow, defaulting to the workflow at the top of the list and streaming with `--follow`.
- [loop](features/ralph/loop/spec.md) — Bounded AI iteration loop over a configured or flag-supplied list of steps, committing to a `loop-<slug>` branch and opening a pull request when the loop ends, running with `--mode` in a Git worktree by default, in-process locally, or in an Argo Workflow.
- [run](features/ralph/run/spec.md) — Primary entry point that drives an AI coding agent through iterative development cycles until every project item is complete, selecting the execution mode with `--mode`.
- [run-local](features/ralph/run-local/spec.md) — Runs the development loop in-process in the current checkout (the `local` execution mode).
- [run-remote](features/ralph/run-remote/spec.md) — Submits an Argo Workflow to a Kubernetes cluster and returns after submission for remote execution (the `remote` execution mode).
- [run-worktree](features/ralph/run-worktree/spec.md) — Runs the development loop in-process in a Git worktree created in a sibling directory, leaving the current checkout untouched (the default `worktree` execution mode).
- [set-config](features/ralph/set-config/spec.md) — One-shot setup of all Kubernetes credentials required for ralph remote execution on Argo Workflows.
- [validate](features/ralph/validate/spec.md) — Checks that a project file parses and that the item query resolves, repairs it via a local agent if not, and rewrites it in canonical format.
- [workflow-command](features/ralph/workflow-command/spec.md) — Container entrypoint that clones the current branch and runs supplied command tokens in the ralph environment.
- [workflow-token](features/ralph/workflow-token/spec.md) — Generate a GitHub App installation token and configure git HTTPS authentication for use inside Argo Workflow containers.
- [workflow-comment](features/ralph/workflow-comment/spec.md) — Prompts the AI agent with a PR comment body and runs one development iteration against its instructions.
- [workflow-run](features/ralph/workflow-run/spec.md) — Executes the project loop after workspace setup by synchronizing the base branch and delegating to run-local.
- [workflow-workspace](features/ralph/workflow-workspace/spec.md) — Shared container bootstrap for all workflow subcommands: auth, credentials, git setup, clone, and checkout.

## webhook

- [config](features/webhook/config/spec.md) — Configure the ralph-webhook service with per-repo settings, webhook secrets, and global defaults.
- [set-config](features/webhook/set-config/spec.md) — One-shot setup of all Kubernetes resources required for the ralph-webhook service to handle GitHub webhook events.
- [events](features/webhook/events/spec.md) — Receive GitHub webhook events for pull requests and dispatch Argo Workflows for comment requests.
