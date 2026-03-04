# Testing

## Overview

Tests are split into three layers: unit, integration, and end-to-end (E2E).
- **Unit** — individual functions and packages in isolation
- **Integration** — full request or execution paths end-to-end within the process
- **E2E** — real Argo Workflows against a live Kubernetes cluster (never part of the standard suite)

## Conventions

### Assertions

Use `github.com/stretchr/testify` (`assert` and `require`) for all assertions.

### Test structure

Use table-driven tests with `t.Run()` subtests. Use `t.TempDir()` for any filesystem interaction — the testing package cleans it up automatically.

### Isolation

Tests must not invoke any external tools or services — no `git`, `gh`, `opencode`, or any other CLI or network call. Functions that call external dependencies must support a dry-run mode that skips the real call and returns inspectable state. Tests enable dry-run and assert on that state.

## E2E Tests

E2E tests live in `tests/e2e/` and are guarded with the `//go:build e2e` tag. They must never run as part of the standard test suite.

### Running

```
go test -tags e2e -timeout 15m ./tests/e2e/...
```

### How E2E tests work

**Preflight** — `TestNamespacePreflight` runs first and verifies all required resources are present before any workflow is submitted:
- Kubernetes secrets: `github-credentials`, `opencode-credentials`
- Files in the test repository: `test-data/e2e-noop-run.yaml`

Failure messages include the exact command needed to fix the missing resource.

**Test project files** — The standard test project file is `test-data/e2e-noop-run.yaml` in `zon/ralph-mock`. All requirements are pre-marked `passing: true` so the iteration loop exits immediately without invoking the AI.

> **Note:** Because all requirements are already passing, the AI makes no commits. `gh pr create` will fail if the branch has no new commits relative to `main` — the remote workflow test requires at least one commit to succeed.

**Cleanup** — Each test registers a `t.Cleanup` function that closes open PRs and deletes the remote branch. Cleanup failures are logged but do not fail the test.

### Helper

Use `testutil.NewE2EContext(t)` to build an execution context from environment variables:

```go
ctx, cfg := testutil.NewE2EContext(t)
// ctx.Repo, ctx.Branch, ctx.DebugBranch are populated from env vars
// cfg.Namespace and cfg.Timeout are available for polling helpers
```
