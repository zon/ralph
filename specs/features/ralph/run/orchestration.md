# Run Command Orchestration

## Purpose

Resolve and validate the input file, reject incompatible flag combinations, then dispatch to the appropriate execution mode.

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

func (r *Runner) Run(inputPath string, cfg *config.RalphConfig) error {
    if err := r.cmd.ChangeWorkingDir(cfg); err != nil {
        return err
    }
    input, err := r.cmd.ResolveInput(inputPath)
    if err != nil {
        return err
    }
    if err := r.cmd.ValidateFlags(cfg); err != nil {
        return err
    }
    switch cfg.Mode {
    case "local":
        return r.RunLocal(input, cfg)
    case "remote":
        return r.remote.RunRemote(input, cfg)
    default:
        return r.RunWorktree(input, cfg)
    }
}
```

### Helpers

- **`r.cmd.ChangeWorkingDir(cfg)`** — changes the process working directory to `cfg.WorkingDir` before anything else; no-op when unset
- **`r.cmd.ResolveInput(inputPath)`** — verifies the file exists on disk, detects its type (project YAML, `orchestration.md`, or `spec.md`), and returns an `InputFile`; returns an error when the file is missing or the type is unrecognized
- **`r.cmd.ValidateFlags(cfg)`** — rejects incompatible flag combinations such as `--follow` and `--debug` with `local` or `worktree` mode
- **`r.RunLocal(input, cfg)`** — drives the development loop in-process in the current checkout
- **`r.RunWorktree(input, cfg)`** — creates a Git worktree for the project branch and drives the development loop in-process inside it
- **`r.remote.RunRemote(input, cfg)`** — submits an Argo Workflow to Kubernetes and returns after submission

## Tests

**Module:** `internal/orchestration/run`

```go
func TestRunChangesWorkingDirFirst(t *testing.T) {
    runner := run.withMocks(
        run.withCmd(cmd.thatRecordsCallOrder()),
    )
    err := runner.Run("projects/foo.yaml", config.withWorkingDir("/tmp/project"))
    require.NoError(t, err)
    require.Equal(t, "ChangeWorkingDir", cmd.firstCallRecorded())
}

func TestRunWorkingDirChangeFails(t *testing.T) {
    runner := run.withMocks(
        run.withCmd(cmd.thatFailsWorkingDirChange()),
    )
    err := runner.Run("projects/foo.yaml", config.any())
    require.Error(t, err)
    require.False(t, cmd.inputResolved())
}

func TestRunInputFileNotFound(t *testing.T) {
    runner := run.withMocks(
        run.withCmd(cmd.thatReturnsInputNotFound()),
    )
    err := runner.Run("projects/missing.yaml", config.any())
    require.Error(t, err)
    require.False(t, cmd.flagsValidated())
}

func TestRunUnrecognizedInputFileType(t *testing.T) {
    runner := run.withMocks(
        run.withCmd(cmd.thatReturnsUnrecognizedInputType()),
    )
    err := runner.Run("some/file.txt", config.any())
    require.Error(t, err)
    require.False(t, cmd.flagsValidated())
}

func TestRunIncompatibleFlagsRejected(t *testing.T) {
    runner := run.withMocks()
    err := runner.Run("projects/foo.yaml", config.withFollowAndLocalMode())
    require.Error(t, err)
    require.False(t, local.runCalled())
    require.False(t, remote.runCalled())
    require.False(t, worktree.runCalled())
}

func TestRunDispatchesToLocalMode(t *testing.T) {
    runner := run.withMocks()
    err := runner.Run("projects/foo.yaml", config.withMode("local"))
    require.NoError(t, err)
    require.True(t, local.runCalled())
    require.False(t, remote.runCalled())
    require.False(t, worktree.runCalled())
}

func TestRunDispatchesToRemoteMode(t *testing.T) {
    runner := run.withMocks()
    err := runner.Run("projects/foo.yaml", config.withMode("remote"))
    require.NoError(t, err)
    require.True(t, remote.runCalled())
    require.False(t, local.runCalled())
    require.False(t, worktree.runCalled())
}

func TestRunDispatchesToWorktreeByDefault(t *testing.T) {
    runner := run.withMocks()
    err := runner.Run("projects/foo.yaml", config.any())
    require.NoError(t, err)
    require.True(t, worktree.runCalled())
    require.False(t, local.runCalled())
    require.False(t, remote.runCalled())
}
```

### Helpers

- **`run.withMocks(opts...)`** — constructs a `Runner` with default mock implementations; pass option helpers to override specific clients
- **`run.withCmd(client)`** — option that sets the cmd client on the mock runner
- **`cmd.thatRecordsCallOrder()`** — returns a cmd client that records the order in which its methods are called
- **`cmd.firstCallRecorded()`** — returns the name of the first method called on the cmd client
- **`cmd.thatFailsWorkingDirChange()`** — returns a cmd client whose `ChangeWorkingDir` returns an error
- **`cmd.thatReturnsInputNotFound()`** — returns a cmd client whose `ResolveInput` returns a not-found error
- **`cmd.thatReturnsUnrecognizedInputType()`** — returns a cmd client whose `ResolveInput` returns an unrecognized type error
- **`cmd.inputResolved()`** — returns true when `ResolveInput` was called
- **`cmd.flagsValidated()`** — returns true when `ValidateFlags` was called
- **`config.any()`** — returns a valid ralph config in a default state; owned by `internal/config`
- **`config.withWorkingDir(path)`** — returns a config with `WorkingDir` set to `path`; owned by `internal/config`
- **`config.withMode(mode)`** — returns a config with `Mode` set to `mode`; owned by `internal/config`
- **`config.withFollowAndLocalMode()`** — returns a config with both `Follow` and `Mode=local` set; owned by `internal/config`
- **`local.runCalled()`** — returns true when `RunLocal` was called on the runner
- **`worktree.runCalled()`** — returns true when `RunWorktree` was called on the runner
- **`remote.runCalled()`** — returns true when `RunRemote` was called on the remote client
