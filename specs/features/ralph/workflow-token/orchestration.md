# Workflow Token Orchestration

## Purpose

Generate a GitHub App installation token from mounted credentials and configure git HTTPS authentication, for use inside Argo Workflow containers.

## Orchestration

**Module:** `internal/orchestration/workflowtoken`

```go
type RepoClient interface {
	Resolve(owner, repo string) (string, string, error)
}

type GitHubClient interface {
	GenerateToken(owner, repo, secretsDir string) (string, error)
}

type GitClient interface {
	ConfigureAuth(token string) error
}

type WorkflowTokenCmd struct {
	Repo   RepoClient
	GitHub GitHubClient
	Git    GitClient
}

type Flags struct {
	Owner      string
	Repo       string
	SecretsDir string
}

func (c *WorkflowTokenCmd) Run(flags Flags) error {
	owner, repo, err := c.Repo.Resolve(flags.Owner, flags.Repo)
	if err != nil {
		return err
	}

	token, err := c.GitHub.GenerateToken(owner, repo, flags.SecretsDir)
	if err != nil {
		return err
	}

	return c.Git.ConfigureAuth(token)
}
```

### Helpers

- **`c.Repo.Resolve(owner, repo)`** — returns the owner and repo as-is if both are provided; otherwise auto-detects them from the git remote of the current working directory
- **`c.GitHub.GenerateToken(owner, repo, secretsDir)`** — reads the GitHub App credentials from `secretsDir`, exchanges them for a short-lived installation token scoped to the target repository
- **`c.Git.ConfigureAuth(token)`** — configures git HTTPS authentication using the installation token so subsequent git operations authenticate as the App

## Tests

**Module:** `internal/orchestration/workflowtoken`

```go
func TestRunGeneratesTokenAndConfiguresAuth(t *testing.T) {
	cmd := workflowtoken.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, github.generateTokenCalled())
	require.True(t, git.configureAuthCalled())
}

func TestRunPropagatesRepoResolutionFailure(t *testing.T) {
	cmd := workflowtoken.withMocks(
		workflowtoken.withRepo(repo.thatFails()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, github.generateTokenCalled())
}

func TestRunPropagatesTokenGenerationFailure(t *testing.T) {
	cmd := workflowtoken.withMocks(
		workflowtoken.withGitHub(github.thatFailsTokenGeneration()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, git.configureAuthCalled())
}

func TestRunResolvesRepoFromFlags(t *testing.T) {
	cmd := workflowtoken.withMocks()
	err := cmd.Run(flags.withOwnerAndRepo("myorg", "myrepo"))
	require.NoError(t, err)
	require.Equal(t, repo.lastResolved(), repo.explicit("myorg", "myrepo"))
}
```

### Helpers

- **`workflowtoken.withMocks(overrides...)`** — constructs a `WorkflowTokenCmd` with default passing mock implementations; accepts option overrides
- **`workflowtoken.withRepo(client)`** — override option that substitutes the repo resolver
- **`workflowtoken.withGitHub(client)`** — override option that substitutes the GitHub client
- **`flags.any()`** — returns a valid `Flags` value with a non-empty `SecretsDir`
- **`flags.withOwnerAndRepo(owner, repo)`** — returns a `Flags` value with explicit owner and repo set
- **`github.generateTokenCalled()`** — reports whether `GenerateToken` was called on the GitHub mock
- **`github.thatFailsTokenGeneration()`** — returns a GitHub mock whose `GenerateToken` call returns an error
- **`git.configureAuthCalled()`** — reports whether `ConfigureAuth` was called on the git mock
- **`repo.thatFails()`** — returns a repo mock whose `Resolve` call returns an error
- **`repo.lastResolved()`** — returns the owner/repo pair last passed to `Resolve`
- **`repo.explicit(owner, repo)`** — returns the owner/repo pair for assertion against explicitly provided values
