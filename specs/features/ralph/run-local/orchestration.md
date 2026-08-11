# Run Local Orchestration

## Purpose

Run the full development loop in-process: set up the environment, generate any missing artifacts, resolve the item array once, iterate until the branch's commit log records every item complete, then clean up, open a pull request, and notify.

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
    proj, err := r.generateArtifacts(input, cfg)
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
    if err := r.removeProjectFile(proj, cfg); err != nil {
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
- **`input.Slug()`** — returns the slug for the input: the project file's top-level `slug` field when it has one, otherwise the input file's base name without its extension
- **`r.git.SwitchToBranch(slug)`** — switches to the branch named by the slug, creating it if it does not exist
- **`r.generateArtifacts(input, cfg)`** — generates any missing artifacts for orchestration or spec inputs, commits them, and returns the project with its item array resolved
- **`r.iterate(proj, cfg)`** — drives the iteration loop; returns nil only when the commit log records every item complete, or a non-nil error when blocked, when a fatal AI error occurs, or when the iteration limit is reached with items still incomplete
- **`r.removeOrchestration(proj)`** — checks whether the project's spec contains an orchestration document and, if so, deletes it and commits the deletion
- **`r.removeProjectFile(proj, cfg)`** — deletes the project file and commits the deletion on its own when cleanup is enabled; no-op otherwise
- **`r.github.CreatePR(proj)`** — generates an AI PR summary from the branch's commit log and opens a pull request from the project branch to the base branch; is a no-op when no commits exist ahead of the base branch
- **`r.notify.Error(slug)`** — sends a desktop error notification for the given project slug when notifications are enabled
- **`r.notify.Success(slug)`** — sends a desktop success notification for the given project slug when notifications are enabled

---

```go
func (r *Runner) generateArtifacts(input *InputFile, cfg *config.RalphConfig) (*project.Project, error) {
    if input.IsProject() {
        return r.project.Resolve(input.Path(), cfg.Items)
    }
    if input.IsSpec() {
        if err := r.ai.WriteOrchestration(input); err != nil {
            return nil, err
        }
    }
    path, err := r.ai.WriteProject(input)
    if err != nil {
        return nil, err
    }
    proj, err := r.project.Resolve(path, cfg.Items)
    if err != nil {
        return nil, err
    }
    return proj, r.git.CommitGeneratedArtifacts(proj)
}
```

### Helpers

- **`input.IsProject()`** — returns true when the input file is a project file (`.yaml`, `.yml`, or `.json`)
- **`input.IsSpec()`** — returns true when the input file is a `spec.md`
- **`input.Path()`** — returns the path of the input file on disk
- **`r.project.Resolve(path, query)`** — parses the file as YAML or JSON, evaluates the item query against it, and returns a `Project` carrying the file path, slug, title, raw contents, and resolved item array; returns an error when the file does not parse, the query fails, or the query yields no items
- **`r.ai.WriteOrchestration(input)`** — invokes the AI agent to generate an `orchestration.md` file in the same directory as the spec and writes it to disk
- **`r.ai.WriteProject(input)`** — invokes the AI agent to generate a project file in `projects/` based on the input, writes it to disk, and returns its path; for spec inputs, reads both the spec and the orchestration from disk
- **`r.git.CommitGeneratedArtifacts(proj)`** — stages and commits all generated files (the project file and any orchestration document) with a fixed message

---

```go
func (r *Runner) iterate(proj *project.Project, cfg *config.RalphConfig) error {
    extra := r.project.ExtraIterations(proj, cfg)
    limit := len(proj.Items) + extra
    for i := 0; i < limit; i++ {
        incomplete, err := r.project.Incomplete(proj, cfg.Base)
        if err != nil {
            return err
        }
        if len(incomplete) == 0 {
            return nil
        }
        if r.git.BlockedFileExists() {
            return ErrBlocked
        }
        if err := r.runIteration(proj, incomplete, cfg); err != nil {
            return err
        }
        if err := r.commitIteration(proj); err != nil {
            return err
        }
    }
    return r.project.IncompleteError(proj, cfg.Base)
}

func (r *Runner) blockAndReturn(err error) error {
    if !r.ai.IsFatal(err) {
        r.git.WriteBlockedFile(err)
    }
    return err
}
```

### Helpers

- **`proj.Items`** — the item array resolved once by `Resolve`; never re-resolved during the loop, so an item's index is stable for the whole run
- **`r.project.Complete(proj, base)`** — parses the completion trailers in `git log <base>..HEAD` and returns the ascending, deduplicated indices they name; warns and drops indices outside `proj.Items`, and warns when a trailer's key does not match the item at its index
- **`r.project.Incomplete(proj, base)`** — returns the items of `proj.Items` whose indices are not in `Complete(proj, base)`, in array order; an empty result is the loop's exit condition
- **`r.git.BlockedFileExists()`** — returns true when `blocked.md` is present in the repository root
- **`r.runIteration(proj, incomplete, cfg)`** — starts services, runs the picker and development agents, stops services, and removes service logs
- **`r.project.ExtraIterations(proj, cfg)`** — returns the configured extra iteration count from config or flag, or 20% of `len(proj.Items)` (rounded up) when unset
- **`r.project.IncompleteError(proj, base)`** — returns an error naming the items that are still incomplete
- **`r.ai.IsFatal(err)`** — returns true when the error is a billing or quota condition that must not be retried
- **`r.git.WriteBlockedFile(err)`** — writes `blocked.md` to the repository root containing the failure reason

---

```go
func (r *Runner) runIteration(proj *project.Project, incomplete []project.Item, cfg *config.RalphConfig) error {
    svc, err := r.services.Start(cfg)
    if err != nil {
        if fixErr := r.ai.FixServiceStartup(cfg, err); fixErr != nil {
            return fixErr
        }
        svc = nil
    }
    defer r.services.Stop(svc)
    defer r.services.RemoveLogs(cfg)
    item, err := r.ai.RunPicker(proj, incomplete)
    if err != nil {
        return r.blockAndReturn(err)
    }
    if err := r.ai.RunDeveloper(proj, item); err != nil {
        return r.blockAndReturn(err)
    }
    return nil
}
```

### Helpers

- **`r.services.Start(cfg)`** — starts all services declared in `.ralph/config.yaml`; returns the service manager and any startup error
- **`r.ai.FixServiceStartup(cfg, err)`** — invokes the development agent with a diagnosis prompt for the failed service; returns nil when the fix succeeds
- **`r.services.Stop(svc)`** — stops all running services; no-op when `svc` is nil
- **`r.services.RemoveLogs(cfg)`** — deletes log files produced by each configured service
- **`r.ai.RunPicker(proj, incomplete)`** — builds a picker prompt from the full project file, the incomplete items with their indices and keys, and the recent commit log, invokes the picker agent with the resolved model and variant, and returns the selected `Item` with its index and key
- **`r.ai.RunDeveloper(proj, item)`** — builds a development prompt containing the full project file, the selected item verbatim, its index, and its key, instructs the agent to write `report.md` and to end that message with the item's completion trailer when the work is finished, then invokes the development agent with the resolved model and variant

Both `RunPicker` and `RunDeveloper` resolve the variant from the execution context using two-level precedence: `--variant` at the command line takes priority; otherwise the top-level `variant` field in `.ralph/config.yaml` is used. When both are unset, `--variant` is omitted entirely from the opencode invocation (unlike model, which always has a default).

`runIteration` performs no post-agent write to the project file. The file is read-only for the whole loop, which is what makes `item.Index` a sufficient identifier.

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
- **`r.ai.GenerateChangelog(proj)`** — invokes the AI agent to produce a changelog and write it to `report.md`; the changelog never contains a completion trailer, so this path completes no item
- **`r.git.CommitFromReport(slug)`** — stages all changes, uses `report.md` as the commit message verbatim, commits, then deletes `report.md`; it neither appends nor rewrites a completion trailer

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

func (r *Runner) removeProjectFile(proj *project.Project, cfg *config.RalphConfig) error {
    if !cfg.Cleanup {
        return nil
    }
    if err := r.project.Remove(proj); err != nil {
        return err
    }
    return r.git.CommitProjectRemoval(proj)
}
```

### Helpers

- **`r.project.HasSpec(proj)`** — returns true when the project references a spec path
- **`r.project.HasOrchestration(proj)`** — returns true when an `orchestration.md` file exists inside the project's spec directory
- **`r.project.RemoveOrchestration(proj)`** — deletes the `orchestration.md` file from the project's spec directory and stages the deletion
- **`r.git.CommitOrchestrationRemoval(slug)`** — commits the staged orchestration deletion with a fixed message
- **`cfg.Cleanup`** — the cleanup setting resolved by the caller: `--cleanup`, then `cleanup` in `.ralph/config.yaml`, then false
- **`r.project.Remove(proj)`** — deletes the project file from disk and stages the deletion
- **`r.git.CommitProjectRemoval(proj)`** — commits the staged project deletion on its own as `chore: clean up completed project <path>`, with no completion trailer

## Tests

**Module:** `internal/orchestration/run`

```go
func TestRunLocalStatsPrintedOnSuccess(t *testing.T) {
    runner := run.withMocks(
        run.withEnv(env.inWorkflow()),
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.True(t, ai.statsPrinted())
}

func TestRunLocalStatsPrintedOnFailure(t *testing.T) {
    runner := run.withMocks(
        run.withEnv(env.inWorkflow()),
        run.withAI(ai.thatAlwaysFails()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.True(t, ai.statsPrinted())
}

func TestRunLocalStatsNotPrintedWhenNotInWorkflow(t *testing.T) {
    runner := run.withMocks(
        run.withEnv(env.notInWorkflow()),
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
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
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, ai.writeProjectCalled())
    require.False(t, git.artifactsCommitted())
}

func TestRunLocalOrchestrationInputGeneratesAndCommitsProject(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forOrchestration(), config.any())
    require.NoError(t, err)
    require.False(t, ai.writeOrchestrationCalled())
    require.True(t, ai.writeProjectCalled())
    require.True(t, git.artifactsCommitted())
}

func TestRunLocalSpecInputGeneratesOrchestrationThenProject(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
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
        run.withProject(project.thatReportsAllComplete()),
        run.withGit(git.thatRecordsCallOrder()),
    )
    err := runner.RunLocal(input.forOrchestration(), config.any())
    require.NoError(t, err)
    require.True(t, git.switchedBeforeArtifactsCommitted())
}

func TestRunLocalResolvesItemsWithConfiguredQuery(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withItems(".requirements"))
    require.NoError(t, err)
    require.Equal(t, ".requirements", project.lastQuery())
}

func TestRunLocalItemQueryYieldingNoItemsAborts(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatFailsResolution()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
    require.Empty(t, ai.pickCalls())
}

func TestRunLocalResolvesItemsOncePerRun(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(3)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(5)), config.any())
    require.NoError(t, err)
    require.Equal(t, 1, project.resolveCount())
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatAlwaysFails()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
}

