# Command Orchestration

## Purpose

Run a user-supplied command through the ralph workflow infrastructure (before-commands, services) on the current branch, without branch creation, AI iteration, or PR creation.

## Orchestration

**Module:** `internal/cmd`

```go
type CommandCmd struct {
    WorkingDir string
    Command    []string
    NoNotify   bool
    NoServices bool
    Verbose    bool
    Local      bool
    Follow     bool
    Debug      string
    Context    string

    cleanupRegistrar func(func())
}

func (c *CommandCmd) Run() error {
    if err := c.changeWorkingDirectory(); err != nil {
        return err
    }

    if err := c.validateArgs(); err != nil {
        return err
    }

    if err := CommandFlags{Follow: c.Follow, Local: c.Local, Debug: c.Debug}.Validate(); err != nil {
        return err
    }

    ralphConfig, err := config.LoadConfig()
    if err != nil {
        return err
    }

    ctx := c.createExecutionContext()

    setup := &run.CommandSetup{
        Command: c.Command,
        Config:  ralphConfig,
    }

    return run.ExecuteCommand(ctx, c.cleanupRegistrar, setup)
}
```

### Helpers

- **`c.changeWorkingDirectory()`** — changes the process working directory to `WorkingDir` when set
- **`c.validateArgs()`** — returns an error if `Command` is empty
- **`CommandFlags{...}.Validate()`** — returns an error for incompatible flag combinations (e.g. `--follow` with `--local`)
- **`config.LoadConfig()`** — loads `.ralph/config.yaml`; provides before-commands and services
- **`c.createExecutionContext()`** — builds a `*context.Context` from the command flags (local, follow, verbose, noNotify, noServices, debug, kubeContext)
- **`run.ExecuteCommand(ctx, cleanupRegistrar, setup)`** — runs the command workflow: remote submission or local before+command

---

**Module:** `internal/run`

```go
type CommandSetup struct {
    Command []string
    Config  *config.RalphConfig
}

func ExecuteCommand(ctx *context.Context, cleanupRegistrar func(func()), setup *CommandSetup) error {
    if !ctx.IsLocal() {
        return executeCommandRemote(ctx, setup)
    }

    if err := infrastructureRunBeforeCommands(setup.Config); err != nil {
        return err
    }

    if err := runCommand(setup.Command); err != nil {
        notify.Error("command", ctx.ShouldNotify())
        return err
    }

    notify.Success("command", ctx.ShouldNotify())
    return nil
}
```

### Helpers

- **`executeCommandRemote(ctx, setup)`** — generates and submits an Argo Workflow embedding the command; follows logs if `--follow` is set
- **`infrastructureRunBeforeCommands(cfg)`** — runs `before` commands from config sequentially; aborts on non-zero exit of a non-optional command (reused from `execute.go`)
- **`runCommand(command)`** — executes the user-supplied command tokens in a subprocess, streaming output; returns a non-nil error if the exit code is non-zero
- **`notify.Error(name, enabled)`** — sends a desktop error notification when enabled
- **`notify.Success(name, enabled)`** — sends a desktop success notification when enabled

---

**Module:** `internal/workflow`

```go
func GenerateCommandWorkflow(ctx *execcontext.Context, cloneBranch string) (*Workflow, error) {
    remoteURL, err := resolveRemoteURL(ctx)
    if err != nil {
        return nil, err
    }

    ralphConfig, err := config.LoadConfig()
    if err != nil {
        return nil, err
    }

    repo, err := githubpkg.ParseRemoteURL(remoteURL)
    if err != nil {
        return nil, err
    }

    opts := workflowOptionsFromConfig(ralphConfig, ctx)

    return &Workflow{
        ProjectName: "command",
        Repo:        repo,
        CloneBranch: cloneBranch,
        Command:     ctx.Command(),
        Verbose:     ctx.IsVerbose(),
        DebugBranch: ctx.DebugBranch(),
        NoServices:  ctx.NoServices(),
        Model:       ctx.Model(),
        Image:       opts.Image,
        ConfigMaps:  opts.ConfigMaps,
        Secrets:     opts.Secrets,
        Env:         opts.Env,
        KubeContext: opts.KubeContext,
        Namespace:   opts.Namespace,
        Labels:      opts.Labels,
    }, nil
}
```

### Helpers

- **`resolveRemoteURL(ctx)`** — returns the repo clone URL from `ctx.Repo()` when set, otherwise reads it from local git via `github.GetRepo`
- **`config.LoadConfig()`** — loads `.ralph/config.yaml`; provides workflow image, configmaps, secrets, and namespace
- **`githubpkg.ParseRemoteURL(remoteURL)`** — parses a git remote URL into a `Repo` value (owner and name)
- **`workflowOptionsFromConfig(cfg, ctx)`** — extracts `WorkflowOptions` from config + context kube-context override (shared with other `Generate*` functions)
- **`ctx.Command()`** — returns the raw command token slice stored on the context for embedding in the workflow container args

## Tests

**Module:** `internal/cmd`

```go
test("missing command returns error", func(t) {
    cmd := command.withNoCommand()
    err := cmd.Run()
    assert.Error(err)
    assert.Contains(err.Error(), "command required")
})

test("follow with local returns error", func(t) {
    flags := command.flagsWith(command.follow(), command.local())
    err := flags.Validate()
    assert.Error(err)
    assert.Contains(err.Error(), "--follow")
})
```

### Helpers

- **`command.withNoCommand()`** — constructs a `CommandCmd` with no `Command` tokens set
- **`command.flagsWith(opts...)`** — constructs a `CommandFlags` with the given flag options applied
- **`command.follow()`** — option that sets `Follow: true`
- **`command.local()`** — option that sets `Local: true`

---

**Module:** `internal/run`

```go
test("before-command failure aborts execution", func(t) {
    setup := run.commandSetup.withFailingBeforeCommand()
    ctx := testutil.NewContext()
    err := run.ExecuteCommand(ctx, nil, setup)
    assert.Error(err)
    assert.False(run.commandRan(setup))
})

test("command failure notifies error", func(t) {
    setup := run.commandSetup.withCommand(command.thatFails())
    ctx := testutil.NewContext()
    err := run.ExecuteCommand(ctx, nil, setup)
    assert.Error(err)
    assert.NotEmpty(run.notifications.errors())
})

test("command success notifies success", func(t) {
    setup := run.commandSetup.withCommand(command.thatSucceeds())
    ctx := testutil.NewContext()
    err := run.ExecuteCommand(ctx, nil, setup)
    assert.NoError(err)
    assert.NotEmpty(run.notifications.successes())
})
```

### Helpers

- **`run.commandSetup.withFailingBeforeCommand()`** — returns a `CommandSetup` whose config contains a before-command that exits non-zero
- **`run.commandSetup.withCommand(cmd)`** — returns a `CommandSetup` with the given mock command
- **`command.thatFails()`** — a command token slice whose subprocess exits non-zero
- **`command.thatSucceeds()`** — a command token slice whose subprocess exits zero
- **`run.commandRan(setup)`** — reports whether the main command subprocess was invoked
- **`run.notifications.errors()`** — returns the list of error notifications sent during the test
- **`run.notifications.successes()`** — returns the list of success notifications sent during the test
