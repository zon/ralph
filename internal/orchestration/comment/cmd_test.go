package comment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
)

func TestRenderInstructions(t *testing.T) {
	tests := []struct {
		name           string
		tmplText       string
		repo           string
		branch         string
		body           string
		pr             string
		expectedOutput string
	}{
		{
			name:           "replaces CommentBody, PRNumber, PRBranch, RepoOwner, RepoName with provided values",
			tmplText:       "Comment: {{.CommentBody}}\nPR: {{.PRNumber}}\nBranch: {{.PRBranch}}\nOwner: {{.RepoOwner}}\nRepo: {{.RepoName}}",
			repo:           "zon/ralph",
			branch:         "feature/test",
			body:           "Please review this",
			pr:             "123",
			expectedOutput: "Comment: Please review this\nPR: 123\nBranch: feature/test\nOwner: zon\nRepo: ralph",
		},
		{
			name:           "splits repo string on / to populate RepoOwner and RepoName correctly",
			tmplText:       "Owner: {{.RepoOwner}}, Repo: {{.RepoName}}",
			repo:           "myorg/my-repo",
			branch:         "main",
			body:           "",
			pr:             "1",
			expectedOutput: "Owner: myorg, Repo: my-repo",
		},
		{
			name:           "returns raw template text when template contains invalid Go template syntax",
			tmplText:       "Invalid: {{.InvalidField",
			repo:           "owner/repo",
			branch:         "main",
			body:           "test",
			pr:             "1",
			expectedOutput: "Invalid: {{.InvalidField",
		},
		{
			name:           "handles repo string without a / by leaving RepoOwner and RepoName as empty strings",
			tmplText:       "Owner: '{{.RepoOwner}}', Repo: '{{.RepoName}}'",
			repo:           "invalid-repo",
			branch:         "main",
			body:           "test",
			pr:             "1",
			expectedOutput: "Owner: '', Repo: ''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderInstructions(tt.tmplText, tt.repo, tt.branch, tt.body, tt.pr)
			assert.Equal(t, tt.expectedOutput, result, "renderInstructions should return expected output")
		})
	}
}

func TestProjectFileFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "converts ralph/my-feature to projects/my-feature.yaml",
			branch:   "ralph/my-feature",
			expected: "projects/my-feature.yaml",
		},
		{
			name:     "converts branch without ralph/ prefix by replacing slashes with dashes",
			branch:   "feature/thing",
			expected: "projects/feature-thing.yaml",
		},
		{
			name:     "handles branch name with no slashes",
			branch:   "my-feature",
			expected: "projects/my-feature.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := projectFileFromBranch(tt.branch)
			assert.Equal(t, tt.expected, result, "projectFileFromBranch should return expected output")
		})
	}
}

func TestRun_SetsVerbose(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("model: test-model\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "comment-instructions.md"), []byte("Do work on {{.PRBranch}}"), 0644))

	mockAI := &mockAIClient{}
	mockSvc := &mockServicesClient{}
	cmd := withMocks(withAI(mockAI), withServices(mockSvc))

	flags := CommentFlags{
		Body:   "test comment",
		Repo:   "owner/repo",
		Branch: "ralph/test",
		PR:     "42",
		Verbose: true,
	}

	err := cmd.Run(flags)
	require.NoError(t, err)

	calls := aiRunAgentCalls(cmd)
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0], "Do work on")
}

func TestRun_NoServicesFlagSkipsServices(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("model: test-model\n"), 0644))

	mockAI := &mockAIClient{}
	mockSvc := &mockServicesClient{}
	cmd := withMocks(withAI(mockAI), withServices(mockSvc))

	flags := CommentFlags{
		Body:       "test",
		Repo:       "owner/repo",
		Branch:     "ralph/test",
		PR:         "1",
		NoServices: true,
	}

	err := cmd.Run(flags)
	require.NoError(t, err)
	assert.False(t, servicesStartCalled(cmd), "services should not be started when NoServices is true")
}

func TestRun_ProjectFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cmd := withMocks()
	flags := CommentFlags{
		Body:   "test",
		Repo:   "owner/repo",
		Branch: "ralph/nonexistent",
		PR:     "1",
	}

	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project file not found")
}

