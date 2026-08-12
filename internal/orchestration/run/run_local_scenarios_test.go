package run

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestProjectClientInterfaceSatisfiedByRealAndMockClients(t *testing.T) {
	var _ ProjectClient = (*project.Client)(nil)
	var _ ProjectClient = (*project.MockProject)(nil)
}

func TestProjectClientInterfaceDoesNotExposeLegacyMethods(t *testing.T) {
	typ := reflect.TypeOf((*ProjectClient)(nil)).Elem()
	for _, name := range []string{"Reload", "AllRequirementsPassing", "HasChanges", "NormalizeAndStage", "ExtraIterationsError"} {
		_, ok := typ.MethodByName(name)
		require.False(t, ok, "ProjectClient must not expose %s", name)
	}
}

func TestRunLocalGeneratedProjectResolvesUnderItemQuery(t *testing.T) {
	t.Run("orchestration input", func(t *testing.T) {
		projMock := project.ThatReportsAllComplete()
		runner := withMocks(
			withProject(projMock),
		)
		err := runner.RunLocal(project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"), config.WithItems(".requirements"))
		require.NoError(t, err)
		require.True(t, aiWriteProjectCalled(runner))
		require.Equal(t, "projects/generated.yaml", projMock.LastPath())
		require.Equal(t, ".requirements", projMock.LastQuery())
	})

	t.Run("spec input", func(t *testing.T) {
		projMock := project.ThatReportsAllComplete()
		runner := withMocks(
			withProject(projMock),
		)
		err := runner.RunLocal(project.ForSpecInput("specs/features/ralph/run/spec.md"), config.WithItems(".items"))
		require.NoError(t, err)
		require.True(t, aiWriteOrchestrationCalled(runner))
		require.True(t, aiWriteProjectCalled(runner))
		require.Equal(t, "projects/generated.yaml", projMock.LastPath())
		require.Equal(t, ".items", projMock.LastQuery())
	})
}

func TestRunLocalGeneratedProjectYieldingNoItemsAborts(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatFailsResolution()),
	)
	err := runner.RunLocal(project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
	require.False(t, gitArtifactsCommitted(runner))
	require.Zero(t, aiPickCalls(runner))
}
