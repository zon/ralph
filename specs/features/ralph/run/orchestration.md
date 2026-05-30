# Run Orchestration

## Purpose

Validate inputs, resolve execution parameters, and dispatch to the local or remote runner.

## Orchestration

**Module:** `internal/orchestration/cmd`

```go
type RunCmd struct {
    workspace WorkspaceClient
    project   ProjectClient
    git       GitClient
    config    ConfigClient
    runner    RunnerClient
}

func (r *RunCmd) Run(flags RunFlags) error {
    if err := r.workspace.ChangeDirectory(flags.WorkingDir); err != nil {
        return err
    }
    if err := r.project.ValidateFile(flags.ProjectFile); err != nil {
        return err
    }
    if err := flags.Validate(); err != nil {
        return err
    }
    setup, err := r.prepareSetup(flags)
    if err != nil {
        return err
    }
    return r.runner.Execute(setup)
}
```

### Helpers

- **`r.workspace.ChangeDirectory(dir)`** — changes the working directory to `dir`; no-op when `dir` is empty
- **`r.project.ValidateFile(path)`** — verifies the project file exists on disk; returns an error containing `"project file not found"` when absent
- **`flags.Validate()`** — checks flag combinations for mutual incompatibility; returns an error for `--follow + --local` or `--debug + --local`
- **`r.prepareSetup(flags)`** — loads config, project, and current branch; resolves the project branch name, base branch, and max iterations into an `ExecutionSetup`
- **`r.runner.Execute(setup)`** — dispatches to the local runner when `--local` is set, otherwise to the remote runner

---

```go
func (r *RunCmd) prepareSetup(flags RunFlags) (ExecutionSetup, error) {
    cfg, err := r.config.Load()
    if err != nil {
        return ExecutionSetup{}, err
    }
    proj, err := r.project.Load(flags.ProjectFile)
    if err != nil {
        return ExecutionSetup{}, err
    }
    currentBranch, err := r.git.CurrentBranch()
    if err != nil {
        return ExecutionSetup{}, err
    }
    projectBranch := git.BranchName(proj.Slug)
    return ExecutionSetup{
        Project:       proj,
        Config:        cfg,
        BranchName:    projectBranch,
        CurrentBranch: currentBranch,
        BaseBranch:    resolveBaseBranch(flags.Base, currentBranch, projectBranch, cfg.DefaultBranch),
        MaxIterations: resolveMaxIterations(cfg.MaxIterations, flags.MaxIterations),
    }, nil
}
```

### Helpers

- **`r.config.Load()`** — loads `.ralph/config.yaml` from the working directory
- **`r.project.Load(path)`** — reads and parses the project YAML file at `path`
- **`r.git.CurrentBranch()`** — returns the name of the currently checked-out git branch
- **`git.BranchName(slug)`** — derives the project branch name from the slug: lowercased, spaces/underscores/dots become hyphens, non-alphanumeric characters stripped, consecutive and leading/trailing hyphens collapsed; empty result becomes `unnamed-project`
- **`resolveBaseBranch(base, current, project, default)`** — returns `base` when non-empty; returns `current` when it differs from `project`; otherwise returns `default`
- **`resolveMaxIterations(cfgMax, flagMax)`** — returns `flagMax` when non-zero; otherwise returns `cfgMax`

## Tests

**Module:** `internal/orchestration/cmd`

```go
func TestRunWorkingDirectoryFailureAbortsEarly(t *testing.T) {
    cmd := run.withMocks(
        run.withWorkspace(workspace.thatFailsChangeDirectory()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, project.fileValidated())
}

func TestRunProjectFileNotFoundAbortsEarly(t *testing.T) {
    cmd := run.withMocks(
        run.withProject(project.thatFailsValidation()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, config.loaded())
}

func TestRunIncompatibleFlagsAbortBeforeSetup(t *testing.T) {
    cmd := run.withMocks()
    err := cmd.Run(flags.withFollowAndLocal())
    require.Error(t, err)
    require.False(t, config.loaded())
}

func TestRunDispatchesWithPreparedSetup(t *testing.T) {
    cmd := run.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, runner.executeCalled())
}

func TestPrepareSetupConfigLoadFailureAbortsEarly(t *testing.T) {
    cmd := run.withMocks(
        run.withConfig(config.thatFailsLoad()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, project.loaded())
}

func TestPrepareSetupProjectLoadFailureAbortsEarly(t *testing.T) {
    cmd := run.withMocks(
        run.withProject(project.thatFailsLoad()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, git.currentBranchCalled())
}

func TestPrepareSetupBaseBranchFromCurrentWhenDifferentFromProject(t *testing.T) {
    cmd := run.withMocks(
        run.withGit(git.onBranch("feature-x")),
        run.withProject(project.withSlug("my-project")),
    )
    err := cmd.Run(flags.withNoBase())
    require.NoError(t, err)
    require.Equal(t, "feature-x", runner.lastSetup().BaseBranch)
}

func TestPrepareSetupMaxIterationsFlagOverridesConfig(t *testing.T) {
    cmd := run.withMocks(
        run.withConfig(config.withMaxIterations(5)),
    )
    err := cmd.Run(flags.withMaxIterations(2))
    require.NoError(t, err)
    require.Equal(t, 2, runner.lastSetup().MaxIterations)
}
```

### Helpers

- **`run.withMocks(opts...)`** — constructs a `RunCmd` with default mock implementations; pass option helpers to override specific clients
- **`run.withWorkspace(client)`** — option that sets the workspace client on the mock runner
- **`run.withProject(client)`** — option that sets the project client on the mock runner
- **`run.withGit(client)`** — option that sets the git client on the mock runner
- **`run.withConfig(client)`** — option that sets the config client on the mock runner
- **`flags.any()`** — returns a valid `RunFlags` in a default state with compatible flag values; owned by `internal/cmd`
- **`flags.withFollowAndLocal()`** — returns `RunFlags` with both `Follow` and `Local` set to true; owned by `internal/cmd`
- **`flags.withNoBase()`** — returns `RunFlags` with an empty `Base` field; owned by `internal/cmd`
- **`flags.withMaxIterations(n)`** — returns `RunFlags` with `MaxIterations` set to `n`; owned by `internal/cmd`
- **`workspace.thatFailsChangeDirectory()`** — returns a workspace client whose `ChangeDirectory` returns an error
- **`project.thatFailsValidation()`** — returns a project client whose `ValidateFile` returns an error
- **`project.thatFailsLoad()`** — returns a project client whose `Load` returns an error
- **`project.withSlug(slug)`** — returns a project client whose `Load` returns a project with the given slug
- **`project.fileValidated()`** — returns true when `ValidateFile` was called during the test
- **`project.loaded()`** — returns true when `Load` was called during the test
- **`config.thatFailsLoad()`** — returns a config client whose `Load` returns an error
- **`config.withMaxIterations(n)`** — returns a config client whose loaded config has `MaxIterations` set to `n`
- **`config.loaded()`** — returns true when `Load` was called during the test
- **`git.onBranch(name)`** — returns a git client whose `CurrentBranch` returns `name`
- **`git.currentBranchCalled()`** — returns true when `CurrentBranch` was called during the test
- **`runner.executeCalled()`** — returns true when `Execute` was called during the test
- **`runner.lastSetup()`** — returns the `ExecutionSetup` passed to the most recent `Execute` call
