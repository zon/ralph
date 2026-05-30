# Run Remote Orchestration

## Purpose

Verify the current branch is in sync with remote, submit an Argo Workflow, and optionally follow its logs and notify on completion.

## Orchestration

**Module:** `internal/orchestration/run`

```go
type GitClient interface {
    CurrentBranch() (string, error)
    IsBranchSyncedWithRemote(branch string) error
}

type WorkflowClient interface {
    Submit(proj *project.Project, cloneBranch string) (string, error)
    FollowLogs(workflowName string) error
    PrintLogHint(workflowName string)
}

type NotifyClient interface {
    Error(slug string)
    Success(slug string)
}

type RemoteRunner struct {
    git      GitClient
    workflow WorkflowClient
    notify   NotifyClient
}

func NewRemoteRunner(git GitClient, workflow WorkflowClient, notify NotifyClient) *RemoteRunner

func (r *RemoteRunner) RunRemote(proj *project.Project, follow bool) error {
    branch, err := r.git.CurrentBranch()
    if err != nil {
        return err
    }
    if err := r.git.IsBranchSyncedWithRemote(branch); err != nil {
        return err
    }
    workflowName, err := r.workflow.Submit(proj, branch)
    if err != nil {
        return err
    }
    if !follow {
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
- **`r.workflow.Submit(proj, cloneBranch)`** — generates an Argo Workflow for the project and submits it to the configured cluster; returns the workflow name on success
- **`r.workflow.PrintLogHint(workflowName)`** — prints the `argo logs` command the user can run to follow the workflow
- **`r.workflow.FollowLogs(workflowName)`** — streams the workflow logs and blocks until the workflow finishes; returns a non-nil error on workflow failure
- **`r.notify.Error(slug)`** — sends a desktop error notification for the given project slug when notifications are enabled
- **`r.notify.Success(slug)`** — sends a desktop success notification for the given project slug when notifications are enabled

## Tests

**Module:** `internal/orchestration/run`

```go
func TestRunRemoteBranchNotPushed(t *testing.T) {
    runner := remote.withMocks(
        remote.withGit(git.thatReportsBranchNotPushed()),
    )
    err := runner.RunRemote(project.any(), false)
    require.Error(t, err)
    require.False(t, workflow.submitted())
}

func TestRunRemoteBranchNotInSync(t *testing.T) {
    runner := remote.withMocks(
        remote.withGit(git.thatReportsBranchNotInSync()),
    )
    err := runner.RunRemote(project.any(), false)
    require.Error(t, err)
    require.False(t, workflow.submitted())
}

func TestRunRemoteWorkflowSubmissionFailure(t *testing.T) {
    runner := remote.withMocks(
        remote.withWorkflow(workflow.thatFailsOnSubmit()),
    )
    err := runner.RunRemote(project.any(), false)
    require.Error(t, err)
}

func TestRunRemoteNoFollowPrintsLogHint(t *testing.T) {
    runner := remote.withMocks()
    err := runner.RunRemote(project.any(), false)
    require.NoError(t, err)
    require.True(t, workflow.logHintPrinted())
    require.Empty(t, notify.successes())
}

func TestRunRemoteFollowSuccess(t *testing.T) {
    runner := remote.withMocks()
    err := runner.RunRemote(project.any(), true)
    require.NoError(t, err)
    require.NotEmpty(t, notify.successes())
}

func TestRunRemoteFollowFailureSendsErrorNotification(t *testing.T) {
    runner := remote.withMocks(
        remote.withWorkflow(workflow.thatFailsOnFollow()),
    )
    err := runner.RunRemote(project.any(), true)
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
}
```

### Helpers

- **`remote.withMocks(opts...)`** — constructs a `RemoteRunner` with default mock implementations; pass option helpers to override specific clients
- **`remote.withGit(client)`** — option that sets the git client on the mock runner
- **`remote.withWorkflow(client)`** — option that sets the workflow client on the mock runner
- **`project.any()`** — returns a valid project in a default state; owned by `internal/project`
- **`git.thatReportsBranchNotPushed()`** — returns a git client whose `IsBranchSyncedWithRemote` returns an error indicating the branch has no remote tracking ref
- **`git.thatReportsBranchNotInSync()`** — returns a git client whose `IsBranchSyncedWithRemote` returns an error indicating local and remote are at different commits
- **`workflow.submitted()`** — returns true when `Submit` was called during the test
- **`workflow.thatFailsOnSubmit()`** — returns a workflow client whose `Submit` returns an error
- **`workflow.logHintPrinted()`** — returns true when `PrintLogHint` was called during the test
- **`workflow.thatFailsOnFollow()`** — returns a workflow client whose `FollowLogs` returns an error
- **`notify.errors()`** — returns the list of error notifications sent during the test
- **`notify.successes()`** — returns the list of success notifications sent during the test
