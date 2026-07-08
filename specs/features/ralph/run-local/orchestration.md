# Run Local Orchestration

## Purpose

Run the full development loop in-process: set up the environment, generate any missing artifacts, iterate until all requirements pass, then open a pull request and notify.

## Orchestration

**Module:** `internal/orchestration/run`

```go
type Runner struct {
    project  ProjectClient
    ai       AIClient
    git      GitClient
    github   GitHubClient
    services ServicesClient
    notify   NotifyClient
    env      EnvClient
    cmd      CmdClient
    remote   RemoteClient
}

func (r *Runner) RunLocal(input *InputFile, cfg *config.RalphConfig) error {
    if r.env.InWorkflow() {
        defer r.ai.PrintStats()
    }
    if err := r.services.RunBeforeCommands(cfg); err != nil {
        return err
    }
    if err := r.git.SwitchToBranch(input.Slug()); err != nil {
        return err
    }
    proj, err := r.generateArtifacts(input)
    if err != nil {
        r.notify.Error(input.Slug())
        return err
    }
    if err := r.iterate(proj, cfg); err != nil {
        r.notify.Error(proj.Slug)
        return err
    }
    if err := r.removeOrchestration(proj); err != nil {
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

- **`r.env.InWorkflow()`** — returns true when ralph is running inside an Argo workflow container
- **`r.ai.PrintStats()`** — prints input tokens, output tokens, and total cost for the run; called via `defer` so it always runs regardless of outcome
- **`r.services.RunBeforeCommands(cfg)`** — runs each `before` command from the ralph config sequentially; aborts on the first non-zero exit
- **`input.Slug()`** — returns the slug for the input: the project slug when the input is a project file, otherwise a slug derived from the input file path
- **`r.git.SwitchToBranch(slug)`** — switches to the branch named by the slug, creating it if it does not exist
- **`r.generateArtifacts(input)`** — generates any missing artifacts for orchestration or spec inputs and commits them; returns the project to run
- **`r.iterate(proj, cfg)`** — drives the iteration loop; returns nil only when all requirements are passing, or a non-nil error when blocked, when a fatal AI error occurs, or when the iteration limit is reached with requirements still failing
- **`r.removeOrchestration(proj)`** — checks whether the project's spec contains an orchestration document and, if so, deletes it and commits the deletion
- **`r.github.CreatePR(proj)`** — generates an AI PR summary and opens a pull request from the project branch to the base branch; is a no-op when no commits exist ahead of the base branch
- **`r.notify.Error(slug)`** — sends a desktop error notification for the given project slug when notifications are enabled
- **`r.notify.Success(slug)`** — sends a desktop success notification for the given project slug when notifications are enabled

---

```go
func (r *Runner) generateArtifacts(input *InputFile) (*project.Project, error) {
    if input.IsProject() {
        return input.Project(), nil
    }
    if input.IsSpec() {
        if err := r.ai.WriteOrchestration(input); err != nil {
            return nil, err
        }
    }
    proj, err := r.ai.WriteProject(input)
    if err != nil {
        return nil, err
    }
    return proj, r.git.CommitGeneratedArtifacts(proj)
}
```

### Helpers

- **`input.IsProject()`** — returns true when the input file is a project YAML
- **`input.IsSpec()`** — returns true when the input file is a `spec.md`
- **`input.Project()`** — returns the project loaded from the input file; only valid when `IsProject()` is true
- **`r.ai.WriteOrchestration(input)`** — invokes the AI agent to generate an `orchestration.md` file in the same directory as the spec and writes it to disk
- **`r.ai.WriteProject(input)`** — invokes the AI agent to generate a project YAML file in `projects/` based on the input, writes it to disk, and returns the loaded project; for spec inputs, reads both the spec and the orchestration from disk
- **`r.git.CommitGeneratedArtifacts(proj)`** — stages and commits all generated files (the project YAML and any orchestration document) with a fixed message

---

```go
func (r *Runner) iterate(proj *project.Project, cfg *config.RalphConfig) error {
    extra := r.project.ExtraIterations(proj, cfg)
    limit := len(proj.Requirements) + extra
    for i := 0; i < limit; i++ {
        proj = r.project.Load(proj)
        if r.project.AllRequirementsPassing(proj) {
            return nil
        }
        if r.git.BlockedFileExists() {
            return ErrBlocked
        }
        if err := r.runIteration(proj, cfg); err != nil {
            return err
        }
        if err := r.commitIteration(proj); err != nil {
            return err
        }
    }
    return r.project.ExtraIterationsError(proj)
}