func TestRunLocalAllItemsCompleteCreatesPR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
        run.withGit(git.withCommitsAhead()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.True(t, github.prCreated())
    require.NotEmpty(t, notify.successes())
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, github.prCreated())
    require.NotEmpty(t, notify.successes())
}

func TestIterateExitsImmediatelyWhenAllComplete(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.pickCalls())
}

func TestIterateExitsEarlyWhenItemsBecomeComplete(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(2)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(5)), config.any())
    require.NoError(t, err)
    require.Len(t, ai.pickCalls(), 2)
    require.Len(t, ai.developCalls(), 2)
}

func TestIterateReadsCompletionEachIteration(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(3)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(5)), config.any())
    require.NoError(t, err)
    require.Equal(t, 4, project.incompleteCallCount())
}

func TestIterateReadsCompletionAgainstSuppliedBase(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withBase("develop"))
    require.NoError(t, err)
    require.Equal(t, "develop", project.lastBase())
}

func TestIterateSkipsCompletedItemsInPicker(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsComplete(0, 2).thenAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(4)), config.any())
    require.NoError(t, err)
    require.Equal(t, []int{1, 3}, ai.lastPickerIndices())
}

func TestIterateReturnsErrorWhenLimitReached(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsIncomplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(1)), config.withExtraIterations(0))
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 1)
    require.Contains(t, err.Error(), "incomplete")
}

