package merge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// scanAndCleanupProjects tests
// ---------------------------------------------------------------------------

func TestScanAndCleanupProjects_NoProjectsDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cmd := withMocks()

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.NoError(t, err)

	assert.False(t, projectFindCompleteProjectsCalled(cmd))
	assert.Empty(t, projectRemoveAndCommitCalls(cmd))
	assert.Empty(t, gitPushCalls(cmd))
}

func TestScanAndCleanupProjects_NoCompleteProjects(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return nil, nil
		},
	}
	cmd := withMocks(withProject(mockProj))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.NoError(t, err)

	assert.True(t, projectFindCompleteProjectsCalled(cmd))
	assert.Empty(t, projectRemoveAndCommitCalls(cmd))
}

func TestScanAndCleanupProjects_CompleteProjectsFound(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return []string{"/tmp/projects/complete.yaml"}, nil
		},
	}
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "feature-branch", nil },
	}
	cmd := withMocks(withProject(mockProj), withGit(mockGit))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.NoError(t, err)

	calls := projectRemoveAndCommitCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"/tmp/projects/complete.yaml"}, calls[0])

	assert.True(t, gitCurrentBranchCalled(cmd))
	pushCalls := gitPushCalls(cmd)
	require.Len(t, pushCalls, 1)
	assert.Equal(t, "feature-branch", pushCalls[0])
}

func TestScanAndCleanupProjects_FindCompleteProjectsError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return nil, errMock
		},
	}
	cmd := withMocks(withProject(mockProj))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan for complete projects")
}

func TestScanAndCleanupProjects_RemoveAndCommitError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return []string{"/tmp/projects/complete.yaml"}, nil
		},
		removeAndCommitFunc: func(files []string) error {
			return errMock
		},
	}
	cmd := withMocks(withProject(mockProj))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove complete projects")
}

func TestScanAndCleanupProjects_PushError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return []string{"/tmp/projects/complete.yaml"}, nil
		},
	}
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "feature-branch", nil },
		pushFunc:          func(branch string) error { return errMock },
	}
	cmd := withMocks(withProject(mockProj), withGit(mockGit))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push after removing complete projects")
}

func TestScanAndCleanupProjects_CurrentBranchError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return []string{"/tmp/projects/complete.yaml"}, nil
		},
	}
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "", errMock },
	}
	cmd := withMocks(withProject(mockProj), withGit(mockGit))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current branch")
}

func TestScanAndCleanupProjects_GitHubWaitError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return []string{"/tmp/projects/complete.yaml"}, nil
		},
	}
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "feature-branch", nil },
	}
	mockGH := &mockGitHubClient{
		getPRHeadRefOidFunc: func(pr string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withProject(mockProj), withGit(mockGit), withGitHub(mockGH))

	flags := MergeFlags{PR: "1"}
	err := cmd.scanAndCleanupProjects(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query PR head")
}

// ---------------------------------------------------------------------------
// waitForGitHubHead tests
// ---------------------------------------------------------------------------

func TestWaitForGitHubHead_Match(t *testing.T) {
	mockGit := &mockGitClient{
		revParseFunc: func(rev string) (string, error) {
			return "abc123def456", nil
		},
	}
	mockGH := &mockGitHubClient{
		getPRHeadRefOidFunc: func(pr string) (string, error) {
			return "abc123def456", nil
		},
	}
	cmd := withMocks(withGit(mockGit), withGitHub(mockGH))

	err := cmd.waitForGitHubHead("1")
	require.NoError(t, err)
}

func TestWaitForGitHubHead_RevParseError(t *testing.T) {
	mockGit := &mockGitClient{
		revParseFunc: func(rev string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withGit(mockGit))

	err := cmd.waitForGitHubHead("1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get local HEAD")
}

func TestWaitForGitHubHead_QueryError(t *testing.T) {
	mockGit := &mockGitClient{
		revParseFunc: func(rev string) (string, error) {
			return "abc123def456", nil
		},
	}
	mockGH := &mockGitHubClient{
		getPRHeadRefOidFunc: func(pr string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withGit(mockGit), withGitHub(mockGH))

	err := cmd.waitForGitHubHead("1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query PR head")
}

// ---------------------------------------------------------------------------
// runLocal tests
// ---------------------------------------------------------------------------

func TestRunLocal_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return nil, nil
		},
	}
	mockGH := &mockGitHubClient{}
	cmd := withMocks(withProject(mockProj), withGitHub(mockGH))

	flags := MergeFlags{Local: true, PR: "42", Repo: "owner/repo"}
	err := cmd.runLocal(flags)
	require.NoError(t, err)

	calls := gitHubMergePRCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, "42", calls[0].pr)
	assert.Equal(t, "owner/repo", calls[0].repo)
}

func TestRunLocal_MergePRError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	mockProj := &mockProjectClient{
		findCompleteProjectsFunc: func(dir string) ([]string, error) {
			return nil, nil
		},
	}
	mockGH := &mockGitHubClient{
		mergePRFunc: func(pr, repo string) error {
			return errMock
		},
	}
	cmd := withMocks(withProject(mockProj), withGitHub(mockGH))

	flags := MergeFlags{Local: true, PR: "42", Repo: "owner/repo"}
	err := cmd.runLocal(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock error")
}

// ---------------------------------------------------------------------------
// Run tests
// ---------------------------------------------------------------------------

func TestRun_LocalDispatchesToRunLocal(t *testing.T) {
	mockGH := &mockGitHubClient{}
	cmd := withMocks(withGitHub(mockGH))

	flags := MergeFlags{Local: true, PR: "42", Repo: "owner/repo"}
	err := cmd.Run(flags)
	require.NoError(t, err)

	calls := gitHubMergePRCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, "42", calls[0].pr)
}

func TestRun_RemoteSubmitsWorkflow(t *testing.T) {
	mockWf := &mockWorkflowClient{}
	cmd := withMocks(withWorkflow(mockWf))

	flags := MergeFlags{Branch: "feature-branch", Local: false}
	err := cmd.Run(flags)
	require.NoError(t, err)

	calls := workflowSubmitMergeWorkflowCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, "feature-branch", calls[0])
}

func TestRun_RemoteWorkflowSubmissionError(t *testing.T) {
	mockWf := &mockWorkflowClient{
		submitMergeWorkflowFunc: func(branch string) (string, error) {
			return "", errMock
		},
	}
	cmd := withMocks(withWorkflow(mockWf))

	flags := MergeFlags{Branch: "feature-branch", Local: false}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to submit merge workflow")
}

func TestRun_LocalMergeError(t *testing.T) {
	mockGH := &mockGitHubClient{
		mergePRFunc: func(pr, repo string) error {
			return errMock
		},
	}
	cmd := withMocks(withGitHub(mockGH))

	flags := MergeFlags{Local: true, PR: "42", Repo: "owner/repo"}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock error")
}
