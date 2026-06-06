# Specs Index

## ralph

- [argo](features/ralph/argo/spec.md) — Convenience CLI commands for inspecting and managing Argo Workflows created by ralph.
- [command](features/ralph/command/spec.md) — Submit an arbitrary command as an Argo Workflow and stream its logs without AI iteration.
- [config-github](features/ralph/config-github/spec.md) — Store GitHub App credentials as a Kubernetes Secret for ralph's remote execution.
- [config-opencode](features/ralph/config-opencode/spec.md) — Store OpenCode AI provider credentials as a Kubernetes Secret for ralph's remote execution.
- [config-pulumi](features/ralph/config-pulumi/spec.md) — Store a Pulumi access token as a Kubernetes Secret for ralph's remote execution.
- [config-webhook](features/ralph/config-webhook/spec.md) — Provision webhook configuration into a Kubernetes ConfigMap for the ralph webhook service.
- [config-webhook-secret](features/ralph/config-webhook-secret/spec.md) — Generate per-repo webhook secrets, register GitHub webhooks, and store them in Kubernetes.
- [pass-requirement](features/ralph/pass-requirement/spec.md) — CLI command for marking a project requirement as passing or failing without editing YAML directly.
- [run](features/ralph/run/spec.md) — Primary entry point that drives an AI coding agent through iterative development cycles until all requirements pass.
- [run-local](features/ralph/run-local/spec.md) — Runs the full development loop in-process on the local machine without submitting an Argo Workflow.
- [run-remote](features/ralph/run-remote/spec.md) — Submits an Argo Workflow to a Kubernetes cluster and returns after submission for remote execution.
- [set-skills](features/ralph/set-skills/spec.md) — Fetch and install Claude Code skills from the ralph GitHub repository into the invoking project.
- [validate](features/ralph/validate/spec.md) — Checks that a project YAML file is well-formed, repairs it via a local agent if not, and rewrites it in canonical format.
- [workflow-command](features/ralph/workflow-command/spec.md) — Container entrypoint that clones the current branch and runs supplied command tokens in the ralph environment.
- [workflow-comment](features/ralph/workflow-comment/spec.md) — Prompts the AI agent with a PR comment body and runs one development iteration against its instructions.
- [workflow-merge](features/ralph/workflow-merge/spec.md) — Runs pre-merge operations and performs the merge after the workspace is ready.
- [workflow-run](features/ralph/workflow-run/spec.md) — Executes the project loop after workspace setup by synchronizing the base branch and delegating to run-local.
- [workflow-workspace](features/ralph/workflow-workspace/spec.md) — Shared container bootstrap for all workflow subcommands: auth, credentials, git setup, clone, and checkout.
- [write-project](features/ralph/write-project/spec.md) — Defines the format of ralph project YAML files and the rules for writing them.

## webhook

- [config](features/webhook/config/spec.md) — Configure the ralph-webhook service with per-repo settings, webhook secrets, and global defaults.
- [events](features/webhook/events/spec.md) — Receive GitHub webhook events for pull requests and dispatch Argo Workflows for comments and merges.
