package architecture

import (
	"github.com/zon/ralph/internal/architecture"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

// mockAIClient
type mockAIClient struct {
	buildArchitecturePromptFunc     func(string) (string, error)
	buildArchitectureFixPromptFunc  func(string, []string) (string, error)
	runAgentFunc                    func(string) error

	buildArchitecturePromptCalls []string
	buildArchitectureFixPromptCalls []buildFixCall
	runAgentCalls                []string
}

type buildFixCall struct {
	output string
	errors []string
}

func (m *mockAIClient) BuildArchitecturePrompt(output string) (string, error) {
	m.buildArchitecturePromptCalls = append(m.buildArchitecturePromptCalls, output)
	if m.buildArchitecturePromptFunc != nil {
		return m.buildArchitecturePromptFunc(output)
	}
	return "architecture prompt", nil
}

func (m *mockAIClient) BuildArchitectureFixPrompt(output string, errors []string) (string, error) {
	m.buildArchitectureFixPromptCalls = append(m.buildArchitectureFixPromptCalls, buildFixCall{output: output, errors: errors})
	if m.buildArchitectureFixPromptFunc != nil {
		return m.buildArchitectureFixPromptFunc(output, errors)
	}
	return "fix prompt", nil
}

func (m *mockAIClient) RunAgent(prompt string) error {
	m.runAgentCalls = append(m.runAgentCalls, prompt)
	if m.runAgentFunc != nil {
		return m.runAgentFunc(prompt)
	}
	return nil
}

// mockGitClient
type mockGitClient struct {
	isFileModifiedOrNewFunc  func(string) bool
	checkoutOrCreateBranchFunc func(string) error
	stageFileFunc            func(string) error
	commitAllAndPushFunc     func(*git.AuthConfig, string, string) error

	isFileModifiedOrNewCalls []string
	checkoutOrCreateBranchCalls []string
	stageFileCalls           []string
	commitAllAndPushCalls    []commitCall
}

type commitCall struct {
	auth       *git.AuthConfig
	branchName string
	commitMsg  string
}

func (m *mockGitClient) IsFileModifiedOrNew(path string) bool {
	m.isFileModifiedOrNewCalls = append(m.isFileModifiedOrNewCalls, path)
	if m.isFileModifiedOrNewFunc != nil {
		return m.isFileModifiedOrNewFunc(path)
	}
	return true
}

func (m *mockGitClient) CheckoutOrCreateBranch(name string) error {
	m.checkoutOrCreateBranchCalls = append(m.checkoutOrCreateBranchCalls, name)
	if m.checkoutOrCreateBranchFunc != nil {
		return m.checkoutOrCreateBranchFunc(name)
	}
	return nil
}

func (m *mockGitClient) StageFile(path string) error {
	m.stageFileCalls = append(m.stageFileCalls, path)
	if m.stageFileFunc != nil {
		return m.stageFileFunc(path)
	}
	return nil
}

func (m *mockGitClient) CommitAllAndPush(auth *git.AuthConfig, branchName, commitMsg string) error {
	m.commitAllAndPushCalls = append(m.commitAllAndPushCalls, commitCall{auth: auth, branchName: branchName, commitMsg: commitMsg})
	if m.commitAllAndPushFunc != nil {
		return m.commitAllAndPushFunc(auth, branchName, commitMsg)
	}
	return nil
}

// mockGitHubClient
type mockGitHubClient struct {
	createPullRequestFunc func(*project.Project, string, string, string) (string, error)

	createPullRequestCalls []prCall
}

type prCall struct {
	proj       *project.Project
	branchName string
	baseBranch string
	prSummary  string
}

func (m *mockGitHubClient) CreatePullRequest(proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
	m.createPullRequestCalls = append(m.createPullRequestCalls, prCall{proj: proj, branchName: branchName, baseBranch: baseBranch, prSummary: prSummary})
	if m.createPullRequestFunc != nil {
		return m.createPullRequestFunc(proj, branchName, baseBranch, prSummary)
	}
	return "https://github.com/owner/repo/pull/1", nil
}

// mockArchitectureClient
type mockArchitectureClient struct {
	loadFunc func(string) (*architecture.Architecture, error)

	loadCalls []string
}

func (m *mockArchitectureClient) Load(path string) (*architecture.Architecture, error) {
	m.loadCalls = append(m.loadCalls, path)
	if m.loadFunc != nil {
		return m.loadFunc(path)
	}
	return &architecture.Architecture{}, nil
}

// option types
type architectureOption func(*ArchitectureCmd)

func withAI(ac AIClient) architectureOption {
	return func(a *ArchitectureCmd) {
		a.ai = ac
	}
}

func withGit(gc GitClient) architectureOption {
	return func(a *ArchitectureCmd) {
		a.git = gc
	}
}

func withGitHub(gc GitHubClient) architectureOption {
	return func(a *ArchitectureCmd) {
		a.github = gc
	}
}

func withArchitecture(ac ArchitectureClient) architectureOption {
	return func(a *ArchitectureCmd) {
		a.archClient = ac
	}
}

func withMocks(opts ...architectureOption) *ArchitectureCmd {
	a := &ArchitectureCmd{
		ai:         &mockAIClient{},
		git:        &mockGitClient{},
		github:     &mockGitHubClient{},
		archClient: &mockArchitectureClient{},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Accessor helpers
func aiBuildArchitecturePromptCalls(cmd *ArchitectureCmd) []string {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.buildArchitecturePromptCalls
	}
	return nil
}

func aiBuildArchitectureFixPromptCalls(cmd *ArchitectureCmd) []buildFixCall {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.buildArchitectureFixPromptCalls
	}
	return nil
}

func aiRunAgentCalls(cmd *ArchitectureCmd) []string {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.runAgentCalls
	}
	return nil
}

func gitIsFileModifiedOrNewCalls(cmd *ArchitectureCmd) []string {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.isFileModifiedOrNewCalls
	}
	return nil
}

func gitCheckoutOrCreateBranchCalls(cmd *ArchitectureCmd) []string {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.checkoutOrCreateBranchCalls
	}
	return nil
}

func gitStageFileCalls(cmd *ArchitectureCmd) []string {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.stageFileCalls
	}
	return nil
}

func gitCommitAllAndPushCalls(cmd *ArchitectureCmd) []commitCall {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.commitAllAndPushCalls
	}
	return nil
}

func gitHubCreatePullRequestCalls(cmd *ArchitectureCmd) []prCall {
	if m, ok := cmd.github.(*mockGitHubClient); ok {
		return m.createPullRequestCalls
	}
	return nil
}

func archClientLoadCalls(cmd *ArchitectureCmd) []string {
	if m, ok := cmd.archClient.(*mockArchitectureClient); ok {
		return m.loadCalls
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
