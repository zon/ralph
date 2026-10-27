# Kubernetes Design

Commands that reach a Kubernetes cluster resolve a context and a namespace before they act. The same rule applies to remote runs, remote loops, `command`, `setup`, `list`, `stop`, and `logs`.

## Context

The context resolves from highest to lowest:

1. `--context`
2. `workflow.context` in [config](app/ralph-config.md)
3. The current kubeconfig context

When kubectl is missing and no flag or config value names a context, the command fails and tells the user. It does not guess.

## Namespace

The namespace resolves from highest to lowest:

1. `--namespace` (`-n`)
2. `workflow.namespace` in the config
3. The default namespace of the resolved context

`setup` is the exception. It prepares a namespace only when the flag or config names one, and never falls back to the context default. With no namespace, it skips preparation. See [Setup Design](setup.md).

## When a Cluster Is Needed

`ralph run` and `ralph loop` reach the cluster only in `remote` mode. Local and worktree modes run without one. `complete`, `incomplete`, and `validate` have no cluster options, because they never reach a cluster.

`ralph setup` reaches the cluster only when it prepares a namespace.
