package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrationMerge "github.com/zon/ralph/internal/orchestration/merge"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	"github.com/zon/ralph/internal/project"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Mocks for orchestration Merge tests
// ---------------------------------------------------------------------------

type mockMergeWorkspaceClient struct {
	setupFn func(flags wksp.WorkspaceFlags) error
}

func (m *mockMergeWorkspaceClient) Setup(flags wksp.WorkspaceFlags) error {
	if m.setupFn != nil {
		return m.setupFn(flags)
	}
	return nil
}

type mockMergeGitClient struct {
	commitAndPushFn func(message string) error
}

func (m *mockMergeGitClient) CommitAndPush(message string) error {
	if m.commitAndPushFn != nil {
		return m.commitAndPushFn(message)
	}
	return nil
}

type mockMergeGitHubClient struct {
	waitForHeadSyncFn func(prBranch string) error
	mergePRFn         func(prNumber int) error
}

func (m *mockMergeGitHubClient) WaitForHeadSync(prBranch string) error {
	if m.waitForHeadSyncFn != nil {
		return m.waitForHeadSyncFn(prBranch)
	}
	return nil
}

func (m *mockMergeGitHubClient) MergePR(prNumber int) error {
	if m.mergePRFn != nil {
		return m.mergePRFn(prNumber)
	}
	return nil
}

type mockMergeProjectClient struct {
	loadAllFn       func() ([]*project.Project, error)
	filterPassingFn func(projects []*project.Project) []*project.Project
	deleteAllFn     func(projects []*project.Project) error
}

func (m *mockMergeProjectClient) LoadAll() ([]*project.Project, error) {
	if m.loadAllFn != nil {
		return m.loadAllFn()
	}
	return nil, nil
}

func (m *mockMergeProjectClient) FilterPassing(projects []*project.Project) []*project.Project {
	if m.filterPassingFn != nil {
		return m.filterPassingFn(projects)
	}
	return nil
}

