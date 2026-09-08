# Manual installation

Install Ralph from source instead of [Homebrew](../README.md#installation). Ralph shells out to Git, GitHub CLI, and OpenCode, so install those first.

## Prerequisites

- [Go](https://go.dev/doc/install)
- [Git](https://git-scm.com/install/)
- [GitHub CLI](https://cli.github.com/)
- [OpenCode](https://opencode.ai/docs/#install)

## Install Ralph

```bash
go install github.com/zon/ralph/cmd/ralph@latest
```

Ensure `$GOPATH/bin` is on your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## Authenticate

Authenticate GitHub:

```bash
gh auth login
```

Configure OpenCode. See [OpenCode authentication docs](https://opencode.ai/docs/cli/#auth).

## Remote workflows

Remote workflows run on Kubernetes through Argo Workflows. To use them, also install kubectl and the Argo CLI:

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Argo Workflows CLI](https://argo-workflows.readthedocs.io/en/latest/installation/)

The Homebrew formula installs both, so skip this section when you install Ralph with Homebrew. Prepare a Kubernetes namespace for Ralph with `ralph setup --help`.
