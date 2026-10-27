# Remote Mode Design

Remote mode submits an Argo Workflow and returns after submission. The workflow runs the same body as local mode inside the Ralph container. See [Workflow Container Design](workflow.md).

## Submission

Before submission, the command confirms that the current branch exists on the remote at the same commit. The workflow clones that branch, so a workflow never builds on commits the remote cannot see.

A run generates a workflow from the resolved inputs:

* the repository and clone branch
* the project path and project branch
* the resolved base branch and item query
* the model, variant, and agent
* the services setting
* the configured image, mounts, environment, labels, and resources

The base branch and item query are resolved locally and passed as arguments, so a config change after submission cannot change a running workflow's items.

The workflow carries the label `app.kubernetes.io/managed-by=ralph`, so [Argo commands](argo.md) can find it. A lock keeps two workflows for one project branch from running at once. A TTL and pod garbage collection clean the workflow up after it completes. It targets the resolved Kubernetes context and namespace. See [Kubernetes Design](kube.md).

The workflow runs the image named by `workflow.image`, which defaults to the Ralph image at the running version. See [Config Design](app/ralph-config.md) and [Deployment Design](deployment.md).

## Follow

Without `--follow`, the command prints the workflow name and an `argo logs` hint, then returns. With `--follow`, it streams logs and waits for the workflow to finish. A followed workflow sends a desktop notification on completion unless notifications are off. See [CLI Design](app/ralph.md).

`--debug <branch>` builds Ralph from that source branch inside the container instead of using the installed binary.

## Command

`ralph command -- <tokens>` uses the same submission path without an AI iteration or a pull request. It checks branch sync, submits the workflow, streams logs by default, and waits for the workflow to finish. `--no-follow` submits without streaming. See [Workflow Container Design](workflow.md).

## Prompt

`ralph prompt` uses the same submission path for one OpenCode pass. It proposes the prompt branch, resolves the base branch, checks branch sync, and submits a `ralph-prompt-` workflow. The container runs the [Prompt Design](prompt.md) flow. Logs follow by default, and `--no-follow` submits without streaming.
