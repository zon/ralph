# Testing

## Overview

Tests are split into two layers: unit and integration.
- **Unit** — individual functions and packages in isolation
- **Integration** — full request or execution paths end-to-end within the process

## Conventions

### Assertions

Use `github.com/stretchr/testify` (`assert` and `require`) for all assertions.

### Test structure

Use table-driven tests with `t.Run()` subtests. Use `t.TempDir()` for any filesystem interaction — the testing package cleans it up automatically.

### Isolation

Tests must not invoke any external tools or services — no `git`, `gh`, `opencode`, or any other CLI or network call. External dependencies must be abstracted behind interfaces so tests can inject simple mock implementations.

**Pattern:** define a minimal interface for each external dependency, accept it as a parameter in the function under test, and implement a fake struct in the `_test.go` file. Use function fields so individual behaviors can be overridden per test case:

```go
type GitClient interface {
    CurrentBranch() (string, error)
    Push(branch string) error
}

type mockGit struct {
    currentBranchFn func() (string, error)
    pushFn          func(string) error
}

func (m *mockGit) CurrentBranch() (string, error) {
    if m.currentBranchFn != nil {
        return m.currentBranchFn()
    }
    return "main", nil
}

func (m *mockGit) Push(branch string) error {
    if m.pushFn != nil {
        return m.pushFn(branch)
    }
    return nil
}
```

The production implementation wraps the real CLI call; tests pass a `*mockGit` instead.

### Module boundaries

Orchestration modules define interfaces and compose behavior — they must not contain adapters or concrete implementations. Only implementation modules may contain adapters (the real CLI/network wrappers) and mocks (the test fakes).

| Module type | May contain |
|---|---|
| Orchestration | interfaces, orchestration functions, tests |
| Implementation | adapters, mocks, tests |

Mocks for an implementation module must live in a `_mock.go` file within that same module — not in `_test.go` files. This makes them importable by other packages that need to stub the dependency without pulling in test infrastructure.
