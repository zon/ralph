# Testing

## Strategy

Tests are split into unit tests and integration tests. Unit tests cover individual functions and packages in isolation. Integration tests exercise full request or execution paths end-to-end within the process.

## Assertions

Use `github.com/stretchr/testify` (`assert` and `require`) for all assertions.

## External Dependencies

Tests must not invoke any external tools or services — no `git`, `gh`, `opencode`, or any other CLI or network call. Any function that calls an external dependency must support a dry-run mode. In dry-run mode the function skips the real call and returns inspectable state describing what it would have done. Tests enable dry-run and assert on that state.

## Structure

Use table-driven tests with `t.Run()` subtests. Use `t.TempDir()` for any file system interaction — the testing package cleans it up automatically.