func TestRun_ConfigLoadFailure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	cmd := withMocks()
	flags := CommentFlags{
		Body:   "test",
		Repo:   "owner/repo",
		Branch: "ralph/test",
		PR:     "1",
	}

	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestRun_ServicesStartFailureStillCallsAI(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("model: test-model\nservices:\n  - name: test-svc\n    command: nonexistent\n"), 0644))

	mockAI := &mockAIClient{}
	mockSvc := &mockServicesClient{
		startFunc: func(_ []config.Service) error { return errMock },
	}
	cmd := withMocks(withAI(mockAI), withServices(mockSvc))

	flags := CommentFlags{
		Body:   "test",
		Repo:   "owner/repo",
		Branch: "ralph/test",
		PR:     "1",
	}

	err := cmd.Run(flags)
	require.NoError(t, err)

	calls := aiRunAgentCalls(cmd)
	require.Len(t, calls, 1, "should still call AI agent even when services fail")
}

func TestRun_ServicesStartSuccessCallsStop(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("model: test-model\nservices:\n  - name: test-svc\n    command: sleep\n    args: [\"30\"]\n"), 0644))

	mockAI := &mockAIClient{}
	mockSvc := &mockServicesClient{}
	cmd := withMocks(withAI(mockAI), withServices(mockSvc))

	flags := CommentFlags{
		Body:   "test",
		Repo:   "owner/repo",
		Branch: "ralph/test",
		PR:     "1",
	}

	err := cmd.Run(flags)
	require.NoError(t, err)
	assert.True(t, servicesStartCalled(cmd), "services should be started")
	assert.True(t, servicesStopCalled(cmd), "services should be stopped via defer")
}

func TestRun_AIExecutionFailure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("model: test-model\n"), 0644))

	mockAI := &mockAIClient{
		runAgentFunc: func(_ string) error { return errMock },
	}
	cmd := withMocks(withAI(mockAI))

	flags := CommentFlags{
		Body:   "test",
		Repo:   "owner/repo",
		Branch: "ralph/test",
		PR:     "1",
	}

	err := cmd.Run(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent execution failed")
}

func TestRun_ResolvesPromptFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "projects", "test.yaml"), []byte("slug: test\ntitle: Test\ntasks: []\nrequirements:\n  - slug: test-req\n    description: test requirement\n    items:\n      - test item\n"), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("model: test-model\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "comment-instructions.md"), []byte("Branch: {{.PRBranch}} PR: {{.PRNumber}}"), 0644))

	mockAI := &mockAIClient{}
	cmd := withMocks(withAI(mockAI))

	flags := CommentFlags{
		Body:   "hello",
		Repo:   "org/repo",
		Branch: "ralph/test",
		PR:     "99",
	}

	err := cmd.Run(flags)
	require.NoError(t, err)

	calls := aiRunAgentCalls(cmd)
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0], "Branch: ralph/test")
	assert.Contains(t, calls[0], "PR: 99")
}

func TestStartServicesIfNeeded_NoServices(t *testing.T) {
	cmd := withMocks()
	cleanup := cmd.startServicesIfNeeded(true, []config.Service{{Name: "test", Command: "sleep", Args: []string{"1"}}})
	assert.Nil(t, cleanup, "should return nil when noServices is true")
}

func TestStartServicesIfNeeded_EmptyServiceList(t *testing.T) {
	cmd := withMocks()
	cleanup := cmd.startServicesIfNeeded(false, []config.Service{})
	assert.Nil(t, cleanup, "should return nil when service list is empty")
}

func TestStartServicesIfNeeded_ServiceStartSuccess(t *testing.T) {
	mockSvc := &mockServicesClient{}
	cmd := withMocks(withServices(mockSvc))

	cleanup := cmd.startServicesIfNeeded(false, []config.Service{{Name: "test", Command: "sleep"}})
	require.NotNil(t, cleanup, "should return cleanup function when services start successfully")
	assert.True(t, servicesStartCalled(cmd))

	cleanup()
	assert.True(t, servicesStopCalled(cmd))
}

func TestStartServicesIfNeeded_ServiceStartFailure(t *testing.T) {
	mockSvc := &mockServicesClient{
		startFunc: func(_ []config.Service) error { return errMock },
	}
	cmd := withMocks(withServices(mockSvc))

	cleanup := cmd.startServicesIfNeeded(false, []config.Service{{Name: "test", Command: "nonexistent"}})
	assert.Nil(t, cleanup, "should return nil when services fail to start")
}
