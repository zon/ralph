# Workflow Command Orchestration

## Purpose

`ralph workflow command`: set up the workspace and execute the supplied command tokens in the cloned repository.

## Orchestration

**Module:** `internal/orchestration/command`

```go
type WorkflowCommandCmd struct {
    workspace WorkspaceSetupClient
    exec      ExecClient
}

type WorkflowCommandFlags struct {
    Repo        string
    CloneBranch string
    BotName     string
    BotEmail    string
    Command     []string
}

func (w *WorkflowCommandCmd) Run(flags WorkflowCommandFlags) error {
    if len(flags.Command) == 0 {
        return ErrMissingCommand
    }
    if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
        return err
    }
    return w.exec.Run(flags.Command)
}
```

### Helpers

- **`flags.WorkspaceFlags()`** — returns a `WorkspaceFlags` with `Repo`, `CloneBranch`, `BotName`, and `BotEmail` from the command flags; `TargetBranch` is empty, `CreateBranch` is false, `Symlinks` is true
- **`w.workspace.Setup(flags)`** — delegates to the workspace setup orchestration defined in [workflow-workspace/orchestration.md](../workflow-workspace/orchestration.md)
- **`w.exec.Run(tokens)`** — executes the command tokens as a subprocess in the working directory; returns a non-nil error when the process exits with a non-zero code

## Tests

**Module:** `internal/orchestration/command`

```go
func TestRunMissingCommandAbortsBeforeWorkspace(t *testing.T) {
    cmd := command.withMocks()
    err := cmd.Run(flags.withNoCommand())
    require.Error(t, err)
    require.False(t, workspace.setupCalled())
}

func TestRunWorkspaceFailureAbortsEarly(t *testing.T) {
    cmd := command.withMocks(
        command.withWorkspace(workspace.thatFailsSetup()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, exec.runCalled())
}

func TestRunExecutesCommandAfterWorkspace(t *testing.T) {
    cmd := command.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, exec.runCalled())
}

func TestRunCommandFailureReturnsError(t *testing.T) {
    cmd := command.withMocks(
        command.withExec(exec.thatFails()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
}
```

### Helpers

- **`command.withMocks(opts...)`** — constructs a `WorkflowCommandCmd` with default mock implementations; pass option helpers to override specific clients
- **`command.withWorkspace(client)`** — option that sets the workspace setup client
- **`command.withExec(client)`** — option that sets the exec client
- **`flags.any()`** — returns a valid `WorkflowCommandFlags` with a non-empty command slice
- **`flags.withNoCommand()`** — returns `WorkflowCommandFlags` with an empty `Command` slice
- **`workspace.thatFailsSetup()`** — returns a workspace client whose `Setup` returns an error
- **`workspace.setupCalled()`** — returns true when `Setup` was called during the test
- **`exec.thatFails()`** — returns an exec client whose `Run` returns a non-zero exit error
- **`exec.runCalled()`** — returns true when `Run` was called during the test
