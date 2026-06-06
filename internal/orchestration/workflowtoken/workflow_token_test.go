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

func TestRunResolvesRepoFromFlags(t *testing.T) {
	cmd := workflowtoken.withMocks()
	err := cmd.Run(flags.withOwnerAndRepo("myorg", "myrepo"))
	require.NoError(t, err)
	gotOwner, gotRepo := repo.lastResolved()
	require.Equal(t, "myorg", gotOwner)
	require.Equal(t, "myrepo", gotRepo)
}