func TestIterateRespectsExtraIterations(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsIncomplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withExtraIterations(2))
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 5)
}

func TestIterateDefaultsToTwentyPercentExtra(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsIncomplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(10)), config.any())
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 12)
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withGit(git.withBlockedFile()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.ErrorIs(t, err, run.ErrBlocked)
    require.Empty(t, ai.pickCalls())
}

func TestIterateFatalPickErrorIsNotRetried(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsFatalPickError()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.Len(t, ai.pickCalls(), 1)
    require.Empty(t, ai.developCalls())
    require.False(t, git.blockedFileWritten())
}

func TestIterateNonFatalPickErrorWritesBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsNonFatalPickError()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.True(t, git.blockedFileWritten())
}

func TestIterateFatalDevelopErrorIsNotRetried(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsFatalDevelopError()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.Len(t, ai.developCalls(), 1)
    require.False(t, git.blockedFileWritten())
}

func TestIterateNonFatalDevelopErrorWritesBlockedFile(t *testing.T) {
    runner := run.withMocks(
        run.withAI(ai.thatReturnsNonFatalDevelopError()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.True(t, git.blockedFileWritten())
}

func TestRunIterationStartsAndStopsServicesEachIteration(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(2)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.Equal(t, 2, services.startCount())
    require.Equal(t, 2, services.stopCount())
    require.Equal(t, 2, services.removeLogsCount())
}

func TestRunIterationServiceStartupFailureTriggersFix(t *testing.T) {
    runner := run.withMocks(
        run.withServices(services.thatFailToStart()),
        run.withProject(project.thatReportsIncompleteUntil(1)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.True(t, ai.serviceFixCalled())
    require.Len(t, ai.pickCalls(), 1)
}

func TestRunIterationServiceFixFailureReturnsError(t *testing.T) {
    runner := run.withMocks(
        run.withServices(services.thatFailToStart()),
        run.withAI(ai.thatFailsServiceFix()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.Empty(t, ai.pickCalls())
}

func TestRunIterationPassesSelectedItemToDeveloper(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
        run.withAI(ai.thatPicksIndex(2)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(4)), config.any())
    require.NoError(t, err)
    require.Equal(t, 2, ai.lastDevelopedIndex())
}

func TestRunIterationLeavesProjectFileUntouched(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(2)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, project.written())
}

func TestCommitIterationUsesReportWhenPresent(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
        run.withGit(git.withChangesAndReport()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.changelogCalls())
    require.True(t, git.committedFromReport())
}

func TestCommitIterationDoesNotAlterReportContents(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
        run.withGit(git.withReport("feat: add serializer\n\nRalph item 0 (csv-serializer) completed")),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.Equal(t, "feat: add serializer\n\nRalph item 0 (csv-serializer) completed", git.lastCommitMessage())
}

func TestCommitIterationGeneratesChangelogWhenNoReport(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
        run.withGit(git.withChangesButNoReport()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.Len(t, ai.changelogCalls(), 1)
    require.True(t, git.committedFromReport())
}

func TestCommitIterationSkipsCommitWhenNoChanges(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
        run.withGit(git.withNoChanges()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, git.committedFromReport())
}

func TestRunIterationPassesConfigVariantToAI(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withVariant("high"))
    require.NoError(t, err)
    require.Equal(t, "high", ai.lastVariant())
}

func TestRunIterationOmitsVariantWhenUnset(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsIncompleteUntil(1)),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.Empty(t, ai.lastVariant())
}

func TestRemoveOrchestrationSkipsWhenNoSpec(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete().withNoSpec()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, git.orchestrationRemovalCommitted())
}

func TestRemoveOrchestrationSkipsWhenNoOrchestration(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete().withSpecButNoOrchestration()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, git.orchestrationRemovalCommitted())
}

func TestRemoveOrchestrationRemovesAndCommitsWhenPresent(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete().withOrchestration()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.True(t, project.orchestrationRemoved())
    require.True(t, git.orchestrationRemovalCommitted())
}

func TestRemoveOrchestrationFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete().withOrchestration().thatFailsRemoval()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.Error(t, err)
    require.NotEmpty(t, notify.errors())
    require.False(t, github.prCreated())
}

