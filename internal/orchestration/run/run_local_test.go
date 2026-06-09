package run

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

func TestRunLocalBeforeCommandFailureAbortsEarly(t *testing.T) {
	runner := withMocks(
		withServices(newServicesThatFailBeforeCommands()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.Any()), config.Any())
	require.Error(t, err)
	require.False(t, gitBranchSwitched(runner))
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatAlwaysFails()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalAllRequirementsPassCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
		withGitHub(newGitHubWithCommitsAhead()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithAllPassing()), config.Any())
	require.NoError(t, err)
	require.True(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithAllPassing()), config.Any())
	require.NoError(t, err)
	require.False(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestRunLocalProjectInputSkipsGeneration(t *testing.T) {
	runner := newRunnerWithMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithAllPassing()), config.Any())
	require.NoError(t, err)
	require.False(t, runner.ai.(*mockAIClient).writeProjectCalled)
	require.False(t, runner.git.(*git.MockClient).CommitGeneratedArtifactsCalled)
}

func TestRunLocalOrchestrationInputGeneratesAndCommitsProject(t *testing.T) {
	runner := newRunnerWithMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"), config.Any())
	require.NoError(t, err)
	require.False(t, runner.ai.(*mockAIClient).writeOrchestrationCalled)
	require.True(t, runner.ai.(*mockAIClient).writeProjectCalled)
	require.True(t, runner.git.(*git.MockClient).CommitGeneratedArtifactsCalled)
}

func TestRunLocalSpecInputGeneratesOrchestrationThenProject(t *testing.T) {
	runner := newRunnerWithMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForSpecInput("specs/features/ralph/run/spec.md"), config.Any())
	require.NoError(t, err)
	require.True(t, runner.ai.(*mockAIClient).writeOrchestrationCalled)
	require.True(t, runner.ai.(*mockAIClient).writeProjectCalled)
	require.True(t, runner.git.(*git.MockClient).CommitGeneratedArtifactsCalled)
}

func TestRunLocalOrchestrationWriteProjectFailureSendsErrorNotification(t *testing.T) {
	ai := &mockAIClient{
		writeProjectFunc: func(*project.InputFile) (*project.Project, error) {
			return nil, errors.New("write project failed")
		},
	}
	runner := newRunnerWithMocks(withAI(ai))
	err := runner.RunLocal(project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, runner.notify.(*mockNotifyClient).errors)
	require.Empty(t, ai.pickCalls)
}

func TestRunLocalSpecWriteOrchestrationFailureAbortsBeforeWriteProject(t *testing.T) {
	ai := &mockAIClient{
		writeOrchestrationFunc: func(*project.InputFile) error {
			return errors.New("write orchestration failed")
		},
	}
	runner := newRunnerWithMocks(withAI(ai))
	err := runner.RunLocal(project.ForSpecInput("specs/features/ralph/run/spec.md"), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, runner.notify.(*mockNotifyClient).errors)
	require.False(t, ai.writeProjectCalled)
	require.Empty(t, ai.pickCalls)
}

func TestRunLocalGenerationHappensAfterBranchSwitch(t *testing.T) {
	order := []string{}
	gitMock := &git.MockClient{
		SwitchToBranchFunc: func(string) error {
			order = append(order, "switch")
			return nil
		},
		CommitGeneratedArtifactsFunc: func(string) error {
			order = append(order, "commit")
			return nil
		},
	}
	runner := newRunnerWithMocks(
		withGit(gitMock),
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"), config.Any())
	require.NoError(t, err)
	require.Equal(t, []string{"switch", "commit"}, order)
}
