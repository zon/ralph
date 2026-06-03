# Workflow Run Orchestration

## Purpose

`ralph workflow run`: set up the workspace, validate inputs, synchronize the base branch, and run the project execution loop.

## Interfaces

**Module:** `internal/orchestration/workflow`

```go
type WorkspaceSetupClient interface {
    Setup(flags WorkspaceFlags) error
}

type GitClient interface {
    FetchBranch(branch string) error
    NeedsMerge(branch string) (bool, error)
    Merge(branch string) error
    AbortMerge()
}

type AIClient interface {
    ResolveMergeConflicts(baseBranch, projectBranch string) error
}

type RunnerClient interface {
    RunLocal(proj *project.Project, cfg *config.RalphConfig) error
}

type ConfigClient interface {
    LoadOptional() (*config.RalphConfig, error)
}

type ProjectClient interface {
    Load(path string) (*project.Project, error)
}

type DebugClient interface {
    Setup(branch string) error
}
```

## Orchestration

**Module:** `internal/orchestration/workflow`

```go
type WorkflowRunCmd struct {
    workspace WorkspaceSetupClient
    git       GitClient
    ai        AIClient
    runner    RunnerClient
    config    ConfigClient
    project   ProjectClient
    debug     DebugClient
}

type WorkflowRunFlags struct {
    Repo           string
    CloneBranch    string
    BaseBranch     string
    ProjectBranch  string
    BotName        string
    BotEmail       string
    ProjectPath    string
    InstructionsMd string
    MaxIterations  int
    Model          string
    NoServices     bool
    Debug          string
}

func (w *WorkflowRunCmd) Run(flags WorkflowRunFlags) error {
    if flags.ProjectPath == "" {
        return ErrMissingProjectPath
    }
    if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
        return err
    }
    if flags.Debug != "" {
        if err := w.debug.Setup(flags.Debug); err != nil {
            return err
        }
    }
    cfg, err := w.config.LoadOptional()
    if err != nil {
        return err
    }
    proj, err := w.project.Load(flags.ProjectPath)
    if err != nil {
        return err
    }
    w.applyFlags(proj, cfg, flags)
    if err := w.syncBaseBranch(flags.BaseBranch, flags.ProjectBranch); err != nil {
        return err
    }
    return w.runner.RunLocal(proj, cfg)
}
```

### Helpers

- **`flags.WorkspaceFlags()`** — returns a `WorkspaceFlags` with `Repo`, `CloneBranch`, `BotName`, and `BotEmail` from the run flags; `TargetBranch` is empty (the project branch is cloned directly), `Symlinks` is true
- **`w.workspace.Setup(flags)`** — delegates to the workspace setup orchestration defined in [workflow-workspace/orchestration.md](../workflow-workspace/orchestration.md)
- **`w.debug.Setup(branch)`** — clones the given ralph source branch into `/workspace/ralph` and writes a wrapper script at `/usr/local/bin/ralph` that invokes `go run ./cmd/ralph/main.go` from the cloned source
- **`w.config.LoadOptional()`** — loads `.ralph/config.yaml` from the working directory; returns a default config when the file does not exist; returns an error when the file exists but cannot be parsed
- **`w.project.Load(path)`** — reads and parses the project YAML file at `path`; returns an error when the file is missing or malformed
- **`w.applyFlags(proj, cfg, flags)`** — applies run-specific overrides: sets `proj.MaxIterations` when `flags.MaxIterations > 0`, sets `cfg.Model` when `flags.Model` is non-empty, clears `cfg.Services` when `flags.NoServices` is true, and sets `cfg.InstructionsMd` when `flags.InstructionsMd` is non-empty
- **`w.syncBaseBranch(baseBranch, projectBranch)`** — fetches and merges the base branch into the project branch; see below
- **`w.runner.RunLocal(proj, cfg)`** — runs the full development loop as described in [run-local/orchestration.md](../run-local/orchestration.md)

---

```go
func (w *WorkflowRunCmd) syncBaseBranch(baseBranch, projectBranch string) error {
    if err := w.git.FetchBranch(baseBranch); err != nil {
        return nil
    }
    needsMerge, err := w.git.NeedsMerge(baseBranch)
    if err != nil {
        return err
    }
    if !needsMerge {
        return nil
    }
    if err := w.git.Merge(baseBranch); err != nil {
        w.git.AbortMerge()
        return w.ai.ResolveMergeConflicts(baseBranch, projectBranch)
    }
    return nil
}
```

### Helpers

- **`w.git.FetchBranch(baseBranch)`** — fetches the base branch from origin; a non-nil error is treated as a warning and sync is skipped entirely
- **`w.git.NeedsMerge(baseBranch)`** — returns true when the merge-base of HEAD and the base branch differs from the base branch tip
- **`w.git.Merge(baseBranch)`** — attempts a fast-forward or auto-merge of the base branch into HEAD; returns a non-nil error when conflicts are detected
- **`w.git.AbortMerge()`** — aborts the in-progress merge and restores the working tree
- **`w.ai.ResolveMergeConflicts(baseBranch, projectBranch)`** — invokes the AI development agent with instructions to re-run the merge, resolve all conflicts, run tests, and stage the resolved files

## Tests

**Module:** `internal/orchestration/workflow`

