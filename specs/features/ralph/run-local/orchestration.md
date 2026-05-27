# Run Local Orchestration

## Purpose

Run the full development loop in-process: set up the environment, iterate until all requirements pass, then open a pull request and notify.

## Orchestration

**Module:** `internal/run`

```go
type Runner struct {
    project  ProjectClient
    ai       AgentClient
    git      GitClient
    github   GitHubClient
    services ServicesClient
    notify   NotifyClient
}

func (r *Runner) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
    if err := r.services.RunBeforeCommands(cfg); err != nil {
        return err
    }
    if err := r.git.SwitchToBranch(proj.Slug); err != nil {
        return err
    }
    if err := r.iterate(proj); err != nil {
        r.notify.Error(proj.Slug)
        return err
    }
    if err := r.github.CreatePR(proj); err != nil {
        r.notify.Error(proj.Slug)
        return err
    }
    r.notify.Success(proj.Slug)
    return nil
}
```

### Helpers

- **`r.services.RunBeforeCommands(cfg)`** — runs each `before` command from the ralph config sequentially; aborts on the first non-zero exit
- **`r.git.SwitchToBranch(slug)`** — switches to the branch named by the project slug, creating it if it does not exist
- **`r.iterate(proj)`** — drives the iteration loop; returns nil only when all requirements are passing, or a non-nil error when blocked, when a fatal AI error occurs, or when max iterations is reached with requirements still failing
- **`r.github.CreatePR(proj)`** — generates an AI PR summary and opens a pull request from the project branch to the base branch; is a no-op when no commits exist ahead of the base branch
- **`r.notify.Error(slug)`** — sends a desktop error notification for the given project slug when notifications are enabled
- **`r.notify.Success(slug)`** — sends a desktop success notification for the given project slug when notifications are enabled

---

```go
func (r *Runner) iterate(proj *project.Project) error {
    for i := 0; i < proj.MaxIterations; i++ {
        if r.project.AllRequirementsPassing(proj) {
            return nil
        }
        if r.git.BlockedFileExists() {
            return ErrBlocked
        }
        if err := r.ai.Iterate(proj); err != nil {
            if r.ai.IsFatal(err) {
                return err
            }
            r.git.WriteBlockedFile(err)
            return err
        }
        if err := r.commitIteration(proj); err != nil {
            return err
        }
    }
    return r.project.MaxIterationsError(proj)
}
```

### Helpers

- **`r.project.AllRequirementsPassing(proj)`** — returns true when every requirement in the project carries `passing: true`
- **`r.git.BlockedFileExists()`** — returns true when `blocked.md` is present in the repository root
- **`r.ai.Iterate(proj)`** — invokes the AI agent for one iteration of the project; returns a non-nil error on failure
- **`r.ai.IsFatal(err)`** — returns true when the error is a billing or quota condition that must not be retried
- **`r.git.WriteBlockedFile(err)`** — writes `blocked.md` to the repository root containing the failure reason
- **`r.project.MaxIterationsError(proj)`** — returns an error naming the count of still-failing requirements

---

```go
func (r *Runner) commitIteration(proj *project.Project) error {
    if !r.git.HasChanges() {
        return nil
    }
    if !r.git.ReportExists() {
        if err := r.ai.GenerateChangelog(proj); err != nil {
            return err
        }
    }
    return r.git.CommitFromReport(proj.Slug)
}
```

### Helpers

- **`r.git.HasChanges()`** — returns true when the working tree has uncommitted changes
- **`r.git.ReportExists()`** — returns true when `report.md` is present in the repository root
- **`r.ai.GenerateChangelog(proj)`** — invokes the AI agent to produce a changelog and write it to `report.md`
- **`r.git.CommitFromReport(slug)`** — stages all changes, uses `report.md` as the commit message, commits, then deletes `report.md`

## Tests

**Module:** `internal/run`

