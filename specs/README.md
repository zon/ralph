# Specs Index

## ralph

- [argo](features/ralph/argo/spec.md) — Convenience CLI commands for inspecting and managing Argo Workflows created by ralph.
- [command](features/ralph/command/spec.md) — Submit an arbitrary command as an Argo Workflow and stream its logs without AI iteration.
- [pass](features/ralph/pass/spec.md) — CLI command for marking a project requirement as passing or failing without editing YAML directly.
- [run](features/ralph/run/spec.md) — Primary entry point that drives an AI coding agent through iterative development cycles until all requirements pass.
- [run-local](features/ralph/run-local/spec.md) — Runs the full development loop in-process on the local machine without submitting an Argo Workflow.
- [run-remote](features/ralph/run-remote/spec.md) — Submits an Argo Workflow to a Kubernetes cluster and returns after submission for remote execution.
- [set-config](features/ralph/set-config/spec.md) — One-shot setup of all Kubernetes credentials required for ralph remote execution on Argo Workflows.
- [set-skills](features/ralph/set-skills/spec.md) — Fetch and install Claude Code skills from the ralph GitHub repository into the invoking project.
- [validate](features/ralph/validate/spec.md) — Checks that a project YAML file is well-formed, repairs it via a local agent if not, and rewrites it in canonical format.
- [workflow-command](features/ralph/workflow-command/spec.md) — Container entrypoint that clones the current branch and runs supplied command tokens in the ralph environment.
- [workflow-token](features/ralph/workflow-token/spec.md) — Generate a GitHub App installation token and configure git HTTPS authentication for use inside Argo Workflow containers.
- [workflow-comment](features/ralph/workflow-comment/spec.md) — Prompts the AI agent with a PR comment body and runs one development iteration against its instructions.
- [workflow-merge](features/ralph/workflow-merge/spec.md) — Runs pre-merge operations and performs the merge after the workspace is ready.
- [workflow-run](features/ralph/workflow-run/spec.md) — Executes the project loop after workspace setup by synchronizing the base branch and delegating to run-local.
- [workflow-workspace](features/ralph/workflow-workspace/spec.md) — Shared container bootstrap for all workflow subcommands: auth, credentials, git setup, clone, and checkout.
- [write-project](features/ralph/write-project/spec.md) — Defines the format of ralph project YAML files and the rules for writing them.

## webhook

- [config](features/webhook/config/spec.md) — Configure the ralph-webhook service with per-repo settings, webhook secrets, and global defaults.
- [set-config](features/webhook/set-config/spec.md) — One-shot setup of all Kubernetes resources required for the ralph-webhook service to handle GitHub webhook events.
- [events](features/webhook/events/spec.md) — Receive GitHub webhook events for pull requests and dispatch Argo Workflows for comments and merges.
