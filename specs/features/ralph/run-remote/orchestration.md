# Run Remote Orchestration

## Purpose

`ralph run` (default, without `--local`): verify the branch is in sync with remote, submit an Argo Workflow, and optionally stream its logs and notify on completion.

## Orchestration

**Module:** `internal/orchestration/run`

```go
type RemoteRunner struct {
    git      GitClient
    workflow WorkflowClient
    notify   NotifyClient
}

type RunRemoteFlags struct {
    Follow bool
    Debug  string
}

func (r *RemoteRunner) Run(proj *project.Project, flags RunRemoteFlags) error {
    branch, err := r.git.CurrentBranch()
    if err != nil {
        return err
    }
    if err := r.git.IsBranchSyncedWithRemote(branch); err != nil {
        return err
    }
    workflowName, err := r.workflow.Submit(proj, branch, flags.Debug)
    if err != nil {
        return err
    }
    if !flags.Follow {
        r.workflow.PrintLogHint(workflowName)
        return nil
    }
    if err := r.workflow.FollowLogs(workflowName); err != nil {
        r.notify.Error(proj.Slug)
        return err
    }
    r.notify.Success(proj.Slug)
    return nil
}
```

### Helpers

- **`r.git.CurrentBranch()`** — returns the name of the currently checked-out branch
- **`r.git.IsBranchSyncedWithRemote(branch)`** — returns an error when the branch has no remote tracking ref or when local and remote are at different commits
- **`r.workflow.Submit(proj, branch, debug)`** — generates and submits an Argo Workflow for the project cloned at `branch`; when `debug` is non-empty the workflow checks out that ralph source branch and invokes ralph via `go run`; returns the submitted workflow name
- **`r.workflow.PrintLogHint(workflowName)`** — prints the `argo logs` command the user can run to follow the workflow
- **`r.workflow.FollowLogs(workflowName)`** — streams the workflow logs and blocks until the workflow finishes; returns a non-nil error on workflow failure
- **`r.notify.Error(slug)`** — sends a desktop error notification for the given project slug when notifications are enabled
- **`r.notify.Success(slug)`** — sends a desktop success notification for the given project slug when notifications are enabled

## Tests

**Module:** `internal/orchestration/run`

```go
func TestRunBranchNotPushedAbortsBeforeSubmit(t *testing.T) {
    runner := remote.withMocks(
        remote.withGit(git.thatReportsBranchNotPushed()),
    )
    err := runner.Run(project.any(), flags.any())
    require.Error(t, err)
    require.False(t, workflow.submitCalled())
}

func TestRunBranchNotInSyncAbortsBeforeSubmit(t *testing.T) {
    runner := remote.withMocks(
        remote.withGit(git.thatReportsBranchNotInSync()),
    )
    err := runner.Run(project.any(), flags.any())
    require.Error(t, err)
    require.False(t, workflow.submitCalled())
}

func TestRunSubmitFailureReturnsError(t *testing.T) {
    runner := remote.withMocks(
        remote.withWorkflow(workflow.thatFailsSubmit()),
    )
    err := runner.Run(project.any(), flags.any())
    require.Error(t, err)
}

func TestRunNoFollowPrintsLogHint(t *testing.T) {
    runner := remote.withMocks()
    err := runner.Run(project.any(), flags.withoutFollow())
    require.NoError(t, err)
    require.True(t, workflow.logHintPrinted())
    require.False(t, workflow.followLogsCalled())
}

func TestRunFollowStreamsLogsAndNotifiesSuccess(t *testing.T) {
    runner := remote.withMocks()
    err := runner.Run(project.any(), flags.withFollow())
    require.NoError(t, err)
    require.True(t, workflow.followLogsCalled())
    require.True(t, notify.successSent())
}

func TestRunFollowFailureNotifiesErrorAndReturns(t *testing.T) {
    runner := remote.withMocks(
        remote.withWorkflow(workflow.thatFailsFollowLogs()),
    )
    err := runner.Run(project.any(), flags.withFollow())
    require.Error(t, err)
    require.True(t, notify.errorSent())
}

func TestRunDebugBranchPassedToSubmit(t *testing.T) {
    runner := remote.withMocks()
    err := runner.Run(project.any(), flags.withDebug("my-fix"))
    require.NoError(t, err)
    require.Equal(t, "my-fix", workflow.lastDebugBranch())
}
```

### Helpers

- **`remote.withMocks(opts...)`** — constructs a `RemoteRunner` with default mock implementations; pass option helpers to override specific clients
- **`remote.withGit(client)`** — option that sets the git client
- **`remote.withWorkflow(client)`** — option that sets the workflow client
- **`flags.any()`** — returns `RunRemoteFlags` with `Follow` false and no debug branch
- **`flags.withFollow()`** — returns `RunRemoteFlags` with `Follow` true
- **`flags.withoutFollow()`** — returns `RunRemoteFlags` with `Follow` false
- **`flags.withDebug(branch)`** — returns `RunRemoteFlags` with `Debug` set to the given branch
- **`git.thatReportsBranchNotPushed()`** — returns a git client whose `IsBranchSyncedWithRemote` returns an error indicating no remote tracking ref
- **`git.thatReportsBranchNotInSync()`** — returns a git client whose `IsBranchSyncedWithRemote` returns an error indicating local and remote are at different commits
- **`workflow.thatFailsSubmit()`** — returns a workflow client whose `Submit` returns an error
- **`workflow.thatFailsFollowLogs()`** — returns a workflow client whose `FollowLogs` returns an error
- **`workflow.submitCalled()`** — returns true when `Submit` was called during the test
- **`workflow.followLogsCalled()`** — returns true when `FollowLogs` was called during the test
- **`workflow.logHintPrinted()`** — returns true when `PrintLogHint` was called during the test
- **`workflow.lastDebugBranch()`** — returns the debug branch passed to the most recent `Submit` call
- **`notify.successSent()`** — returns true when `Success` was called during the test
- **`notify.errorSent()`** — returns true when `Error` was called during the test
