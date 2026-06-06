package workflowtoken

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestScenarioSuccessfulTokenGeneration(t *testing.T) {
	cmd := workflowtoken.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, github.generateTokenCalled(), "a GitHub App installation token should be generated")
	require.True(t, git.configureAuthCalled(), "git HTTPS authentication should be configured")
}

func TestScenarioMissingCredentials(t *testing.T) {
	cmd := workflowtoken.withMocks(
		workflowtoken.withGitHub(github.thatFailsTokenGeneration()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err, "an error should be returned when credentials are missing")
	require.False(t, git.configureAuthCalled(), "no git configuration should be written")
}

func TestScenarioInvalidCredentials(t *testing.T) {
	cmd := workflowtoken.withMocks(
		workflowtoken.withGitHub(github.thatFailsTokenGeneration()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err, "an error should be returned when the GitHub API rejects the JWT")
	require.False(t, git.configureAuthCalled(), "no git configuration should be written")
}

func TestScenarioAutoDetectionFromGitRemote(t *testing.T) {
	cmd := workflowtoken.withMocks(
		workflowtoken.withRepo(repo.thatDetectsFromRemote()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	gotOwner, gotRepo := repo.lastResolved()
	require.Equal(t, "", gotOwner, "Resolve should be called with empty owner when --owner is not provided")
	require.Equal(t, "", gotRepo, "Resolve should be called with empty repo when --repo is not provided")
	tokenOwner, tokenRepo := github.generateTokenLastArgs()
	require.Equal(t, "detected-owner", tokenOwner, "GenerateToken should receive owner detected from git remote")
	require.Equal(t, "detected-repo", tokenRepo, "GenerateToken should receive repo detected from git remote")
	require.True(t, git.configureAuthCalled(), "git auth should be configured after successful token generation")
}

func TestRunResolvesRepoFromFlags(t *testing.T) {
	cmd := workflowtoken.withMocks()
	err := cmd.Run(flags.withOwnerAndRepo("myorg", "myrepo"))
	require.NoError(t, err)
	gotOwner, gotRepo := repo.lastResolved()
	require.Equal(t, "myorg", gotOwner)
	require.Equal(t, "myrepo", gotRepo)
}