func (r *Runner) blockAndReturn(err error) error {
    if !r.ai.IsFatal(err) {
        r.git.WriteBlockedFile(err)
    }
    return err
}
```

### Helpers

- **`r.project.Load(proj)`** — reloads the project from disk, returning the latest state; falls back to the in-memory project if the file cannot be read
- **`r.project.AllRequirementsPassing(proj)`** — returns true when every requirement in the project carries `passing: true`
- **`r.git.BlockedFileExists()`** — returns true when `blocked.md` is present in the repository root
- **`r.runIteration(proj, cfg)`** — starts services, runs the picker and development agents, stops services, and removes service logs
- **`r.project.ExtraIterations(proj, cfg)`** — returns the configured extra iteration count from config or flag, or 20% of the project's requirement count (rounded up) when unset
- **`r.project.ExtraIterationsError(proj)`** — returns an error naming the count of still-failing requirements

---

```go
func (r *Runner) runIteration(proj *project.Project, cfg *config.RalphConfig) error {
    svc, err := r.services.Start(cfg)
    if err != nil {
        if fixErr := r.ai.FixServiceStartup(cfg, err); fixErr != nil {
            return fixErr
        }
        svc = nil
    }
    defer r.services.Stop(svc)
    defer r.services.RemoveLogs(cfg)
    req, err := r.ai.RunPicker(proj)
    if err != nil {
        return r.blockAndReturn(err)
    }
    if err := r.ai.RunDeveloper(proj, req); err != nil {
        return r.blockAndReturn(err)
    }
    return r.cleanup(proj)
}
```

### Helpers

- **`r.services.Start(cfg)`** — starts all services declared in `.ralph/config.yaml`; returns the service manager and any startup error
- **`r.ai.FixServiceStartup(cfg, err)`** — invokes the development agent with a diagnosis prompt for the failed service; returns nil when the fix succeeds
- **`r.services.Stop(svc)`** — stops all running services; no-op when `svc` is nil
- **`r.services.RemoveLogs(cfg)`** — deletes log files produced by each configured service
- **`r.ai.RunPicker(proj)`** — builds a picker prompt from project content and the recent commit log, invokes the picker agent with the resolved model and variant, reads `picked-requirement.yaml`, and returns its YAML content
- **`r.ai.RunDeveloper(proj, req)`** — builds a development prompt with project content and the selected requirement, then invokes the development agent with the resolved model and variant
- **`r.ai.IsFatal(err)`** — returns true when the error is a billing or quota condition that must not be retried

Both `RunPicker` and `RunDeveloper` resolve the variant from the execution context using two-level precedence: `--variant` at the command line takes priority; otherwise the top-level `variant` field in `.ralph/config.yaml` is used. When both are unset, `--variant` is omitted entirely from the opencode invocation (unlike model, which always has a default).
- **`r.git.WriteBlockedFile(err)`** — writes `blocked.md` to the repository root containing the failure reason
- **`r.cleanup(proj)`** — normalizes trailing newlines in the project file and stages it if changed

---

```go
func (r *Runner) cleanup(proj *project.Project) error {
    if r.project.HasChanges(proj) {
        r.project.NormalizeAndStage(proj)
    }
    return nil
}
```

### Helpers

- **`r.project.HasChanges(proj)`** — returns true when the project file has uncommitted changes relative to the index
- **`r.project.NormalizeAndStage(proj)`** — strips excess trailing newlines from the project file and stages it

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

---

```go
func (r *Runner) removeOrchestration(proj *project.Project) error {
    if !r.project.HasSpec(proj) {
        return nil
    }
    if !r.project.HasOrchestration(proj) {
        return nil
    }
    if err := r.project.RemoveOrchestration(proj); err != nil {
        return err
    }
    return r.git.CommitOrchestrationRemoval(proj.Slug)
}
```

### Helpers

- **`r.project.HasSpec(proj)`** — returns true when the project references a spec path
- **`r.project.HasOrchestration(proj)`** — returns true when an `orchestration.md` file exists inside the project's spec directory
- **`r.project.RemoveOrchestration(proj)`** — deletes the `orchestration.md` file from the project's spec directory and stages the deletion
- **`r.git.CommitOrchestrationRemoval(slug)`** — commits the staged orchestration deletion with a fixed message

## Tests

**Module:** `internal/orchestration/run`

```go
func TestRunLocalStatsPrintedOnSuccess(t *testing.T) {
    runner := run.withMocks(
        run.withEnv(env.inWorkflow()),
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.True(t, ai.statsPrinted())
}

func TestRunLocalStatsPrintedOnFailure(t *testing.T) {
    runner := run.withMocks(
        run.withEnv(env.inWorkflow()),
        run.withAI(ai.thatAlwaysFails()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.True(t, ai.statsPrinted())
}

func TestRunLocalStatsNotPrintedWhenNotInWorkflow(t *testing.T) {
    runner := run.withMocks(
        run.withEnv(env.notInWorkflow()),
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.False(t, ai.statsPrinted())
}

func TestRunLocalBeforeCommandFailureAbortsEarly(t *testing.T) {
    runner := run.withMocks(
        run.withServices(services.thatFailBeforeCommands()),
    )
    err := runner.RunLocal(input.forProject(project.any()), config.any())
    require.Error(t, err)
    require.False(t, git.branchSwitched())
}

func TestRunLocalProjectInputSkipsGeneration(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.False(t, ai.writeProjectCalled())
    require.False(t, git.artifactsCommitted())
}

func TestRunLocalOrchestrationInputGeneratesAndCommitsProject(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forOrchestration(), config.any())
    require.NoError(t, err)
    require.False(t, ai.writeOrchestrationCalled())
    require.True(t, ai.writeProjectCalled())
    require.True(t, git.artifactsCommitted())
}

func TestRunLocalSpecInputGeneratesOrchestrationThenProject(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forSpec(), config.any())
    require.NoError(t, err)
    require.True(t, ai.writeOrchestrationCalled())
    require.True(t, ai.writeProjectCalled())
    require.True(t, git.artifactsCommitted())
}

func TestRunLocalOrchestrationWriteProjectFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatFailsWriteProject()),
    )
    err := runner.RunLocal(input.forOrchestration(), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
    require.Empty(t, ai.pickCalls())
}

func TestRunLocalSpecWriteOrchestrationFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatFailsWriteOrchestration()),
    )
    err := runner.RunLocal(input.forSpec(), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
    require.False(t, ai.writeProjectCalled())
    require.Empty(t, ai.pickCalls())
}

func TestRunLocalSpecWriteProjectFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatFailsWriteProject()),
    )
    err := runner.RunLocal(input.forSpec(), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
    require.Empty(t, ai.pickCalls())
}

func TestRunLocalGenerationHappensAfterBranchSwitch(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
        run.withGit(git.thatRecordsCallOrder()),
    )
    err := runner.RunLocal(input.forOrchestration(), config.any())
    require.NoError(t, err)
    require.True(t, git.switchedBeforeArtifactsCommitted())
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatAlwaysFails()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
}

func TestRunLocalAllRequirementsPassCreatesPR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
        run.withGit(git.withCommitsAhead()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.True(t, github.prCreated())
    require.NotEmpty(t, notify.successes())
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.False(t, github.prCreated())
    require.NotEmpty(t, notify.successes())
}

func TestIterateExitsImmediatelyWhenAllPassing(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.pickCalls())
}

func TestIterateExitsEarlyWhenRequirementsPass(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(2)),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.Len(t, ai.pickCalls(), 2)
    require.Len(t, ai.developCalls(), 2)
}

func TestIterateReturnsErrorWhenLimitReached(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsFailures()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements(1)), config.withExtraIterations(0))
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 1)
}

func TestIterateRespectsExtraIterations(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsFailures()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements(3)), config.withExtraIterations(2))
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 5)
}

func TestIterateDefaultsToTwentyPercentExtra(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsFailures()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements(10)), config.any())
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 12)
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withGit(git.withBlockedFile()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.ErrorIs(t, err, run.ErrBlocked)
    require.Empty(t, ai.pickCalls())
}

func TestIterateFatalPickErrorIsNotRetried(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsFatalPickError()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 1)
    require.Empty(t, ai.developCalls())
    require.False(t, git.blockedFileWritten())
}

func TestIterateNonFatalPickErrorWritesBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsNonFatalPickError()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.True(t, git.blockedFileWritten())
}

func TestIterateFatalDevelopErrorIsNotRetried(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsFatalDevelopError()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.Len(t, ai.developCalls(), 1)
    require.False(t, git.blockedFileWritten())
}

func TestIterateNonFatalDevelopErrorWritesBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsNonFatalDevelopError()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.True(t, git.blockedFileWritten())
}

func TestRunIterationStartsAndStopsServicesEachIteration(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(2)),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.Equal(t, 2, services.startCount())
    require.Equal(t, 2, services.stopCount())
    require.Equal(t, 2, services.removeLogsCount())
}

