package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
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
	const maxIterations = 3
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(maxIterations)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithMaxIterations(maxIterations)), config.Any())
	require.NoError(t, err)
	require.Len(t, aiPickCalls(runner), maxIterations)
}

func TestIterateReturnsErrorAtMaxIterations(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithMaxIterations(3)), config.Any())
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
