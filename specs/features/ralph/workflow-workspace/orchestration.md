# Workflow Workspace Orchestration

## Purpose

Bootstrap the container environment for all `ralph workflow` subcommands: authenticate to GitHub, place AI credentials, configure git identity, clone the repository, check out the target branch, and symlink mounted files.

## Orchestration

**Module:** `internal/orchestration/workspace`

```go
type WorkspaceSetup struct {
    github    GitHubClient
    workspace WorkspaceClient
    git       GitClient
}

type WorkspaceFlags struct {
    Repo         string
    CloneBranch  string
    TargetBranch string
    CreateBranch bool
    BotName      string
    BotEmail     string
    Symlinks     bool
}

func (w *WorkspaceSetup) Setup(flags WorkspaceFlags) error {
    if err := w.github.ConfigureAuth(flags.Repo); err != nil {
        return err
    }
    if err := w.workspace.SetupCredentials(); err != nil {
        return err
    }
    w.git.ConfigureUser(flags.BotName, flags.BotEmail)
    if err := w.git.Clone(flags.CloneBranch); err != nil {
        return err
    }
    if flags.TargetBranch != "" {
        if err := w.checkoutBranch(flags.TargetBranch, flags.CreateBranch); err != nil {
            return err
        }
    }
    if flags.Symlinks {
        return w.workspace.SetupSymlinks()
    }
    return nil
}
```

### Helpers

- **`w.github.ConfigureAuth(repo)`** — generates a GitHub App installation token and configures git HTTPS authentication for the target repo using credentials mounted at `/secrets/github`
- **`w.workspace.SetupCredentials()`** — copies OpenCode credentials from `/secrets/opencode/auth.json` to `~/.local/share/opencode/`
- **`w.git.ConfigureUser(name, email)`** — sets `user.name` and `user.email` in global git config; errors are silently discarded
- **`w.git.Clone(branch)`** — clones the repository at `branch` into `/workspace/repo` and changes the working directory into it
- **`w.checkoutBranch(branch, create)`** — fetches and checks out an existing remote branch, or creates a new local branch when `create` is true; see below
- **`w.workspace.SetupSymlinks()`** — loads `.ralph/config.yaml` and symlinks any ConfigMap or Secret mounts declared with `link: true` into the working directory; skips destinations that are already symlinked

---

```go
func (w *WorkspaceSetup) checkoutBranch(branch string, create bool) error {
    exists, err := w.git.RemoteBranchExists(branch)
    if err != nil {
        return err
    }
    if exists {
        return w.git.FetchAndCheckout(branch)
    }
    if !create {
        return ErrBranchNotFound
    }
    return w.git.CreateAndCheckout(branch)
}
```

### Helpers

- **`w.git.RemoteBranchExists(branch)`** — returns true when the branch exists on the remote
- **`w.git.FetchAndCheckout(branch)`** — fetches the branch from origin and checks it out
- **`w.git.CreateAndCheckout(branch)`** — creates a new local branch at HEAD and checks it out

## Tests

**Module:** `internal/orchestration/workspace`