func TestRemoveProjectFileSkippedWhenCleanupDisabled(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.any())
    require.NoError(t, err)
    require.False(t, project.removed())
    require.False(t, git.projectRemovalCommitted())
}

func TestRemoveProjectFileRemovesAndCommitsWhenCleanupEnabled(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withCleanup())
    require.NoError(t, err)
    require.True(t, project.removed())
    require.True(t, git.projectRemovalCommitted())
}

func TestRemoveProjectFileCommitsBeforePR(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete()),
        run.withGit(git.thatRecordsCallOrder().withCommitsAhead()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withCleanup())
    require.NoError(t, err)
    require.True(t, git.projectRemovalCommittedBeforePR())
}

func TestRemoveProjectFileSkippedWhenIterationLimitReached(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatAlwaysReportsIncomplete()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(1)), config.withCleanup().withExtraIterations(0))
    require.Error(t, err)
    require.False(t, project.removed())
}

func TestRemoveProjectFileFailureSendsErrorNotification(t *testing.T) {
    runner := run.withMocks(
        run.withProject(project.thatReportsAllComplete().thatFailsProjectRemoval()),
    )
    err := runner.RunLocal(input.forProject(project.withItems(3)), config.withCleanup())
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
- **`input.forProject(p)`** — returns an `InputFile` wrapping the given project file; `IsProject()` returns true and `Slug()` returns the project slug; owned by `internal/project`
- **`input.forOrchestration()`** — returns an `InputFile` representing an orchestration document; `IsProject()` returns false, `IsSpec()` returns false; owned by `internal/project`
- **`input.forSpec()`** — returns an `InputFile` representing a spec document; `IsProject()` returns false, `IsSpec()` returns true; owned by `internal/project`
- **`project.any()`** — returns a valid project in a default state; owned by `internal/project`
- **`project.withItems(n)`** — returns a project whose resolved array holds exactly `n` items; owned by `internal/project`
- **`project.thatReportsAllComplete()`** — returns a project client whose `Incomplete` always returns an empty slice
- **`project.thatReportsIncompleteUntil(n)`** — returns a project client whose `Incomplete` returns a non-empty slice for the first `n` calls and an empty slice thereafter
- **`project.thatAlwaysReportsIncomplete()`** — returns a project client whose `Incomplete` always returns a non-empty slice
- **`project.thatReportsComplete(indices...)`** — returns a project client whose `Complete` returns the given indices, so `Incomplete` returns the rest
- **`project.thatReportsComplete(indices...).thenAllComplete()`** — chains a modifier so the second and later calls report every item complete, ending the loop after one iteration
- **`project.thatFailsResolution()`** — returns a project client whose `Resolve` returns an item-query error
- **`project.lastQuery()`** — returns the item query passed to the most recent `Resolve` call
- **`project.lastBase()`** — returns the base branch passed to the most recent `Incomplete` call
- **`project.resolveCount()`** — returns the number of times `Resolve` was called during the test
- **`project.incompleteCallCount()`** — returns the number of times `Incomplete` was called during the test
- **`project.written()`** — returns true when any write to the project file was attempted during the test
- **`project.removed()`** — returns true when `Remove` was called during the test
- **`project.thatReportsAllComplete().withNoSpec()`** — chains a modifier so `HasSpec` returns false
- **`project.thatReportsAllComplete().withSpecButNoOrchestration()`** — chains a modifier so `HasSpec` returns true and `HasOrchestration` returns false
- **`project.thatReportsAllComplete().withOrchestration()`** — chains a modifier so `HasSpec` and `HasOrchestration` both return true
- **`project.thatReportsAllComplete().withOrchestration().thatFailsRemoval()`** — chains a modifier so `RemoveOrchestration` returns an error
- **`project.thatReportsAllComplete().thatFailsProjectRemoval()`** — chains a modifier so `Remove` returns an error
- **`project.orchestrationRemoved()`** — returns true when `RemoveOrchestration` was called during the test
- **`config.any()`** — returns a valid ralph config in a default state; owned by `internal/config`
- **`config.withItems(query)`** — returns a config whose resolved item query is `query`; owned by `internal/config`
- **`config.withBase(branch)`** — returns a config whose resolved base branch is `branch`; owned by `internal/config`
- **`config.withCleanup()`** — returns a config with cleanup enabled; owned by `internal/config`
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
- **`ai.thatPicksIndex(i)`** — returns an AI client whose `RunPicker` always selects the item at index `i`
- **`ai.serviceFixCalled()`** — returns true when `FixServiceStartup` was called during the test
- **`ai.writeOrchestrationCalled()`** — returns true when `WriteOrchestration` was called during the test
- **`ai.writeProjectCalled()`** — returns true when `WriteProject` was called during the test
- **`ai.thatReturnsFatalPickError()`** — returns an AI client whose `RunPicker` returns a billing or quota error
- **`ai.thatReturnsNonFatalPickError()`** — returns an AI client whose `RunPicker` returns a non-fatal error
- **`ai.thatReturnsFatalDevelopError()`** — returns an AI client whose `RunDeveloper` returns a billing or quota error
- **`ai.thatReturnsNonFatalDevelopError()`** — returns an AI client whose `RunDeveloper` returns a non-fatal error
- **`ai.pickCalls()`** — returns the list of projects passed to `RunPicker` during the test
- **`ai.lastPickerIndices()`** — returns the indices of the incomplete items offered to the most recent `RunPicker` call
- **`ai.developCalls()`** — returns the list of items passed to `RunDeveloper` during the test
- **`ai.lastDevelopedIndex()`** — returns the index of the item passed to the most recent `RunDeveloper` call
- **`ai.changelogCalls()`** — returns the list of projects passed to `GenerateChangelog` during the test
- **`git.withCommitsAhead()`** — returns a git client that reports commits ahead of the base branch
- **`git.withBlockedFile()`** — returns a git client that reports `blocked.md` as present
- **`git.withChangesAndReport()`** — returns a git client that reports uncommitted changes and a present `report.md`
- **`git.withReport(message)`** — returns a git client that reports uncommitted changes and a `report.md` containing `message`
- **`git.withChangesButNoReport()`** — returns a git client that reports uncommitted changes and no `report.md`
- **`git.withNoChanges()`** — returns a git client that reports a clean working tree
- **`git.lastCommitMessage()`** — returns the message of the most recent commit created during the test
- **`git.thatRecordsCallOrder()`** — returns a git client that records the order in which its methods are called
- **`git.switchedBeforeArtifactsCommitted()`** — returns true when `SwitchToBranch` was called before `CommitGeneratedArtifacts` during the test
- **`git.branchSwitched()`** — returns true when `SwitchToBranch` was called during the test
- **`git.blockedFileWritten()`** — returns true when `WriteBlockedFile` was called during the test
- **`git.committedFromReport()`** — returns true when `CommitFromReport` was called during the test
- **`git.artifactsCommitted()`** — returns true when `CommitGeneratedArtifacts` was called during the test
- **`git.orchestrationRemovalCommitted()`** — returns true when `CommitOrchestrationRemoval` was called during the test
- **`git.projectRemovalCommitted()`** — returns true when `CommitProjectRemoval` was called during the test
- **`git.projectRemovalCommittedBeforePR()`** — returns true when `CommitProjectRemoval` was called before `CreatePR` during the test
- **`github.prCreated()`** — returns true when `CreatePR` was called and produced a pull request
- **`notify.errors()`** — returns the list of error notifications sent during the test
- **`notify.successes()`** — returns the list of success notifications sent during the test
