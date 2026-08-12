package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestItemSelectionScenario_SelectionIsNotArrayOrder(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsComplete(0).WithResolvedItems(4).ThenAllComplete()),
		withAI(aiThatPicksIndex(3)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(4)), config.Any())
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, aiLastPickerIndices(runner), "the picker is offered only the incomplete items at indices 1, 2, and 3")
	require.Equal(t, 3, aiLastDevelopedIndex(runner), "the picker may select index 3 before index 1")
}

func TestItemSelectionScenario_PickedItemHandedStraightToDeveloper(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsComplete(0, 1).WithResolvedItems(4).ThenAllComplete()),
		withAI(aiThatPicksIndex(2)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(4)), config.Any())
	require.NoError(t, err)
	require.Equal(t, "item-2", aiLastDevelopedValue(runner), "the picked item's value is handed straight to the developer, not re-read from disk")
}

func TestItemSelectionScenario_NoNormalizationAfterAgentRun(t *testing.T) {
	projMock := project.ThatReportsIncompleteUntil(2)
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, projMock.Written(), "no normalization or staging is applied to the project file after an agent run")
}
