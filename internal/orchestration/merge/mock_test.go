package merge

import (
	"io"

	"github.com/zon/ralph/internal/output"
)

// mockGitClient
type mockGitClient struct {
	currentBranchFunc func() (string, error)
	revParseFunc      func(string) (string, error)
	pushFunc          func(string) error

	currentBranchCalled bool
	pushCalls           []string
}

func (m *mockGitClient) CurrentBranch() (string, error) {
	m.currentBranchCalled = true
	if m.currentBranchFunc != nil {
		return m.currentBranchFunc()
	}
	return "main", nil
}

func (m *mockGitClient) RevParse(rev string) (string, error) {
	if m.revParseFunc != nil {
		return m.revParseFunc(rev)
	}
	return "abc123def456", nil
}

func (m *mockGitClient) Push(branch string) error {
	m.pushCalls = append(m.pushCalls, branch)
	if m.pushFunc != nil {
		return m.pushFunc(branch)
	}
	return nil
}

// mockGitHubClient
type mockGitHubClient struct {
	mergePRFunc          func(string, string) error
	getPRHeadRefOidFunc  func(string) (string, error)

	mergePRCalls     []mergePRCall
	prHeadRefOidCall bool
}

type mergePRCall struct {
	pr   string
	repo string
}

func (m *mockGitHubClient) MergePR(pr, repo string) error {
	m.mergePRCalls = append(m.mergePRCalls, mergePRCall{pr: pr, repo: repo})
	if m.mergePRFunc != nil {
		return m.mergePRFunc(pr, repo)
	}
	return nil
}

func (m *mockGitHubClient) GetPRHeadRefOid(pr string) (string, error) {
	m.prHeadRefOidCall = true
	if m.getPRHeadRefOidFunc != nil {
		return m.getPRHeadRefOidFunc(pr)
	}
	return "abc123def456", nil
}

// mockProjectClient
type mockProjectClient struct {
	findCompleteProjectsFunc func(string) ([]string, error)
	removeAndCommitFunc      func([]string) error

	findCompleteProjectsCalled bool
	removeAndCommitCalls       [][]string
}

func (m *mockProjectClient) FindCompleteProjects(dir string) ([]string, error) {
	m.findCompleteProjectsCalled = true
	if m.findCompleteProjectsFunc != nil {
		return m.findCompleteProjectsFunc(dir)
	}
	return nil, nil
}

func (m *mockProjectClient) RemoveAndCommit(files []string) error {
	m.removeAndCommitCalls = append(m.removeAndCommitCalls, files)
	if m.removeAndCommitFunc != nil {
		return m.removeAndCommitFunc(files)
	}
	return nil
}

// mockWorkflowClient
type mockWorkflowClient struct {
	submitMergeWorkflowFunc func(string) (string, error)

	submitMergeWorkflowCalls []string
}

func (m *mockWorkflowClient) SubmitMergeWorkflow(branch string) (string, error) {
	m.submitMergeWorkflowCalls = append(m.submitMergeWorkflowCalls, branch)
	if m.submitMergeWorkflowFunc != nil {
		return m.submitMergeWorkflowFunc(branch)
	}
	return "merge-workflow-123", nil
}

// option types
type mergeOption func(*MergeCmd)

func withGit(gc GitClient) mergeOption {
	return func(m *MergeCmd) {
		m.git = gc
	}
}

func withGitHub(gc GitHubClient) mergeOption {
	return func(m *MergeCmd) {
		m.github = gc
	}
}

func withProject(pc ProjectClient) mergeOption {
	return func(m *MergeCmd) {
		m.project = pc
	}
}

func withWorkflow(wc WorkflowClient) mergeOption {
	return func(m *MergeCmd) {
		m.workflow = wc
	}
}

func withMocks(opts ...mergeOption) *MergeCmd {
	m := &MergeCmd{
		git:      &mockGitClient{},
		github:   &mockGitHubClient{},
		project:  &mockProjectClient{},
		workflow: &mockWorkflowClient{},
		out:      output.NewClient(io.Discard, io.Discard, false),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Accessor helpers
func gitCurrentBranchCalled(cmd *MergeCmd) bool {
	if mc, ok := cmd.git.(*mockGitClient); ok {
		return mc.currentBranchCalled
	}
	return false
}

func gitPushCalls(cmd *MergeCmd) []string {
	if mc, ok := cmd.git.(*mockGitClient); ok {
		return mc.pushCalls
	}
	return nil
}

func gitHubMergePRCalls(cmd *MergeCmd) []mergePRCall {
	if mc, ok := cmd.github.(*mockGitHubClient); ok {
		return mc.mergePRCalls
	}
	return nil
}

func projectFindCompleteProjectsCalled(cmd *MergeCmd) bool {
	if mc, ok := cmd.project.(*mockProjectClient); ok {
		return mc.findCompleteProjectsCalled
	}
	return false
}

func projectRemoveAndCommitCalls(cmd *MergeCmd) [][]string {
	if mc, ok := cmd.project.(*mockProjectClient); ok {
		return mc.removeAndCommitCalls
	}
	return nil
}

func workflowSubmitMergeWorkflowCalls(cmd *MergeCmd) []string {
	if mc, ok := cmd.workflow.(*mockWorkflowClient); ok {
		return mc.submitMergeWorkflowCalls
	}
	return nil
}

var errMock = &mockError{"mock error"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
