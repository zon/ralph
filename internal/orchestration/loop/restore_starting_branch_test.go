package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loopRestoring builds an in-process loop command whose git client is the
// supplied mock, so the starting-checkout scenarios can assert what the loop
// did after opening the pull request without touching a real repository.
func loopRestoring(git *mockGitClient, pr *mockPullRequestOpener) *Cmd {
	return NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{},
		&mockAIClient{},
		&mockReportReader{reports: nothingToDoReports()},
		git,
		pr,
		envNotInWorkflow(),
	)
}

// TestRunLocalRestoresBranchTheLoopBranchWasCreatedFrom covers the
// "Starting branch restored after a pull request" scenario: a local loop that
// opened a pull request checks out the branch the loop branch was created from.
func TestRunLocalRestoresBranchTheLoopBranchWasCreatedFrom(t *testing.T) {
	tests := []struct {
		name     string
		starting string
	}{
		{name: "loop started on main returns to main", starting: "main"},
		{name: "loop started on a feature branch returns to it", starting: "feature-x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := &mockGitClient{currentBranch: tt.starting, commitsAhead: true}
			pr := &mockPullRequestOpener{}
			cmd := loopRestoring(git, pr)

			_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

			require.NoError(t, err)
			assert.Equal(t, 1, pr.calls, "the pull request is opened after the loop ends")
			require.True(t, git.checkoutBranchCalled, "the loop checks out a branch after opening the pull request")
			assert.Equal(t, tt.starting, git.lastCheckedOutBranch, "the checkout returns to the branch the loop branch was created from")
		})
	}
}

// TestRunLocalStaysOnLoopBranchWhenItStartedThere asserts the loop leaves the
// checkout alone when the loop branch was already the current branch.
func TestRunLocalStaysOnLoopBranchWhenItStartedThere(t *testing.T) {
	git := &mockGitClient{currentBranch: "loop-fmt", commitsAhead: true}
	cmd := loopRestoring(git, &mockPullRequestOpener{})

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	assert.False(t, git.checkoutBranchCalled, "no checkout is needed when the loop started on the loop branch")
}

// TestRunLocalStaysOnLoopBranchWhenNoPullRequestIsOpened covers the "Checkout
// stays on the loop branch when nothing was committed" scenario: no commit was
// made on loop-<slug>, so no pull request is opened and the checkout stays put.
func TestRunLocalStaysOnLoopBranchWhenNoPullRequestIsOpened(t *testing.T) {
	git := &mockGitClient{currentBranch: "main", commitsAhead: false}
	cmd := loopRestoring(git, &mockPullRequestOpener{})

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	assert.False(t, git.checkoutBranchCalled, "the checkout stays on the loop branch when no pull request is opened")
}

// TestRunResolvedInWorktreeDoesNotRestoreStartingCheckout covers the "Worktree
// mode does not switch the starting checkout" scenario: the pull request opens
// but the current checkout is never switched.
func TestRunResolvedInWorktreeDoesNotRestoreStartingCheckout(t *testing.T) {
	git := &mockGitClient{commitsAhead: true}
	pr := &mockPullRequestOpener{}
	cmd := loopRestoring(git, pr)
	cmd.base = "main"

	err := cmd.RunResolvedInWorktree(&Result{Slug: "fmt", Steps: []string{"run gofmt"}}, 10)

	require.NoError(t, err)
	assert.NotZero(t, pr.calls, "the pull request is opened after the loop ends")
	assert.False(t, git.checkoutBranchCalled, "worktree mode leaves the starting checkout untouched")
}

// TestRunLocalPropagatesCheckoutError asserts a failure to check the starting
// branch back out is returned unchanged after the pull request opens.
func TestRunLocalPropagatesCheckoutError(t *testing.T) {
	checkoutErr := errors.New("checkout boom")
	git := &mockGitClient{currentBranch: "main", commitsAhead: true, checkoutErr: checkoutErr}
	pr := &mockPullRequestOpener{}
	cmd := loopRestoring(git, pr)

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.Error(t, err)
	assert.Equal(t, checkoutErr, err, "the checkout error is returned unchanged")
	assert.Equal(t, 1, pr.calls, "the pull request is opened before the checkout is restored")
}
