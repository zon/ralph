package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

func TestRunIterationPassesConfigVariantToAI(t *testing.T) {
	aiMock := &mockAIClient{}
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(1)),
		withAI(aiMock),
	)
	err := runner.RunLocal(failingProject(), config.WithVariant("high"))
	require.NoError(t, err)
	require.Equal(t, "high", aiMock.lastVariant())
}

func TestRunIterationOmitsVariantWhenUnset(t *testing.T) {
	aiMock := &mockAIClient{}
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(1)),
		withAI(aiMock),
	)
	err := runner.RunLocal(failingProject(), anyConfig())
	require.NoError(t, err)
	require.Empty(t, aiMock.lastVariant())
}
