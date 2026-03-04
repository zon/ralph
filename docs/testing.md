# Testing

## Strategy

Tests are split into unit tests and integration tests. Unit tests cover individual functions and packages in isolation. Integration tests exercise full request or execution paths end-to-end within the process.

## Assertions

Use `github.com/stretchr/testify` (`assert` and `require`) for all assertions.

## External Dependencies

Tests must not invoke any external tools or services — no `git`, `gh`, `opencode`, or any other CLI or network call. Any function that calls an external dependency must support a dry-run mode. In dry-run mode the function skips the real call and returns inspectable state describing what it would have done. Tests enable dry-run and assert on that state.

## Structure

Use table-driven tests with `t.Run()` subtests. Use `t.TempDir()` for any file system interaction — the testing package cleans it up automatically.

## E2E Tests

E2E tests live in `tests/e2e/` and are guarded with the `//go:build e2e` tag. They submit real Argo Workflows against a dedicated test repository using a live Kubernetes cluster. They must never run as part of the standard test suite.

### Running E2E tests

```
go test -tags e2e -timeout 15m ./tests/e2e/...
```

### Required environment variables

| Variable | Description | Default |
|---|---|---|
| `RALPH_E2E_REPO` | `owner/repo` of the test repository | `zon/ralph-mock` |
| `RALPH_E2E_BRANCH` | Branch the workflow container will clone | `main` |
| `RALPH_E2E_DEBUG_BRANCH` | Ralph source branch to use inside the container via `go run` | current git branch |
| `RALPH_E2E_NAMESPACE` | Argo Workflows Kubernetes namespace | `ralph-mock` |
| `RALPH_E2E_TIMEOUT` | Per-workflow poll timeout (Go duration, e.g. `"10m"`) | `10m` |

### Preflight test

`TestNamespacePreflight` runs first and verifies that all resources required by the other E2E tests are present in the namespace before any workflow is submitted. It checks:

- Kubernetes secrets: `github-credentials`, `opencode-credentials`
- Files in the test repository: `test-data/e2e-noop-run.yaml`

Failure messages include the exact command needed to fix the missing resource.

### Test project files

Use `test-data/e2e-noop-run.yaml` as the standard test project file. All of its requirements are pre-marked `passing: true`, so the iteration loop exits immediately without invoking the AI.

> **Note:** Because all requirements are already passing, the AI makes no commits. This means `gh pr create` will fail if the branch has no new commits relative to `main`. This is a known limitation — the remote workflow test requires at least one commit on the branch to succeed.

### Cleanup

Each E2E test registers a `t.Cleanup` function that closes any open PRs and deletes the remote branch created during the test. Cleanup failures are logged but do not fail the test.

### Helper

Use `testutil.NewE2EContext(t)` to build an execution context from environment variables:

```go
ctx, cfg := testutil.NewE2EContext(t)
// ctx.Repo, ctx.Branch, ctx.DebugBranch are populated from env vars
// cfg.Namespace and cfg.Timeout are available for polling helpers
```
