# Workflow Comment Orchestration

## Purpose

`ralph workflow comment`: set up the workspace on the project branch, invoke the AI agent with a rendered comment prompt, commit any changes, post a reply on the PR.

## Interfaces

**Module:** `internal/orchestration/comment`

```go
type WorkspaceSetupClient interface {
    Setup(flags WorkspaceFlags) error
}

type ConfigClient interface {
    LoadOptional() (*config.RalphConfig, error)
}

type AIClient interface {
    RenderCommentPrompt(ctx CommentContext, instructionsFile string) (string, error)
    RunAgent(prompt string) error
    GenerateChangelog() error
    GenerateCommentReply(ctx CommentContext, pushed bool) (string, error)
}

type ServicesClient interface {
    Start(cfg *config.RalphConfig) (*services.Handle, error)
    Stop(handle *services.Handle)
}

type GitClient interface {
    HasChanges() bool
    ReportExists() bool
    CommitAndPushFromReport() error
}

type GitHubClient interface {
    PostComment(prNumber int, body string) error
}
```

## Orchestration

**Module:** `internal/orchestration/comment`

```go
type WorkflowCommentCmd struct {
    workspace WorkspaceSetupClient
    config    ConfigClient
    ai        AIClient
    services  ServicesClient
    git       GitClient
    github    GitHubClient
}

type WorkflowCommentFlags struct {
    Repo             string
    CloneBranch      string
    ProjectBranch    string
    BotName          string
    BotEmail         string
    CommentBody      string
    PRNumber         int
    RepoOwner        string
    RepoName         string
    NoServices       bool
    InstructionsFile string
}

func (w *WorkflowCommentCmd) Run(flags WorkflowCommentFlags) error {
    if flags.CommentBody == "" {
        return ErrMissingCommentBody
    }
    if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
        return err
    }
    cfg, err := w.config.LoadOptional()
    if err != nil {
        return err
    }
    if flags.NoServices {
        cfg.Services = nil
    }
    prompt, err := w.ai.RenderCommentPrompt(flags.CommentContext(), flags.InstructionsFile)
    if err != nil {
        return err
    }
    svc, err := w.services.Start(cfg)
    if err != nil {
        return err
    }
    defer w.services.Stop(svc)
    if err := w.ai.RunAgent(prompt); err != nil {
        return err
    }
    pushed, err := w.commitChanges()
    if err != nil {
        return err
    }
    reply, err := w.ai.GenerateCommentReply(flags.CommentContext(), pushed)
    if err != nil {
        return err
    }
    return w.github.PostComment(flags.PRNumber, reply)
}
```

### Helpers

- **`flags.WorkspaceFlags()`** — returns a `WorkspaceFlags` with `Repo`, `CloneBranch`, `BotName`, and `BotEmail` from the comment flags; `TargetBranch` is set to `ProjectBranch`, `CreateBranch` is true, `Symlinks` is true
- **`flags.CommentContext()`** — returns a `CommentContext` struct with `CommentBody`, `PRNumber`, `PRBranch`, `RepoOwner`, and `RepoName` populated from the flags
- **`w.workspace.Setup(flags)`** — delegates to the workspace setup orchestration defined in [workflow-workspace/orchestration.md](../workflow-workspace/orchestration.md)
- **`w.config.LoadOptional()`** — loads `.ralph/config.yaml`; returns a default config when the file does not exist; returns an error when the file exists but cannot be parsed
- **`w.ai.RenderCommentPrompt(ctx, instructionsFile)`** — renders the comment instructions template with the PR context variables; uses the custom template at `instructionsFile` when provided, otherwise the built-in default; falls back to the raw template text when the template cannot be parsed
- **`w.services.Start(cfg)`** — starts all services declared in `.ralph/config.yaml`; returns a nil handle when no services are configured, making `Stop` a no-op
- **`w.services.Stop(svc)`** — stops all running services; no-op when `svc` is nil
- **`w.ai.RunAgent(prompt)`** — invokes the AI development agent with the rendered prompt; returns a non-nil error on failure
- **`w.commitChanges()`** — commits and pushes any changes the agent produced; returns true when a commit was pushed; see below
- **`w.ai.GenerateCommentReply(ctx, pushed)`** — generates a reply to the original comment; when `pushed` is true the reply summarizes the committed changes; when false the reply answers the comment without referencing code changes
- **`w.github.PostComment(prNumber, body)`** — posts a reply comment on the pull request

---

```go
func (w *WorkflowCommentCmd) commitChanges() (bool, error) {
    if !w.git.HasChanges() {
        return false, nil
    }
    if !w.git.ReportExists() {
        if err := w.ai.GenerateChangelog(); err != nil {
            return false, err
        }
    }
    if err := w.git.CommitAndPushFromReport(); err != nil {
        return false, err
    }
    return true, nil
}
```

### Helpers