func TestRunIterationServiceStartupFailureTriggersFix(t *testing.T) {
    runner := run.withMocks(
        run.withServices(services.thatFailToStart()),
        run.withProject(project.thatReportsPassingAfterIterations(1)),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.True(t, ai.serviceFixCalled())
    require.Len(t, ai.pickCalls(), 1)
}

func TestRunIterationServiceFixFailureReturnsError(t *testing.T) {
    runner := run.withMocks(
        run.withServices(services.thatFailToStart()),
        run.withAI(ai.thatFailsServiceFix()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.Error(t, err)
    require.Empty(t, ai.pickCalls())
}

func TestCleanupNormalizesProjectFileWhenChanged(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1).withChanges()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.True(t, project.normalizedAndStaged())
}

func TestCleanupSkipsNormalizationWhenNoChanges(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1).withNoChanges()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.False(t, project.normalizedAndStaged())
}

func TestCommitIterationUsesReportWhenPresent(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
        run.withGit(git.withChangesAndReport()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.changelogCalls())
    require.True(t, git.committedFromReport())
}

func TestCommitIterationGeneratesChangelogWhenNoReport(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
        run.withGit(git.withChangesButNoReport()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.Len(t, ai.changelogCalls(), 1)
    require.True(t, git.committedFromReport())
}

func TestCommitIterationSkipsCommitWhenNoChanges(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
        run.withGit(git.withNoChanges()),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.False(t, git.committedFromReport())
}

func TestRunIterationPassesConfigVariantToAI(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.withVariant("high"))
    require.NoError(t, err)
    require.Equal(t, "high", ai.lastVariant())
}

func TestRunIterationOmitsVariantWhenUnset(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsPassingAfterIterations(1)),
    )
    err := runner.RunLocal(input.forProject(project.withFailingRequirements()), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.lastVariant())
}

func TestRemoveOrchestrationSkipsWhenNoSpec(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing().withNoSpec()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.False(t, git.orchestrationRemovalCommitted())
}

func TestRemoveOrchestrationSkipsWhenNoOrchestration(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing().withSpecButNoOrchestration()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.False(t, git.orchestrationRemovalCommitted())
}

func TestRemoveOrchestrationRemovesAndCommitsWhenPresent(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing().withOrchestration()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.NoError(t, err)
    require.True(t, project.orchestrationRemoved())
    require.True(t, git.orchestrationRemovalCommitted())
}

func TestRemoveOrchestrationFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllPassing().withOrchestration().thatFailsRemoval()),
    )
    err := runner.RunLocal(input.forProject(project.withAllPassing()), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
    require.False(t, github.prCreated())
}
```

### Helpers

- **`run.withMocks(opts...)`** — constructs a `Runner` with default mock implementations; pass option helpers to override specific clients
- **`run.withEnv(client)`** — option that sets the env client on the mock runner
- **`run.withServices(client)`** — option that sets the services client on the mock runner
- **`run.withAI(client)`** — option that sets the AI client on the mock runner
- **`run.withProject(client)`** — option that sets the project client on the mock runner
- **`run.withGit(client)`** — option that sets the git client on the mock runner
- **`input.forProject(p)`** — returns an `InputFile` wrapping the given project; `IsProject()` returns true and `Slug()` returns the project slug; owned by `internal/project`
- **`input.forOrchestration()`** — returns an `InputFile` representing an orchestration document; `IsProject()` returns false, `IsSpec()` returns false; owned by `internal/project`
- **`input.forSpec()`** — returns an `InputFile` representing a spec document; `IsProject()` returns false, `IsSpec()` returns true; owned by `internal/project`
- **`project.any()`** — returns a valid project in a default state; owned by `internal/project`
- **`project.withAllPassing()`** — returns a project where every requirement has `passing: true`; owned by `internal/project`
- **`project.withFailingRequirements()`** — returns a project with at least one failing requirement; owned by `internal/project`
- **`project.withFailingRequirements(n)`** — returns a project with exactly `n` failing requirements; owned by `internal/project`
- **`project.thatReportsAllPassing()`** — returns a project client whose `Load` and `AllRequirementsPassing` always reflect all requirements passing
- **`project.thatReportsAllPassing().withNoSpec()`** — chains a modifier so `HasSpec` returns false
- **`project.thatReportsAllPassing().withSpecButNoOrchestration()`** — chains a modifier so `HasSpec` returns true and `HasOrchestration` returns false
- **`project.thatReportsAllPassing().withOrchestration()`** — chains a modifier so `HasSpec` and `HasOrchestration` both return true
- **`project.thatReportsAllPassing().withOrchestration().thatFailsRemoval()`** — chains a modifier so `RemoveOrchestration` returns an error
- **`project.thatReportsPassingAfterIterations(n)`** — returns a project client whose `AllRequirementsPassing` returns false for the first `n` calls and true thereafter
- **`project.thatAlwaysReportsFailures()`** — returns a project client whose `AllRequirementsPassing` always returns false
- **`project.orchestrationRemoved()`** — returns true when `RemoveOrchestration` was called during the test
- **`config.any()`** — returns a valid ralph config in a default state; owned by `internal/config`
- **`config.withExtraIterations(n)`** — returns a config whose `ExtraIterations` field is set to `n`; owned by `internal/config`
- **`config.withVariant(v)`** — returns a config whose `Variant` field is set to `v`; owned by `internal/config`
- **`env.inWorkflow()`** — returns an env client that reports `InWorkflow() = true`
- **`env.notInWorkflow()`** — returns an env client that reports `InWorkflow() = false`
- **`services.thatFailBeforeCommands()`** — returns a services client whose `RunBeforeCommands` returns an error
- **`services.thatFailToStart()`** — returns a services client whose `Start` returns an error
- **`services.startCount()`** — returns the number of times `Start` was called during the test
- **`services.stopCount()`** — returns the number of times `Stop` was called during the test
- **`services.removeLogsCount()`** — returns the number of times `RemoveLogs` was called during the test
- **`ai.statsPrinted()`** — returns true when `PrintStats` was called during the test
- **`ai.lastVariant()`** — returns the variant resolved during the most recent AI invocation, or empty string when `--variant` was omitted
- **`ai.thatAlwaysFails()`** — returns an AI client whose `RunPicker` always returns a non-fatal error
- **`ai.thatFailsServiceFix()`** — returns an AI client whose `FixServiceStartup` returns an error
- **`ai.thatFailsWriteOrchestration()`** — returns an AI client whose `WriteOrchestration` returns an error
- **`ai.thatFailsWriteProject()`** — returns an AI client whose `WriteProject` returns an error
- **`ai.serviceFixCalled()`** — returns true when `FixServiceStartup` was called during the test
- **`ai.writeOrchestrationCalled()`** — returns true when `WriteOrchestration` was called during the test
- **`ai.writeProjectCalled()`** — returns true when `WriteProject` was called during the test
- **`ai.thatReturnsFatalPickError()`** — returns an AI client whose `RunPicker` returns a billing or quota error
- **`ai.thatReturnsNonFatalPickError()`** — returns an AI client whose `RunPicker` returns a non-fatal error
- **`ai.thatReturnsFatalDevelopError()`** — returns an AI client whose `RunDeveloper` returns a billing or quota error
- **`ai.thatReturnsNonFatalDevelopError()`** — returns an AI client whose `RunDeveloper` returns a non-fatal error
- **`ai.pickCalls()`** — returns the list of projects passed to `RunPicker` during the test
- **`ai.developCalls()`** — returns the list of projects passed to `RunDeveloper` during the test
- **`ai.changelogCalls()`** — returns the list of projects passed to `GenerateChangelog` during the test
- **`project.thatReportsPassingAfterIterations(n).withChanges()`** — chains a modifier so `HasChanges` returns true during that iteration
- **`project.thatReportsPassingAfterIterations(n).withNoChanges()`** — chains a modifier so `HasChanges` returns false during that iteration
- **`project.normalizedAndStaged()`** — returns true when `NormalizeAndStage` was called during the test
- **`git.withCommitsAhead()`** — returns a git client that reports commits ahead of the base branch
- **`git.withBlockedFile()`** — returns a git client that reports `blocked.md` as present
- **`git.withChangesAndReport()`** — returns a git client that reports uncommitted changes and a present `report.md`
- **`git.withChangesButNoReport()`** — returns a git client that reports uncommitted changes and no `report.md`
- **`git.withNoChanges()`** — returns a git client that reports a clean working tree
- **`git.thatRecordsCallOrder()`** — returns a git client that records the order in which its methods are called
- **`git.switchedBeforeArtifactsCommitted()`** — returns true when `SwitchToBranch` was called before `CommitGeneratedArtifacts` during the test
- **`git.branchSwitched()`** — returns true when `SwitchToBranch` was called during the test
- **`git.blockedFileWritten()`** — returns true when `WriteBlockedFile` was called during the test
- **`git.committedFromReport()`** — returns true when `CommitFromReport` was called during the test
- **`git.artifactsCommitted()`** — returns true when `CommitGeneratedArtifacts` was called during the test
- **`git.orchestrationRemovalCommitted()`** — returns true when `CommitOrchestrationRemoval` was called during the test
- **`github.prCreated()`** — returns true when `CreatePR` was called and produced a pull request
- **`notify.errors()`** — returns the list of error notifications sent during the test
- **`notify.successes()`** — returns the list of success notifications sent during the test
