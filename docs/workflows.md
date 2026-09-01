# Workflows

With `--mode remote`, ralph runs your project remotely on Kubernetes using [Argo Workflows](https://argo-workflows.readthedocs.io/). This gives you isolated, containerized execution with proper resource management.

## Overview

When you run `ralph my-feature.yaml --mode remote`, ralph:

1. Generates an Argo Workflow that embeds your project file, the resolved item query, and config
2. Submits it to your Kubernetes cluster
3. The container clones your repository, checks out the current branch, and runs ralph
4. Branches and pull requests are created just like local execution

The item query is resolved once at submission time and travels with the workflow, so a remote run indexes items consistently for its whole lifetime even if `.ralph/config.yaml` changes underneath it. Completion is read from the branch's commit log inside the container, which means a run that is stopped and resubmitted against the same branch resumes where it left off. See [Iterations](iterations.md).

The default mode, `worktree`, runs the loop in-process in a Git worktree created in a sibling directory, leaving your current checkout untouched. To run in the current checkout instead:

```bash
ralph my-feature.yaml --mode local
```

To submit remotely and monitor progress in real time:

```bash
ralph my-feature.yaml --mode remote --watch
```

See [Configuration](config.md#mode) for the `mode` config setting and its precedence.

## Setup

Configure the Kubernetes credentials once with `ralph set remote`, then run remotely. Credentials are stored as Secrets in the target cluster and namespace. See [Configuration](config.md) for workflow settings including custom images, namespaces, and environment variables.

### Quickstart: personal access token

The simplest path reuses your existing GitHub login. With `gh` authenticated, configure everything with one command:

```bash
ralph set remote
```

When no key or token is given, ralph stores the token from your `gh` login, or from `GITHUB_TOKEN` if set. To provide a token explicitly:

```bash
ralph set remote --github-token <token>
```

### Recommended: GitHub App

For teams, a GitHub App gives short-lived installation tokens and fine-grained, repo-scoped access as a bot identity. Create an App, install it on the target repositories, then:

```bash
ralph set remote --github-key <key.pem>
```

### Prerequisites

- Kubernetes cluster with [Argo Workflows](https://argo-workflows.readthedocs.io/en/stable/) installed
- `kubectl` configured with cluster access
- [Argo CLI](https://argo-workflows.readthedocs.io/en/stable/cli/) installed
