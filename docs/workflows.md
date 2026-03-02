# Workflows

By default, ralph runs your project remotely on Kubernetes using [Argo Workflows](https://argo-workflows.readthedocs.io/). This gives you isolated, containerized execution with proper resource management.

## Overview

When you run `ralph my-feature.yaml`, ralph:

1. Generates an Argo Workflow that embeds your project file and config
2. Submits it to your Kubernetes cluster
3. The container clones your repository, checks out the current branch, and runs ralph
4. Branches and pull requests are created just like local execution

To run locally instead:

```bash
ralph my-feature.yaml --local
```

To submit remotely and monitor progress in real time:

```bash
ralph my-feature.yaml --watch
```

Before running remotely, configure Kubernetes credentials once with `ralph config git`, `ralph config github`, and `ralph config opencode`. See [CLI reference](cli.md) for all flags and commands, and [Configuration](config.md) for workflow settings including custom images, namespaces, and environment variables.

### Prerequisites

- Kubernetes cluster with [Argo Workflows](https://argo-workflows.readthedocs.io/en/stable/) installed
- `kubectl` configured with cluster access
- [Argo CLI](https://argo-workflows.readthedocs.io/en/stable/cli/) installed
