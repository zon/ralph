# Manual installation

Install Ralph from source instead of [Homebrew](../README.md#installation). Ralph shells out to Git, GitHub CLI, and OpenCode, so install those first.

## Prerequisites

- [Go](https://go.dev/doc/install)
- [Git](https://git-scm.com/install/)
- [GitHub CLI](https://cli.github.com/)
- [OpenCode](https://opencode.ai/docs/#install)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Argo Workflows CLI](https://argo-workflows.readthedocs.io/en/latest/installation/)

## Install Ralph

```bash
go install github.com/zon/ralph/cmd/ralph@latest
```

Ensure `$GOPATH/bin` is on your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## Run

Run help to get started:

```bash
ralph --help
```
