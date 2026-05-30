package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/project"
)

func TestCleanupNormalizesProjectFileWhenChanged(t *testing.T) {
	projMock := newProjectThatReportsPassingAfterIterations(1)
	projMock.HasChangesFunc = func(_ *project.Project) bool { return true }
	runner := withMocks(withProject(projMock))
	err := runner.RunLocal(failingProject(), anyConfig())
	require.NoError(t, err)
	require.True(t, projMock.NormalizeAndStageCalled)
}

func TestCleanupSkipsNormalizationWhenNoChanges(t *testing.T) {
	projMock := newProjectThatReportsPassingAfterIterations(1)
	projMock.HasChangesFunc = func(_ *project.Project) bool { return false }
	runner := withMocks(withProject(projMock))
	err := runner.RunLocal(failingProject(), anyConfig())
	require.NoError(t, err)
	require.False(t, projMock.NormalizeAndStageCalled)
}