```go
func TestRunMissingProjectPathAbortsBeforeWorkspace(t *testing.T) {
    cmd := run.withMocks()
    err := cmd.Run(flags.withNoProjectPath())
    require.Error(t, err)
    require.False(t, workspace.setupCalled())
}

func TestRunWorkspaceFailureAbortsEarly(t *testing.T) {
    cmd := run.withMocks(
        run.withWorkspace(workspace.thatFailsSetup()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, config.loadCalled())
}

func TestRunDebugSetupFailureAbortsEarly(t *testing.T) {
    cmd := run.withMocks(
        run.withDebug(debug.thatFailsSetup()),
    )
    err := cmd.Run(flags.withDebugBranch("my-ralph-branch"))
    require.Error(t, err)
    require.False(t, config.loadCalled())
}

func TestRunMissingConfigProceedsWithDefaults(t *testing.T) {
    cmd := run.withMocks(
        run.withConfig(config.thatReportsMissing()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, runner.runLocalCalled())
}

func TestRunMalformedConfigAbortsBeforeSync(t *testing.T) {
    cmd := run.withMocks(
        run.withConfig(config.thatFailsParsing()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, git.fetchCalled())
}

func TestRunProjectLoadFailureAbortsBeforeSync(t *testing.T) {
    cmd := run.withMocks(
        run.withProject(project.thatFailsLoad()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, git.fetchCalled())
}

func TestSyncBaseBranchFetchFailureContinues(t *testing.T) {
    cmd := run.withMocks(
        run.withGit(git.thatFailsFetch()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, runner.runLocalCalled())
}

func TestSyncBaseBranchUpToDateSkipsMerge(t *testing.T) {
    cmd := run.withMocks(
        run.withGit(git.thatReportsUpToDate()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.False(t, git.mergeCalled())
}

func TestSyncBaseBranchCleanMergeProceeds(t *testing.T) {
    cmd := run.withMocks(
        run.withGit(git.thatNeedsMerge().thatMergesCleanly()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, git.mergeCalled())
    require.False(t, ai.conflictsResolved())
}

func TestSyncBaseBranchConflictsAbortAndInvokeAI(t *testing.T) {
    cmd := run.withMocks(
        run.withGit(git.thatNeedsMerge().thatProducesConflicts()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, git.mergeAborted())
    require.True(t, ai.conflictsResolved())
}

func TestSyncBaseBranchAIResolutionFailureReturnsError(t *testing.T) {
    cmd := run.withMocks(
        run.withGit(git.thatNeedsMerge().thatProducesConflicts()),
        run.withAI(ai.thatFailsConflictResolution()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, runner.runLocalCalled())
}

func TestRunDelegatesToLocalRunner(t *testing.T) {
    cmd := run.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, runner.runLocalCalled())
}
```

### Helpers

- **`run.withMocks(opts...)`** — constructs a `WorkflowRunCmd` with default mock implementations; pass option helpers to override specific clients
- **`run.withWorkspace(client)`** — option that sets the workspace setup client
- **`run.withGit(client)`** — option that sets the git client
- **`run.withAI(client)`** — option that sets the AI client
- **`run.withRunner(client)`** — option that sets the runner client
- **`run.withConfig(client)`** — option that sets the config client
- **`run.withProject(client)`** — option that sets the project client
- **`run.withDebug(client)`** — option that sets the debug client
- **`flags.any()`** — returns a valid `WorkflowRunFlags` with a non-empty project path and no debug branch
- **`flags.withNoProjectPath()`** — returns `WorkflowRunFlags` with an empty `ProjectPath`
- **`flags.withDebugBranch(branch)`** — returns `WorkflowRunFlags` with `Debug` set to the given branch
- **`workspace.thatFailsSetup()`** — returns a workspace client whose `Setup` returns an error
- **`workspace.setupCalled()`** — returns true when `Setup` was called during the test
- **`debug.thatFailsSetup()`** — returns a debug client whose `Setup` returns an error
- **`config.thatReportsMissing()`** — returns a config client whose `LoadOptional` returns a default config and no error
- **`config.thatFailsParsing()`** — returns a config client whose `LoadOptional` returns an error
- **`config.loadCalled()`** — returns true when `LoadOptional` was called during the test
- **`project.thatFailsLoad()`** — returns a project client whose `Load` returns an error
- **`git.thatFailsFetch()`** — returns a git client whose `FetchBranch` returns an error
- **`git.thatReportsUpToDate()`** — returns a git client whose `NeedsMerge` returns false
- **`git.thatNeedsMerge()`** — returns a git client builder whose `NeedsMerge` returns true
- **`git.thatNeedsMerge().thatMergesCleanly()`** — chains so `Merge` returns nil
- **`git.thatNeedsMerge().thatProducesConflicts()`** — chains so `Merge` returns a non-nil error
- **`git.fetchCalled()`** — returns true when `FetchBranch` was called during the test
- **`git.mergeCalled()`** — returns true when `Merge` was called during the test
- **`git.mergeAborted()`** — returns true when `AbortMerge` was called during the test
- **`ai.thatFailsConflictResolution()`** — returns an AI client whose `ResolveMergeConflicts` returns an error
- **`ai.conflictsResolved()`** — returns true when `ResolveMergeConflicts` was called during the test
- **`runner.runLocalCalled()`** — returns true when `RunLocal` was called during the test
