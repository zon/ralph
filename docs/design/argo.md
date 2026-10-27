# Argo Design

Ralph submits its remote work as Argo Workflows. Three commands inspect and manage those workflows.

## Workflow Identity

Every workflow Ralph submits carries the label `app.kubernetes.io/managed-by=ralph`. The commands use that label so they see Ralph workflows and not other workflows in the same namespace.

## `ralph list`

Lists Ralph-owned workflows in the target namespace.

## `ralph stop`

Stops the named workflow. A workflow name is required.

## `ralph logs`

Prints the pod logs of a workflow. The workflow name is optional. Without it, Ralph picks the newest workflow from the `ralph list` order. `--follow` (`-f`) streams until the pod ends. When no Ralph workflow exists, the command fails instead of printing nothing.

## Targeting

All three accept `--context` and `--namespace` (`-n`). Context resolves from the flag, then `workflow.context`, then the current kubeconfig context. Namespace resolves from the flag, then `workflow.namespace`, then the context's default namespace. See [Kubernetes Design](kube.md).

## Related

* [Remote Mode Design](remote.md)
* [Kubernetes Design](kube.md)
* [Config Design](app/ralph-config.md)