- **`w.git.HasChanges()`** — returns true when the working tree has uncommitted changes
- **`w.git.ReportExists()`** — returns true when `report.md` is present in the repository root
- **`w.ai.GenerateChangelog()`** — invokes the AI agent to produce a changelog and write it to `report.md`
- **`w.git.CommitAndPushFromReport()`** — stages all changes, uses `report.md` as the commit message, commits, deletes `report.md`, and pushes to the remote

## Tests

**Module:** `internal/orchestration/comment`

```go
func TestRunMissingCommentBodyAbortsBeforeWorkspace(t *testing.T) {
    cmd := comment.withMocks()
    err := cmd.Run(flags.withNoCommentBody())
    require.Error(t, err)
    require.False(t, workspace.setupCalled())
}

func TestRunWorkspaceFailureAbortsEarly(t *testing.T) {
    cmd := comment.withMocks(
        comment.withWorkspace(workspace.thatFailsSetup()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, config.loadCalled())
}

func TestRunMalformedConfigAbortsBeforeAgent(t *testing.T) {
    cmd := comment.withMocks(
        comment.withConfig(config.thatFailsParsing()),
    )
    err := cmd.Run(flags.any())
    require.Error(t, err)
    require.False(t, ai.agentInvoked())
}

func TestRunServicesStartedBeforeAgentAndStopped(t *testing.T) {
    cmd := comment.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.Equal(t, 1, services.startCount())
    require.Equal(t, 1, services.stopCount())
    require.True(t, services.startedBeforeAgent())
}

func TestRunNoServicesFlagSkipsServiceStartup(t *testing.T) {
    cmd := comment.withMocks()
    err := cmd.Run(flags.withNoServices())
    require.NoError(t, err)
    require.Equal(t, 0, services.startCount())
}

func TestRunAgentInvokedWithRenderedPrompt(t *testing.T) {
    cmd := comment.withMocks()
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, ai.agentInvoked())
    require.NotEmpty(t, ai.lastPrompt())
}

func TestRunChangesCommittedAndPushed(t *testing.T) {
    cmd := comment.withMocks(
        comment.withGit(git.withChangesAndReport()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, git.committedAndPushed())
}

func TestRunNoChangesSkipsCommit(t *testing.T) {
    cmd := comment.withMocks(
        comment.withGit(git.withNoChanges()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.False(t, git.committedAndPushed())
}

func TestRunReplyPostedAfterCommit(t *testing.T) {
    cmd := comment.withMocks(
        comment.withGit(git.withChangesAndReport()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.True(t, git.committedAndPushed())
    require.True(t, github.commentPosted())
}

func TestRunReplyPostedWhenNoChanges(t *testing.T) {
    cmd := comment.withMocks(
        comment.withGit(git.withNoChanges()),
    )
    err := cmd.Run(flags.any())
    require.NoError(t, err)
    require.False(t, git.committedAndPushed())
    require.True(t, github.commentPosted())
}
```

### Helpers

- **`comment.withMocks(opts...)`** — constructs a `WorkflowCommentCmd` with default mock implementations; pass option helpers to override specific clients
- **`comment.withWorkspace(client)`** — option that sets the workspace setup client
- **`comment.withConfig(client)`** — option that sets the config client
- **`comment.withAI(client)`** — option that sets the AI client
- **`comment.withServices(client)`** — option that sets the services client
- **`comment.withGit(client)`** — option that sets the git client
- **`comment.withGitHub(client)`** — option that sets the GitHub client
- **`flags.any()`** — returns a valid `WorkflowCommentFlags` with a non-empty comment body
- **`flags.withNoCommentBody()`** — returns `WorkflowCommentFlags` with an empty `CommentBody`
- **`flags.withNoServices()`** — returns `WorkflowCommentFlags` with `NoServices` true
- **`workspace.thatFailsSetup()`** — returns a workspace client whose `Setup` returns an error
- **`workspace.setupCalled()`** — returns true when `Setup` was called during the test
- **`config.thatFailsParsing()`** — returns a config client whose `LoadOptional` returns an error
- **`config.loadCalled()`** — returns true when `LoadOptional` was called during the test
- **`ai.agentInvoked()`** — returns true when `RunAgent` was called during the test
- **`ai.lastPrompt()`** — returns the prompt passed to the most recent `RunAgent` call
- **`services.startCount()`** — returns the number of times `Start` was called during the test
- **`services.stopCount()`** — returns the number of times `Stop` was called during the test
- **`services.startedBeforeAgent()`** — returns true when `Start` was called before `RunAgent` during the test
- **`git.withChangesAndReport()`** — returns a git client that reports uncommitted changes and a present `report.md`
- **`git.withNoChanges()`** — returns a git client that reports a clean working tree
- **`git.committedAndPushed()`** — returns true when `CommitAndPushFromReport` was called during the test
- **`github.commentPosted()`** — returns true when `PostComment` was called during the test
