# Workflow Merge Orchestration

## Purpose

`ralph workflow merge`: set up the workspace on the PR branch, clean up completed project files, confirm GitHub has processed any push, and merge the PR.

## Orchestration

**Module:** `internal/orchestration/merge`

```go
type WorkflowMergeCmd struct {
    workspace WorkspaceSetupClient
    git       GitClient
    github    GitHubClient
    project   ProjectClient
}

type WorkflowMergeFlags struct {
    Repo        string
    CloneBranch string
    PRBranch    string
    PRNumber    int
    BotName     string
    BotEmail    string
}

func (w *WorkflowMergeCmd) Merge(flags WorkflowMergeFlags) error {
    if err := w.workspace.Setup(flags.WorkspaceFlags()); err != nil {
        return err
    }
    pushed, err := w.cleanupCompletedProjects()
    if err != nil {
        return err
    }
    if pushed {
        if err := w.github.WaitForHeadSync(flags.PRBranch); err != nil {
            return err
        }
    }
    return w.github.MergePR(flags.PRNumber)
}
```

### Helpers

- **`flags.WorkspaceFlags()`** — returns a `WorkspaceFlags` with `Repo`, `CloneBranch`, `BotName`, and `BotEmail` from the merge flags; `TargetBranch` is set to `PRBranch`, `CreateBranch` is false, `Symlinks` is false
- **`w.workspace.Setup(flags)`** — delegates to the workspace setup orchestration defined in [workflow-workspace/orchestration.md](../workflow-workspace/orchestration.md)
- **`w.cleanupCompletedProjects()`** — deletes passing project files, commits and pushes the deletion; returns true when a commit was pushed; see below
- **`w.github.WaitForHeadSync(prBranch)`** — polls GitHub until it reports the expected head SHA for the branch; returns an error when the timeout is exceeded
- **`w.github.MergePR(prNumber)`** — merges the pull request into the base branch

---

```go
func (w *WorkflowMergeCmd) cleanupCompletedProjects() (bool, error) {
    projects, err := w.project.LoadAll()
    if err != nil {
        return false, err
    }
    completed := w.project.FilterPassing(projects)
    if len(completed) == 0 {
        return false, nil
    }
    if err := w.project.DeleteAll(completed); err != nil {
        return false, err
    }
    if err := w.git.CommitAndPush("chore: remove completed project files"); err != nil {
        return false, err
    }
    return true, nil
}
```

### Helpers

- **`w.project.LoadAll()`** — loads all project files from the `projects/` directory
- **`w.project.FilterPassing(projects)`** — returns the subset of projects where every requirement has `passing: true`
- **`w.project.DeleteAll(projects)`** — deletes the project files for the given projects from the working tree
- **`w.git.CommitAndPush(message)`** — stages all deletions, commits with the given message, and pushes to the remote

## Tests

**Module:** `internal/orchestration/merge`

```go
func TestMergeWorkspaceFailureAbortsEarly(t *testing.T) {
    cmd := merge.withMocks(
        merge.withWorkspace(workspace.thatFailsSetup()),
    )
    err := cmd.Merge(flags.any())
    require.Error(t, err)
    require.False(t, project.loadAllCalled())
}

func TestMergeNoCompletedProjectsSkipsCleanupAndSync(t *testing.T) {
    cmd := merge.withMocks(
        merge.withProject(project.withNoCompletedProjects()),
    )
    err := cmd.Merge(flags.any())
    require.NoError(t, err)
    require.False(t, git.commitAndPushCalled())
    require.False(t, github.waitForHeadSyncCalled())
    require.True(t, github.mergePRCalled())
}

func TestMergeCompletedProjectsDeletedCommittedAndPushed(t *testing.T) {
    cmd := merge.withMocks(
        merge.withProject(project.withCompletedProjects()),
    )
    err := cmd.Merge(flags.any())
    require.NoError(t, err)
    require.True(t, project.deletedAll())
    require.True(t, git.commitAndPushCalled())
}

func TestMergeCommitFailureAbortsBeforeSync(t *testing.T) {
    cmd := merge.withMocks(
        merge.withProject(project.withCompletedProjects()),
        merge.withGit(git.thatFailsCommitAndPush()),
    )
    err := cmd.Merge(flags.any())
    require.Error(t, err)
    require.False(t, github.waitForHeadSyncCalled())
    require.False(t, github.mergePRCalled())
}

func TestMergeHeadSyncCalledAfterPush(t *testing.T) {
    cmd := merge.withMocks(
        merge.withProject(project.withCompletedProjects()),
    )
    err := cmd.Merge(flags.any())
    require.NoError(t, err)
    require.True(t, github.waitForHeadSyncCalled())
}

func TestMergeHeadSyncTimeoutAbortsBeforeMerge(t *testing.T) {
    cmd := merge.withMocks(
        merge.withProject(project.withCompletedProjects()),
        merge.withGitHub(github.thatTimesOutHeadSync()),
    )
    err := cmd.Merge(flags.any())
    require.Error(t, err)
    require.False(t, github.mergePRCalled())
}

func TestMergePRCalledOnSuccess(t *testing.T) {
    cmd := merge.withMocks()
    err := cmd.Merge(flags.any())
    require.NoError(t, err)
    require.True(t, github.mergePRCalled())
}
```

### Helpers

- **`merge.withMocks(opts...)`** — constructs a `WorkflowMergeCmd` with default mock implementations; pass option helpers to override specific clients
- **`merge.withWorkspace(client)`** — option that sets the workspace setup client
- **`merge.withGit(client)`** — option that sets the git client
- **`merge.withGitHub(client)`** — option that sets the GitHub client
- **`merge.withProject(client)`** — option that sets the project client
- **`flags.any()`** — returns a valid `WorkflowMergeFlags` with a non-zero PR number
- **`workspace.thatFailsSetup()`** — returns a workspace client whose `Setup` returns an error
- **`project.withNoCompletedProjects()`** — returns a project client whose `FilterPassing` returns an empty slice
- **`project.withCompletedProjects()`** — returns a project client whose `FilterPassing` returns one or more projects
- **`project.loadAllCalled()`** — returns true when `LoadAll` was called during the test
- **`project.deletedAll()`** — returns true when `DeleteAll` was called during the test
- **`git.thatFailsCommitAndPush()`** — returns a git client whose `CommitAndPush` returns an error
- **`git.commitAndPushCalled()`** — returns true when `CommitAndPush` was called during the test
- **`github.thatTimesOutHeadSync()`** — returns a GitHub client whose `WaitForHeadSync` returns an error
- **`github.waitForHeadSyncCalled()`** — returns true when `WaitForHeadSync` was called during the test
- **`github.mergePRCalled()`** — returns true when `MergePR` was called during the test