func (m *mockMergeProjectClient) DeleteAll(projects []*project.Project) error {
	if m.deleteAllFn != nil {
		return m.deleteAllFn(projects)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests for orchestration Merge
// ---------------------------------------------------------------------------

func TestWorkflowMergeCmd_Merge_HappyPath(t *testing.T) {
	t.Parallel()

	var workspaceSetupCalled bool
	var mergePRCalled bool

	cmd := orchestrationMerge.NewWorkflowMergeCmd(
		&mockMergeWorkspaceClient{
			setupFn: func(flags wksp.WorkspaceFlags) error {
				workspaceSetupCalled = true
				assert.Equal(t, "owner/repo", flags.Repo)
				return nil
			},
		},
		&mockMergeGitClient{},
		&mockMergeGitHubClient{
			mergePRFn: func(prNumber int) error {
				mergePRCalled = true
				assert.Equal(t, 42, prNumber)
				return nil
			},
		},
		&mockMergeProjectClient{
			loadAllFn: func() ([]*project.Project, error) {
				return nil, nil
			},
		},
	)

	err := cmd.Merge(orchestrationMerge.WorkflowMergeFlags{
		Repo:     "owner/repo",
		PRBranch: "feature",
		PRNumber: 42,
	})
	require.NoError(t, err)
	assert.True(t, workspaceSetupCalled)
	assert.True(t, mergePRCalled)
}

func TestWorkflowMergeCmd_Merge_WithCompletedProjects(t *testing.T) {
	t.Parallel()

	var waitForSyncCalled bool
	var deletedProjects []*project.Project

	cmd := orchestrationMerge.NewWorkflowMergeCmd(
		&mockMergeWorkspaceClient{},
		&mockMergeGitClient{
			commitAndPushFn: func(message string) error {
				assert.Equal(t, "chore: remove completed project files", message)
				return nil
			},
		},
		&mockMergeGitHubClient{
			waitForHeadSyncFn: func(prBranch string) error {
				waitForSyncCalled = true
				assert.Equal(t, "feature", prBranch)
				return nil
			},
			mergePRFn: func(prNumber int) error {
				return nil
			},
		},
		&mockMergeProjectClient{
			loadAllFn: func() ([]*project.Project, error) {
				return []*project.Project{
					{Slug: "completed-1", Path: "proj1.yaml"},
				}, nil
			},
			filterPassingFn: func(projects []*project.Project) []*project.Project {
				return projects
			},
			deleteAllFn: func(projects []*project.Project) error {
				deletedProjects = projects
				return nil
			},
		},
	)

	err := cmd.Merge(orchestrationMerge.WorkflowMergeFlags{
		Repo:     "owner/repo",
		PRBranch: "feature",
		PRNumber: 1,
	})
	require.NoError(t, err)
	assert.True(t, waitForSyncCalled)
	require.Len(t, deletedProjects, 1)
	assert.Equal(t, "completed-1", deletedProjects[0].Slug)
}

func TestWorkflowMergeCmd_Merge_NoCompletedProjects(t *testing.T) {
	t.Parallel()

	var waitForSyncCalled bool

	cmd := orchestrationMerge.NewWorkflowMergeCmd(
		&mockMergeWorkspaceClient{},
		&mockMergeGitClient{},
		&mockMergeGitHubClient{
			waitForHeadSyncFn: func(prBranch string) error {
				waitForSyncCalled = true
				return nil
			},
			mergePRFn: func(prNumber int) error {
				return nil
			},
		},
		&mockMergeProjectClient{
			loadAllFn: func() ([]*project.Project, error) {
				return []*project.Project{}, nil
			},
			filterPassingFn: func(projects []*project.Project) []*project.Project {
				return nil
			},
		},
	)

	err := cmd.Merge(orchestrationMerge.WorkflowMergeFlags{
		Repo:     "owner/repo",
		PRBranch: "feature",
		PRNumber: 1,
	})
	require.NoError(t, err)
	assert.False(t, waitForSyncCalled)
}

func TestWorkflowMergeCmd_Merge_WorkspaceSetupError(t *testing.T) {
	t.Parallel()

	cmd := orchestrationMerge.NewWorkflowMergeCmd(
		&mockMergeWorkspaceClient{
			setupFn: func(flags wksp.WorkspaceFlags) error {
				return assert.AnError
			},
		},
		&mockMergeGitClient{},
		&mockMergeGitHubClient{},
		&mockMergeProjectClient{},
	)

	err := cmd.Merge(orchestrationMerge.WorkflowMergeFlags{})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Tests for workflowMergeProjectClient methods
// ---------------------------------------------------------------------------

func TestWorkflowMergeProjectClient_LoadAll(t *testing.T) {
	dir := t.TempDir()
	origDir := chdir(t, dir)
	defer chdir(t, origDir)

	projectsDir := filepath.Join(dir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	t.Run("loads yaml files from projects directory", func(t *testing.T) {
		writeMergeProjectFile(t, projectsDir, "proj-a.yaml", "proj-a", true)
		writeMergeProjectFile(t, projectsDir, "proj-b.yaml", "proj-b", false)

		client := &workflowMergeProjectClient{}
		projects, err := client.LoadAll()
		require.NoError(t, err)
		assert.Len(t, projects, 2)
	})

	t.Run("returns nil for empty projects directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		origDir2 := chdir(t, emptyDir)
		defer chdir(t, origDir2)

		require.NoError(t, os.MkdirAll(filepath.Join(emptyDir, "projects"), 0755))

		client := &workflowMergeProjectClient{}
		projects, err := client.LoadAll()
		require.NoError(t, err)
		assert.Empty(t, projects)
	})
}

func TestWorkflowMergeProjectClient_FilterPassing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		projects []*project.Project
		want     int
	}{
		{
			name:     "filters to only passing projects",
			projects: []*project.Project{
				{Slug: "passing", Requirements: []project.Requirement{{Slug: "req", Passing: true}}},
				{Slug: "failing", Requirements: []project.Requirement{{Slug: "req", Passing: false}}},
			},
			want: 1,
		},
		{
			name:     "returns empty when none passing",
			projects: []*project.Project{
				{Slug: "failing", Requirements: []project.Requirement{{Slug: "req", Passing: false}}},
			},
			want: 0,
		},
		{
			name:     "returns all when all passing",
			projects: []*project.Project{
				{Slug: "passing", Requirements: []project.Requirement{{Slug: "req", Passing: true}}},
				{Slug: "also-passing", Requirements: []project.Requirement{{Slug: "req", Passing: true}}},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &workflowMergeProjectClient{}
			result := client.FilterPassing(tt.projects)
			assert.Len(t, result, tt.want)
		})
	}
}

func TestWorkflowMergeProjectClient_DeleteAll(t *testing.T) {
	dir := t.TempDir()
	origDir := chdir(t, dir)
	defer chdir(t, origDir)

	projectsDir := filepath.Join(dir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	t.Run("deletes all project files", func(t *testing.T) {
		p1 := writeMergeProjectFile(t, projectsDir, "a.yaml", "proj-a", true)
		p2 := writeMergeProjectFile(t, projectsDir, "b.yaml", "proj-b", false)

		client := &workflowMergeProjectClient{}
		err := client.DeleteAll([]*project.Project{
			{Path: p1},
			{Path: p2},
		})
		require.NoError(t, err)

		_, err = os.Stat(p1)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(p2)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("no error for empty input", func(t *testing.T) {
		client := &workflowMergeProjectClient{}
		err := client.DeleteAll(nil)
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// helper
// ---------------------------------------------------------------------------

func writeMergeProjectFile(t *testing.T, dir, name, slug string, passing bool) string {
	t.Helper()
	proj := project.Project{
		Slug:  slug,
		Title: "Test " + slug,
		Requirements: []project.Requirement{
			{Slug: slug + "-req", Description: "requirement", Items: []string{"item"}, Passing: passing},
		},
	}
	data, err := yaml.Marshal(proj)
	require.NoError(t, err)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}
