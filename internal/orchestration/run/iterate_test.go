package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/project"
)

func TestIterateExitsImmediatelyWhenAllPassing(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithAllPassing()), config.Any())
	require.NoError(t, err)
	require.Empty(t, aiPickCalls(runner))
}

func TestIterateExitsEarlyWhenRequirementsPass(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(2)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.NoError(t, err)
	require.Len(t, aiPickCalls(runner), 2)
	require.Len(t, aiDevelopCalls(runner), 2)
}

func TestIterateSucceedsWhenFinalIterationCompletesAllRequirements(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(3)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.WithExtraIterations(2))
	require.NoError(t, err)
	require.Len(t, aiPickCalls(runner), 3)
}

func TestIterateReturnsErrorAtLimit(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.WithExtraIterations(2))
	require.Error(t, err)
	require.Len(t, aiPickCalls(runner), 3)
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
	runner := withMocks(
		withGit(newGitWithBlockedFile()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.ErrorIs(t, err, ErrBlocked)
	require.Empty(t, aiPickCalls(runner))
}


func TestIterateFatalPickErrorIsNotRetried(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatReturnsFatalError()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.Error(t, err)
	require.Len(t, aiPickCalls(runner), 1)
	require.Empty(t, aiDevelopCalls(runner))
	require.False(t, gitBlockedFileWritten(runner))
}

func TestIterateNonFatalPickErrorWritesBlockedFile(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatReturnsNonFatalError()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.Error(t, err)
	require.True(t, gitBlockedFileWritten(runner))
}

func TestIterateFatalDevelopErrorIsNotRetried(t *testing.T) {
	runner := withMocks(
		withAI(&mockAIClient{
			runDeveloperFunc: func(string) error { return errFatal },
			isFatalFunc:      func(err error) bool { return err == errFatal },
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.Error(t, err)
	require.Len(t, aiDevelopCalls(runner), 1)
	require.False(t, gitBlockedFileWritten(runner))
}

func TestIterateNonFatalDevelopErrorWritesBlockedFile(t *testing.T) {
	runner := withMocks(
		withAI(&mockAIClient{
			runDeveloperFunc: func(string) error { return errNonFatal },
			isFatalFunc:      func(err error) bool { return false },
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.Error(t, err)
	require.True(t, gitBlockedFileWritten(runner))
}

func TestIterateReturnsErrorWhenLimitReached(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirementsCount(1)), config.WithExtraIterations(0))
	require.Error(t, err)
	require.Len(t, aiPickCalls(runner), 1)
}

func TestIterateRespectsExtraIterations(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirementsCount(3)), config.WithExtraIterations(2))
	require.Error(t, err)
	require.Len(t, aiPickCalls(runner), 5)
}

func TestIterateDefaultsToTwentyPercentExtra(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirementsCount(10)), config.Any())
	require.Error(t, err)
	require.Len(t, aiPickCalls(runner), 12)
}

func TestIterateDefaultsRoundsUp(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirementsCount(3)), config.Any())
	require.Error(t, err)
	require.Len(t, aiPickCalls(runner), 4)
}

func TestRunLocalSkipsPRWhenIterationLimitReached(t *testing.T) {
	prCalled := false
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
		withGitHub(&github.MockClient{
			CreatePRFunc: func(*project.Project) error {
				prCalled = true
				return nil
			},
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirementsCount(1)), config.WithExtraIterations(0))
	require.Error(t, err)
	require.False(t, prCalled, "PR should not be created when iteration limit is reached")
}
