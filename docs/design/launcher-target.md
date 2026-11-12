# Launcher Target Design

A [launcher](launcher.md) does not run the agent in its own pod. The launcher pod clones the repository, submits one remote run into a target namespace, and waits for it. An entry that sets no `namespace` targets the launcher's own namespace.

## Purpose

The agent's pod takes its image, mounts, environment, labels, and resources from the repository's `.ralph/config.yaml` at the clone branch, as any remote `ralph prompt` run does. A config change reaches the next run with no launcher release.

The launcher pod holds the GitHub credentials and the right to submit workflows into the target. It holds nothing the agent's pod holds.

## Entry

An entry can set `namespace`. When it is unset, the chart uses the release namespace. The chart passes the result to the workflow as a parameter, and the container passes it to `ralph workflow prompt --namespace`. Every launcher run passes `--namespace`, so no entry runs the agent in the launcher pod.

The target is a namespace of the launcher's own cluster. Another cluster is outside this design.

## Container

`ralph workflow prompt --namespace <namespace>` runs these steps:

1. Prepare the [workspace](workflow.md). GitHub authentication and the clone are the same. The OpenCode credentials step loads nothing when the mount is absent.
2. Check for an open pull request at the prefix, as any launcher run does. A match ends the command successfully.
3. Run the prompt as a remote run from the clone, following [Prompt Design](prompt.md). Ralph resolves the base, checks branch sync, and submits a `ralph-prompt-` workflow into the target namespace. It sends the prefix and no branch, and asks the AI for nothing. The target workflow proposes the prompt slug.
4. Follow the logs and wait for the target workflow to finish, with notifications off.
5. Exit with the target workflow's result.

The flag wins over `workflow.namespace` in the checkout's config. The checkout's `workflow.context` does not apply, because the pod reaches only its own cluster. See [Kubernetes Design](kube.md#in-cluster).

A failed target workflow fails the launcher workflow. Stopping the launcher workflow leaves the target workflow running, as stopping a local remote command does.

## Rights

The pod reaches the cluster with its ServiceAccount. For each distinct target namespace, the chart renders a Role and a RoleBinding in that namespace for the workflow ServiceAccount. The release namespace is a target whenever an entry sets no `namespace` or names it. The Role allows:

* `create`, `get`, `list`, and `watch` on workflows
* `get`, `list`, and `watch` on pods and pod logs

`target.rbac.create: false` skips the Role and RoleBinding for an operator who grants the rights another way.

## OpenCode Credentials

The launcher pod never runs the agent and never calls a model, so it needs no OpenCode credential. The target workflow proposes the prompt slug, because the launcher pod has no model configuration of its own and the model comes from the repository's `.ralph/config.yaml`. Leave `workflow.credentials.opencodeSecret` empty to mount none. When it names a Secret, the launcher still mounts it.

## Mutex and Lock

The launcher mutex still queues runs of one entry. The target workflow takes no lock, because the launcher sends no branch and a lock named from the prefix would collide with the launcher mutex while the launcher waits for the target. The launcher mutex is the only serialization a launcher run needs.
