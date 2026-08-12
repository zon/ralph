# Testing

## Overview

Tests are split into two layers: unit and integration.
- **Unit** — individual functions and packages in isolation
- **Integration** — full request or execution paths end-to-end within the process

Examples below are Go; the rules are language-independent. Substitute your language's idiomatic assertion library, subtest mechanism, and temp-directory helper.

## Conventions

### Assertions

Use one assertion library across the whole suite, and use it for every assertion. In Go that is `github.com/stretchr/testify` (`assert` and `require`).

### Test structure

Use table-driven tests with named subtests (`t.Run()` in Go). Use a managed temporary directory for any filesystem interaction (`t.TempDir()` in Go) so cleanup is automatic.

### Isolation

Unit tests may run code with real side effects; integration tests must always use mocks.

A real dependency may only be invoked in a unit test when it is safe, local, and cheap. Version control and local CLI tools qualify — but only against an isolated temporary directory, **never** against the real repository. Create the directory with the temp-directory helper and initialise a fresh repo inside it before the test runs.

A real dependency must **never** be invoked in any test when it:

- mutates state outside the test (remote APIs, clusters, deployments, notifications)
- costs money per call
- requires credentials the suite does not own
- reaches the network for anything the test's correctness depends on

Each project lists its own banned dependencies in its agent instructions. These must always be abstracted behind interfaces and replaced with mocks.

External dependencies must be abstracted behind interfaces. Each interface has two implementations: a real one that calls the actual dependency, and a mock used in tests.

**Pattern:** define a minimal interface for each external dependency, accept it as a parameter in the function under test, and implement a mock struct in the test file. Use function fields so individual behaviors can be overridden per test case:

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

Unit tests for real external dependencies must be small, focused, and cheap — they exist only to verify the lowest-level interface of the real implementation, not to exercise full workflows.

- Test only the minimal surface needed to confirm the real dependency works (e.g. a single command round-trip).
- Use the shortest possible inputs. For a metered dependency such as an AI CLI, hard-code the cheapest available model and a trivial prompt like `"say hi"`.
- These tests must not accumulate: one or two cases per real dependency are enough.

### Module boundaries

Orchestration modules define interfaces and compose behavior — they must not contain real dependency implementations. Only implementation modules may contain real dependency implementations and mocks. See [Writing Code](code.md).

| Module type | May contain |
|---|---|
| Orchestration | interfaces, orchestration functions, tests |
| Implementation | real dependency implementations, mocks, tests |

A module's mocks must be importable without pulling in test infrastructure, so other packages can stub the dependency. In Go that means a `_mock.go` file in the module itself, not a `_test.go` file. Use whatever your language's equivalent of a non-test source file is.

### CLI command validation

Commands defined in specs must be tested to verify they exist in the expected format. Use the command's own help output to confirm the structure matches the specification:

```go
func TestCommand(t *testing.T) {
    cmd := exec.Command("go", "run", "./cmd/myapp", "subcommand", "name", "--help")
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
