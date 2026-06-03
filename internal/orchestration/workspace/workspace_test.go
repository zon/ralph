package workspace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupGitHubAuthFailureAbortsEarly(t *testing.T) {
	cmd := withMocks(
		withGitHub(thatFailsAuth()),
	)
	err := cmd.Setup(flagsAny())
	require.Error(t, err)
	require.False(t, credentialsSetUp(cmd))
}

func TestSetupCredentialsFailureAbortsEarly(t *testing.T) {
	cmd := withMocks(
		withWorkspace(thatFailsCredentials()),
	)
	err := cmd.Setup(flagsAny())
	require.Error(t, err)
	require.False(t, cloned(cmd))
}

func TestSetupCloneFailureAbortsEarly(t *testing.T) {
	cmd := withMocks(
		withGit(thatFailsClone()),
	)
	err := cmd.Setup(flagsAny())
	require.Error(t, err)
	require.False(t, checkoutCalled(cmd))
}

func TestSetupNoTargetBranchSkipsCheckout(t *testing.T) {
	cmd := withMocks()
	err := cmd.Setup(flagsWithNoTargetBranch())
	require.NoError(t, err)
	require.False(t, checkoutCalled(cmd))
}

func TestSetupExistingTargetBranchFetchesAndChecksOut(t *testing.T) {
	cmd := withMocks(
		withGit(thatReportsRemoteBranchExists()),
	)
	err := cmd.Setup(flagsWithTargetBranch("ralph/my-feature"))
	require.NoError(t, err)
	require.True(t, fetchAndCheckoutCalled(cmd))
	require.False(t, createAndCheckoutCalled(cmd))
}

func TestSetupNonExistingBranchWithCreateChecksOut(t *testing.T) {
	cmd := withMocks(
		withGit(thatReportsRemoteBranchAbsent()),
	)
	err := cmd.Setup(flagsWithTargetBranch("ralph/my-feature").withCreateBranch())
	require.NoError(t, err)
	require.True(t, createAndCheckoutCalled(cmd))
}

func TestSetupNonExistingBranchWithoutCreateReturnsError(t *testing.T) {
	cmd := withMocks(
		withGit(thatReportsRemoteBranchAbsent()),
	)
	err := cmd.Setup(flagsWithTargetBranch("ralph/my-feature"))
	require.Error(t, err)
	require.False(t, createAndCheckoutCalled(cmd))
}

func TestSetupSymlinksDisabledSkipsSetup(t *testing.T) {
	cmd := withMocks()
	err := cmd.Setup(flagsWithSymlinksDisabled())
	require.NoError(t, err)
	require.False(t, symlinksSetUp(cmd))
}

func TestSetupSymlinksEnabledCallsSetup(t *testing.T) {
	cmd := withMocks()
	err := cmd.Setup(flagsWithSymlinksEnabled())
	require.NoError(t, err)
	require.True(t, symlinksSetUp(cmd))
}
