# Command Orchestration

## Purpose

`ralph command`: validate the command, submit an Argo Workflow, and stream its logs.

## Orchestration

**Module:** `internal/orchestration/command`

```go
type CommandCmd struct {
    workflow WorkflowClient
}

type CommandFlags struct {
    Command  []string
    NoFollow bool
}

func (c *CommandCmd) Run(flags CommandFlags) error {
    if len(flags.Command) == 0 {
        return ErrMissingCommand
    }
    if err := c.workflow.Submit(flags.Command); err != nil {
        return err
    }
    if flags.NoFollow {
        return nil
    }
    return c.workflow.StreamLogs()
}
```

### Helpers

- **`c.workflow.Submit(command)`** — generates and submits an Argo Workflow embedding the command tokens; loads `.ralph/config.yaml` internally to resolve the workflow image, configmaps, secrets, and namespace
- **`c.workflow.StreamLogs()`** — streams the submitted workflow's logs to stdout until the workflow completes; returns a non-nil error when the workflow fails

## Tests

**Module:** `internal/orchestration/command`

```go
func TestRunMissingCommandReturnsError(t *testing.T) {
    cmd := command.withMocks()
    err := cmd.Run(flags.withNoCommand())
    require.Error(t, err)
    require.False(t, workflow.submitCalled())
}

func TestRunSubmitsWorkflow(t *testing.T) {
    cmd := command.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, workflow.submitCalled())
}

func TestRunSubmitFailureReturnsError(t *testing.T) {
    cmd := command.withMocks(
        command.withWorkflow(workflow.thatFailsSubmit()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, workflow.streamLogsCalled())
}

func TestRunStreamsLogsByDefault(t *testing.T) {
    cmd := command.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, workflow.streamLogsCalled())
}

func TestRunNoFollowSkipsLogStreaming(t *testing.T) {
    cmd := command.withMocks()
    err := cmd.Run(flags.withNoFollow())
    require.NoError(t, err)
    require.True(t, workflow.submitCalled())
    require.False(t, workflow.streamLogsCalled())
}

func TestRunWorkflowFailurePropagatesError(t *testing.T) {
    cmd := command.withMocks(
        command.withWorkflow(workflow.thatFailsStreaming()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
}
```

### Helpers

- **`command.withMocks(opts...)`** — constructs a `CommandCmd` with default mock implementations; pass option helpers to override specific clients
- **`command.withWorkflow(client)`** — option that sets the workflow client
- **`flags.any()`** — returns a valid `CommandFlags` with a non-empty command slice
- **`flags.withNoCommand()`** — returns `CommandFlags` with an empty `Command` slice
- **`flags.withNoFollow()`** — returns `CommandFlags` with `NoFollow` true
- **`workflow.thatFailsSubmit()`** — returns a workflow client whose `Submit` returns an error
- **`workflow.thatFailsStreaming()`** — returns a workflow client whose `StreamLogs` returns an error
- **`workflow.submitCalled()`** — returns true when `Submit` was called during the test
- **`workflow.streamLogsCalled()`** — returns true when `StreamLogs` was called during the test
