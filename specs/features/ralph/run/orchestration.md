# Run Orchestration

## Purpose

Validate inputs, resolve execution parameters, and invoke the local or remote runner directly.

## Orchestration

**Module:** `internal/orchestration/run`

```go
type RunCmd struct {
    workspace WorkspaceClient
    project   ProjectClient
    git       GitClient
    config    ConfigClient
    local     LocalRunnerClient
    remote    RemoteRunnerClient
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
    if flags.Local {
        return r.local.RunLocal(setup.Project, setup.Config)
    }
    return r.remote.RunRemote(setup.Project, flags.Follow)
}
```

### Helpers

- **`r.workspace.ChangeDirectory(dir)`** — changes the working directory to `dir`; no-op when `dir` is empty
- **`r.project.ValidateFile(path)`** — verifies the project file exists on disk; returns an error containing `"project file not found"` when absent
- **`flags.Validate()`** — checks flag combinations for mutual incompatibility; returns an error for `--follow + --local` or `--debug + --local`
- **`r.prepareSetup(flags)`** — loads config, project, and current branch; resolves the project branch name, base branch, and max iterations into an `ExecutionSetup`
- **`r.local.RunLocal(proj, cfg)`** — invokes `Runner.RunLocal` to run the development loop in-process
- **`r.remote.RunRemote(proj, follow)`** — invokes `RemoteRunner.RunRemote` to submit an Argo Workflow and optionally follow its logs

---

```go
type LocalRunnerClient interface {
    RunLocal(proj *project.Project, cfg *config.RalphConfig) error
}

type RemoteRunnerClient interface {
    RunRemote(proj *project.Project, follow bool) error
}
```

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

**Module:** `internal/orchestration/run`

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

func TestRunLocalDispatchesToLocalRunner(t *testing.T) {
    cmd := run.withMocks()
    err := cmd.Run(flags.withLocal())
    require.NoError(t, err)
    require.True(t, local.runLocalCalled())
    require.False(t, remote.runRemoteCalled())
}

func TestRunRemoteDispatchesToRemoteRunner(t *testing.T) {
    cmd := run.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, remote.runRemoteCalled())
    require.False(t, local.runLocalCalled())
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
    require.Equal(t, "feature-x", remote.lastProject().BaseBranch)
}

func TestPrepareSetupMaxIterationsFlagOverridesConfig(t *testing.T) {
    cmd := run.withMocks(
        run.withConfig(config.withMaxIterations(5)),
    )
    err := cmd.Run(flags.withMaxIterations(2))
    require.NoError(t, err)
    require.Equal(t, 2, remote.lastProject().MaxIterations)
}
```

### Helpers

- **`run.withMocks(opts...)`** — constructs a `RunCmd` with default mock implementations; pass option helpers to override specific clients
- **`run.withWorkspace(client)`** — option that sets the workspace client
- **`run.withProject(client)`** — option that sets the project client
- **`run.withGit(client)`** — option that sets the git client
- **`run.withConfig(client)`** — option that sets the config client
- **`run.withLocal(client)`** — option that sets the local runner client
- **`run.withRemote(client)`** — option that sets the remote runner client
- **`flags.any()`** — returns a valid `RunFlags` in a default remote state (no `--local`); owned by `internal/cmd`
- **`flags.withLocal()`** — returns `RunFlags` with `Local` set to true; owned by `internal/cmd`
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
- **`local.runLocalCalled()`** — returns true when `RunLocal` was called during the test
- **`remote.runRemoteCalled()`** — returns true when `RunRemote` was called during the test
- **`remote.lastProject()`** — returns the `*project.Project` passed to the most recent `RunRemote` call
