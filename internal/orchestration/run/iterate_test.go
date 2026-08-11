package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestIterateExitsImmediatelyWhenAllComplete(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Zero(t, aiPickCalls(runner))
}

func TestIterateExitsEarlyWhenItemsBecomeComplete(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(5)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 2, aiPickCalls(runner))
	require.Equal(t, 2, aiDevelopCalls(runner))
}

func TestIterateReadsCompletionEachIteration(t *testing.T) {
	projMock := project.ThatReportsIncompleteUntil(3)
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(5)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 4, projMock.IncompleteCallCount())
}

func TestIterateReadsCompletionAgainstSuppliedBase(t *testing.T) {
	projMock := project.ThatReportsIncompleteUntil(1)
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("develop"))
	require.NoError(t, err)
	require.Equal(t, "develop", projMock.LastBase())
}

func TestIterateSkipsCompletedItemsInPicker(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsComplete(0, 2).WithResolvedItems(4).ThenAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(4)), config.Any())
	require.NoError(t, err)
	require.Equal(t, []int{1, 3}, aiLastPickerIndices(runner))
}

func TestIterateReturnsErrorWhenLimitReached(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(1)), config.WithExtraIterations(0))
	require.Error(t, err)
	require.Equal(t, 1, aiPickCalls(runner))
	require.Contains(t, err.Error(), "incomplete")
}

func TestIterateRespectsExtraIterations(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(2))
	require.Error(t, err)
	require.Equal(t, 5, aiPickCalls(runner))
}

func TestIterateDefaultsToTwentyPercentExtra(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(10)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(10)), config.Any())
	require.Error(t, err)
	require.Equal(t, 12, aiPickCalls(runner))
}

func TestIterateDefaultsRoundsUp(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.Equal(t, 4, aiPickCalls(runner))
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
	runner := withMocks(
		withGit(gitWithBlockedFile()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.ErrorIs(t, err, ErrBlocked)
	require.Zero(t, aiPickCalls(runner))
}

func TestIterateFatalPickErrorIsNotRetried(t *testing.T) {
	runner := withMocks(
		withAI(aiThatReturnsFatalPickError()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.Equal(t, 1, aiPickCalls(runner))
	require.Zero(t, aiDevelopCalls(runner))
	require.False(t, gitBlockedFileWritten(runner))
}

func TestIterateNonFatalPickErrorWritesBlockedFile(t *testing.T) {
	runner := withMocks(
		withAI(aiThatReturnsNonFatalPickError()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.True(t, gitBlockedFileWritten(runner))
}

func TestIterateFatalDevelopErrorIsNotRetried(t *testing.T) {
	runner := withMocks(
		withAI(aiThatReturnsFatalDevelopError()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.Equal(t, 1, aiDevelopCalls(runner))
	require.False(t, gitBlockedFileWritten(runner))
}

func TestIterateNonFatalDevelopErrorWritesBlockedFile(t *testing.T) {
	runner := withMocks(
		withAI(aiThatReturnsNonFatalDevelopError()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.True(t, gitBlockedFileWritten(runner))
}