```go
func TestRunLocalBeforeCommandFailureAbortsEarly(t *testing.T) {
    runner := run.withMocks(
        run.withServices(services.thatFailBeforeCommands()),
    )
    err := runner.RunLocal(project.any(), config.any())
    require.Error(t, err)
    require.False(t, git.branchSwitched())
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatAlwaysFails()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
}

func TestRunLocalAllRequirementsPassCreatesPR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
        run.withGit(git.withCommitsAhead()),
    )
    err := runner.RunLocal(project.withAllPassing(), config.any())
    require.NoError(t, err)
    require.True(t, github.prCreated())
    require.NotEmpty(t, notify.successes())
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(project.withAllPassing(), config.any())
    require.NoError(t, err)
    require.False(t, github.prCreated())
    require.NotEmpty(t, notify.successes())
}

func TestIterateExitsImmediatelyWhenAllPassing(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(project.withAllPassing(), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.iterateCalls())
}

func TestIterateExitsEarlyWhenRequirementsPass(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(2)),
        run.withProject(project.withMaxIterations(10)),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.NoError(t, err)
    require.Len(t, ai.iterateCalls(), 2)
}

func TestIterateReturnsErrorAtMaxIterations(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsFailures()),
        run.withProject(project.withMaxIterations(3)),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.Error(t, err)
    require.Len(t, ai.iterateCalls(), 3)
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withGit(git.withBlockedFile()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.ErrorIs(t, err, run.ErrBlocked)
    require.Empty(t, ai.iterateCalls())
}

func TestIterateFatalAIErrorIsNotRetried(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsFatalError()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.Error(t, err)
    require.Len(t, ai.iterateCalls(), 1)
    require.False(t, git.blockedFileWritten())
}

func TestIterateNonFatalAIErrorWritesBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsNonFatalError()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.Error(t, err)
    require.True(t, git.blockedFileWritten())
}

func TestCommitIterationUsesReportWhenPresent(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
        run.withGit(git.withChangesAndReport()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.changelogCalls())
    require.True(t, git.committedFromReport())
}

func TestCommitIterationGeneratesChangelogWhenNoReport(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
        run.withGit(git.withChangesButNoReport()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.NoError(t, err)
    require.Len(t, ai.changelogCalls(), 1)
    require.True(t, git.committedFromReport())
}

func TestCommitIterationSkipsCommitWhenNoChanges(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
        run.withGit(git.withNoChanges()),
    )
    err := runner.RunLocal(project.withFailingRequirements(), config.any())
    require.NoError(t, err)
    require.False(t, git.committedFromReport())
}
```

### Helpers

- **`run.withMocks(opts...)`** — constructs a `Runner` with default mock implementations; pass option helpers to override specific clients
- **`run.withServices(client)`** — option that sets the services client on the mock runner
- **`run.withAI(client)`** — option that sets the AI client on the mock runner
- **`run.withProject(client)`** — option that sets the project client on the mock runner
- **`run.withGit(client)`** — option that sets the git client on the mock runner
- **`project.any()`** — returns a valid project in a default state; owned by `internal/project`
- **`project.withAllPassing()`** — returns a project where every requirement has `passing: true`; owned by `internal/project`
- **`project.withFailingRequirements()`** — returns a project with at least one failing requirement; owned by `internal/project`
- **`project.withMaxIterations(n)`** — returns a project whose `MaxIterations` is set to `n`; owned by `internal/project`
- **`project.thatReportsAllPassing()`** — returns a project client whose `AllRequirementsPassing` always returns true
- **`project.thatReportsPassingAfterIterations(n)`** — returns a project client whose `AllRequirementsPassing` returns false for the first `n` calls and true thereafter
- **`project.thatAlwaysReportsFailures()`** — returns a project client whose `AllRequirementsPassing` always returns false
- **`config.any()`** — returns a valid ralph config in a default state; owned by `internal/config`
- **`services.thatFailBeforeCommands()`** — returns a services client whose `RunBeforeCommands` returns an error
- **`ai.thatAlwaysFails()`** — returns an AI client whose `Iterate` always returns a non-fatal error
- **`ai.thatReturnsFatalError()`** — returns an AI client whose `Iterate` returns a billing or quota error
- **`ai.thatReturnsNonFatalError()`** — returns an AI client whose `Iterate` returns a non-fatal error
- **`ai.iterateCalls()`** — returns the list of projects passed to `Iterate` during the test
- **`ai.changelogCalls()`** — returns the list of projects passed to `GenerateChangelog` during the test
- **`git.withCommitsAhead()`** — returns a git client that reports commits ahead of the base branch
- **`git.withBlockedFile()`** — returns a git client that reports `blocked.md` as present
- **`git.withChangesAndReport()`** — returns a git client that reports uncommitted changes and a present `report.md`
- **`git.withChangesButNoReport()`** — returns a git client that reports uncommitted changes and no `report.md`
- **`git.withNoChanges()`** — returns a git client that reports a clean working tree
- **`git.branchSwitched()`** — returns true when `SwitchToBranch` was called during the test
- **`git.blockedFileWritten()`** — returns true when `WriteBlockedFile` was called during the test
- **`git.committedFromReport()`** — returns true when `CommitFromReport` was called during the test
- **`github.prCreated()`** — returns true when `CreatePR` was called and produced a pull request
- **`notify.errors()`** — returns the list of error notifications sent during the test
- **`notify.successes()`** — returns the list of success notifications sent during the test