```go
func TestSetupGitHubAuthFailureAbortsEarly(t *testing.T) {
    cmd := workspace.withMocks(
        workspace.withGitHub(github.thatFailsAuth()),
    )
    err := cmd.Setup(flags.any())
    require.Error(t, err)
    require.False(t, workspace.credentialsSetUp())
}

func TestSetupCredentialsFailureAbortsEarly(t *testing.T) {
    cmd := workspace.withMocks(
        workspace.withWorkspace(workspace.thatFailsCredentials()),
    )
    err := cmd.Setup(flags.any())
    require.Error(t, err)
    require.False(t, git.cloned())
}

func TestSetupCloneFailureAbortsEarly(t *testing.T) {
    cmd := workspace.withMocks(
        workspace.withGit(git.thatFailsClone()),
    )
    err := cmd.Setup(flags.any())
    require.Error(t, err)
    require.False(t, git.checkoutCalled())
}

func TestSetupNoTargetBranchSkipsCheckout(t *testing.T) {
    cmd := workspace.withMocks()
    err := cmd.Setup(flags.withNoTargetBranch())
    require.NoError(t, err)
    require.False(t, git.checkoutCalled())
}

func TestSetupExistingTargetBranchFetchesAndChecksOut(t *testing.T) {
    cmd := workspace.withMocks(
        workspace.withGit(git.thatReportsRemoteBranchExists()),
    )
    err := cmd.Setup(flags.withTargetBranch("ralph/my-feature"))
    require.NoError(t, err)
    require.True(t, git.fetchAndCheckoutCalled())
    require.False(t, git.createAndCheckoutCalled())
}

func TestSetupNonExistingBranchWithCreateChecksOut(t *testing.T) {
    cmd := workspace.withMocks(
        workspace.withGit(git.thatReportsRemoteBranchAbsent()),
    )
    err := cmd.Setup(flags.withTargetBranch("ralph/my-feature").withCreateBranch())
    require.NoError(t, err)
    require.True(t, git.createAndCheckoutCalled())
}

func TestSetupNonExistingBranchWithoutCreateReturnsError(t *testing.T) {
    cmd := workspace.withMocks(
        workspace.withGit(git.thatReportsRemoteBranchAbsent()),
    )
    err := cmd.Setup(flags.withTargetBranch("ralph/my-feature"))
    require.Error(t, err)
    require.False(t, git.createAndCheckoutCalled())
}

func TestSetupSymlinksDisabledSkipsSetup(t *testing.T) {
    cmd := workspace.withMocks()
    err := cmd.Setup(flags.withSymlinksDisabled())
    require.NoError(t, err)
    require.False(t, workspace.symlinksSetUp())
}

func TestSetupSymlinksEnabledCallsSetup(t *testing.T) {
    cmd := workspace.withMocks()
    err := cmd.Setup(flags.withSymlinksEnabled())
    require.NoError(t, err)
    require.True(t, workspace.symlinksSetUp())
}
```

### Helpers

- **`workspace.withMocks(opts...)`** — constructs a `WorkspaceSetup` with default mock implementations; pass option helpers to override specific clients
- **`workspace.withGitHub(client)`** — option that sets the GitHub client
- **`workspace.withWorkspace(client)`** — option that sets the workspace client
- **`workspace.withGit(client)`** — option that sets the git client
- **`flags.any()`** — returns a valid `WorkspaceFlags` with no target branch and symlinks enabled
- **`flags.withNoTargetBranch()`** — returns `WorkspaceFlags` with `TargetBranch` empty
- **`flags.withTargetBranch(branch)`** — returns `WorkspaceFlags` with the given `TargetBranch` and `CreateBranch` false
- **`flags.withTargetBranch(branch).withCreateBranch()`** — chains `CreateBranch` true onto the flags
- **`flags.withSymlinksDisabled()`** — returns `WorkspaceFlags` with `Symlinks` false
- **`flags.withSymlinksEnabled()`** — returns `WorkspaceFlags` with `Symlinks` true
- **`github.thatFailsAuth()`** — returns a GitHub client whose `ConfigureAuth` returns an error
- **`workspace.thatFailsCredentials()`** — returns a workspace client whose `SetupCredentials` returns an error
- **`workspace.credentialsSetUp()`** — returns true when `SetupCredentials` was called during the test
- **`workspace.symlinksSetUp()`** — returns true when `SetupSymlinks` was called during the test
- **`git.thatFailsClone()`** — returns a git client whose `Clone` returns an error
- **`git.cloned()`** — returns true when `Clone` was called during the test
- **`git.checkoutCalled()`** — returns true when `FetchAndCheckout` or `CreateAndCheckout` was called during the test
- **`git.thatReportsRemoteBranchExists()`** — returns a git client whose `RemoteBranchExists` returns true
- **`git.thatReportsRemoteBranchAbsent()`** — returns a git client whose `RemoteBranchExists` returns false
- **`git.fetchAndCheckoutCalled()`** — returns true when `FetchAndCheckout` was called during the test
- **`git.createAndCheckoutCalled()`** — returns true when `CreateAndCheckout` was called during the test
