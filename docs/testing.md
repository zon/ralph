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

Unit tests may run code with real side effects; integration tests must always use mocks. `git` and `opencode` may be invoked in unit tests, but only against an isolated temporary directory — never the real repository. Use `t.TempDir()` to create the directory and initialise a fresh repo before the test runs.

`beeep`, the GitHub API, the `gh` CLI, `argo`, and `kubectl` must never be invoked with real implementations in any test. These must always be abstracted behind interfaces and replaced with mocks.

External dependencies must be abstracted behind interfaces. Each interface has two implementations: a real one that calls the actual dependency, and a mock used in tests.

**Pattern:** define a minimal interface for each external dependency, accept it as a parameter in the function under test, and implement a mock struct in the `_test.go` file. Use function fields so individual behaviors can be overridden per test case:

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

The real implementation calls the actual dependency; tests pass a `*mockGit` instead.

### Dependency unit tests

Unit tests for real external dependencies (e.g. `opencode`, `git`) must be small, focused, and cheap — they exist only to verify the lowest-level interface of the real implementation, not to exercise full workflows.

- Test only the minimal surface needed to confirm the real dependency works (e.g. a single command round-trip).
- Use the shortest possible inputs. For `opencode`, hard-code an inexpensive model such as `deepseek v4 flash` and use a trivial prompt like `"say hi"`.
- These tests must not accumulate: one or two cases per real dependency are enough.

### Module boundaries

Orchestration modules define interfaces and compose behavior — they must not contain real dependency implementations. Only implementation modules may contain real dependency implementations and mocks.

| Module type | May contain |
|---|---|
| Orchestration | interfaces, orchestration functions, tests |
| Implementation | real dependency implementations, mocks, tests |

Mocks for an implementation module must live in a `_mock.go` file within that same module — not in `_test.go` files. This makes them importable by other packages that need to stub the dependency without pulling in test infrastructure.

### CLI command validation

Commands defined in specs must be tested to verify they exist in the expected format. Use `--help` to confirm the command structure matches the specification:

```go
func TestCommand(t *testing.T) {
    cmd := exec.Command("go", "run", "./cmd/ralph", "subcommand", "name", "--help")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)
    assert.Contains(t, string(output), "Expected help text from spec")
}
```

This catches structural issues such as:
- Incorrect command names (e.g. duplicated subcommand names)
- Missing subcommands
- Mismatched help text or flags

Run these tests as part of the standard test suite to ensure the CLI structure stays aligned with specifications.
