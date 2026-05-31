package review

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// mockAIClient
type mockAIClient struct {
	buildReviewItemPromptFunc  func(string) (string, error)
	buildLoopItemPromptFunc    func(string, string, string) (string, error)
	runAgentFunc               func(string) error
	displayStatsFunc           func() error
	generateReviewPRBodyFunc   func(string, string, []string) (string, error)
	setModelFunc               func(string)

	model string

	runAgentCalls   []string
	displayStatsCalled bool
	prBodyCalls     []prBodyCall
}

type prBodyCall struct {
	slug               string
	title              string
	requirementSummaries []string
}

func (m *mockAIClient) BuildReviewItemPrompt(content string) (string, error) {
	if m.buildReviewItemPromptFunc != nil {
		return m.buildReviewItemPromptFunc(content)
	}
	return "mock-review-prompt-" + content, nil
}

func (m *mockAIClient) BuildLoopItemPrompt(content, funcName, funcPath string) (string, error) {
	if m.buildLoopItemPromptFunc != nil {
		return m.buildLoopItemPromptFunc(content, funcName, funcPath)
	}
	return "mock-loop-prompt-" + content, nil
}

func (m *mockAIClient) RunAgent(prompt string) error {
	m.runAgentCalls = append(m.runAgentCalls, prompt)
	if m.runAgentFunc != nil {
		return m.runAgentFunc(prompt)
	}
	return nil
}

func (m *mockAIClient) DisplayStats() error {
	m.displayStatsCalled = true
	if m.displayStatsFunc != nil {
		return m.displayStatsFunc()
	}
	return nil
}

func (m *mockAIClient) GenerateReviewPRBody(slug, title string, requirementSummaries []string) (string, error) {
	m.prBodyCalls = append(m.prBodyCalls, prBodyCall{slug: slug, title: title, requirementSummaries: requirementSummaries})
	if m.generateReviewPRBodyFunc != nil {
		return m.generateReviewPRBodyFunc(slug, title, requirementSummaries)
	}
	return "mock-pr-body", nil
}

func (m *mockAIClient) SetModel(model string) {
	m.model = model
	if m.setModelFunc != nil {
		m.setModelFunc(model)
	}
}

// mockGitClient
type mockGitClient struct {
	currentBranchFunc            func() (string, error)
	hasUncommittedChangesFunc    func() bool
	commitAllAndPushFunc         func(string, string) error
	detectModifiedProjectFileFunc func(string) (string, error)
	isBranchSyncedWithRemoteFunc func(string) error
	tmpPathFunc                  func(string) (string, error)

	currentBranchCalled          bool
	commitAllAndPushCalls        []commitCall
	detectModifiedProjectFileCalled bool
}

type commitCall struct {
	branch    string
	commitMsg string
}

func (m *mockGitClient) CurrentBranch() (string, error) {
	m.currentBranchCalled = true
	if m.currentBranchFunc != nil {
		return m.currentBranchFunc()
	}
	return "main", nil
}

func (m *mockGitClient) HasUncommittedChanges() bool {
	if m.hasUncommittedChangesFunc != nil {
		return m.hasUncommittedChangesFunc()
	}
	return false
}

func (m *mockGitClient) CommitAllAndPush(branch, commitMsg string) error {
	m.commitAllAndPushCalls = append(m.commitAllAndPushCalls, commitCall{branch: branch, commitMsg: commitMsg})
	if m.commitAllAndPushFunc != nil {
		return m.commitAllAndPushFunc(branch, commitMsg)
	}
	return nil
}

func (m *mockGitClient) DetectModifiedProjectFile(dir string) (string, error) {
	m.detectModifiedProjectFileCalled = true
	if m.detectModifiedProjectFileFunc != nil {
		return m.detectModifiedProjectFileFunc(dir)
	}
	return "", nil
}

func (m *mockGitClient) IsBranchSyncedWithRemote(branch string) error {
	if m.isBranchSyncedWithRemoteFunc != nil {
		return m.isBranchSyncedWithRemoteFunc(branch)
	}
	return nil
}

func (m *mockGitClient) TmpPath(filename string) (string, error) {
	if m.tmpPathFunc != nil {
		return m.tmpPathFunc(filename)
	}
	return "/tmp/" + filename, nil
}

// mockGitHubClient
type mockGitHubClient struct {
	createPullRequestFunc func(*project.Project, string, string, string) (string, error)
	createPRCall          *createPRCallData
}

type createPRCallData struct {
	proj       *project.Project
	reviewName string
	baseBranch string
	body       string
}

func (m *mockGitHubClient) CreatePullRequest(proj *project.Project, reviewName, baseBranch, body string) (string, error) {
	m.createPRCall = &createPRCallData{proj: proj, reviewName: reviewName, baseBranch: baseBranch, body: body}
	if m.createPullRequestFunc != nil {
		return m.createPullRequestFunc(proj, reviewName, baseBranch, body)
	}
	return "https://github.com/owner/repo/pull/1", nil
}

// mockWorkflowClient
type mockWorkflowClient struct {
	submitReviewFunc func(string) (string, error)
	followLogsFunc   func(string) error

	submitReviewCalls []string
	followLogsCalled  bool
}

func (m *mockWorkflowClient) SubmitReview(cloneBranch string) (string, error) {
	m.submitReviewCalls = append(m.submitReviewCalls, cloneBranch)
	if m.submitReviewFunc != nil {
		return m.submitReviewFunc(cloneBranch)
	}
	return "test-workflow", nil
}

func (m *mockWorkflowClient) FollowLogs(workflowName string) error {
	m.followLogsCalled = true
	if m.followLogsFunc != nil {
		return m.followLogsFunc(workflowName)
	}
	return nil
}

// option types
type reviewOption func(*ReviewCmd)

func withAI(ac AIClient) reviewOption {
	return func(r *ReviewCmd) {
		r.ai = ac
	}
}

func withGit(gc GitClient) reviewOption {
	return func(r *ReviewCmd) {
		r.git = gc
	}
}

func withGitHub(gc GitHubClient) reviewOption {
	return func(r *ReviewCmd) {
		r.github = gc
	}
}

func withWorkflow(wc WorkflowClient) reviewOption {
	return func(r *ReviewCmd) {
		r.workflow = wc
	}
}

func withMocks(opts ...reviewOption) *ReviewCmd {
	r := &ReviewCmd{
		ai:       &mockAIClient{},
		git:      &mockGitClient{},
		github:   &mockGitHubClient{},
		workflow: &mockWorkflowClient{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Accessor helpers
func aiRunAgentCalls(cmd *ReviewCmd) []string {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.runAgentCalls
	}
	return nil
}

func aiDisplayStatsCalled(cmd *ReviewCmd) bool {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.displayStatsCalled
	}
	return false
}

func aiModel(cmd *ReviewCmd) string {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.model
	}
	return ""
}

func gitCurrentBranchCalled(cmd *ReviewCmd) bool {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.currentBranchCalled
	}
	return false
}

func gitDetectModifiedProjectFileCalled(cmd *ReviewCmd) bool {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.detectModifiedProjectFileCalled
	}
	return false
}

func gitCommitAllAndPushCalls(cmd *ReviewCmd) []commitCall {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.commitAllAndPushCalls
	}
	return nil
}

func workflowSubmitReviewCalls(cmd *ReviewCmd) []string {
	if m, ok := cmd.workflow.(*mockWorkflowClient); ok {
		return m.submitReviewCalls
	}
	return nil
}

func workflowFollowLogsCalled(cmd *ReviewCmd) bool {
	if m, ok := cmd.workflow.(*mockWorkflowClient); ok {
		return m.followLogsCalled
	}
	return false
}

func gitHubCreatePRCall(cmd *ReviewCmd) *createPRCallData {
	if m, ok := cmd.github.(*mockGitHubClient); ok {
		return m.createPRCall
	}
	return nil
}

// mock config helpers
func anyConfig() *config.RalphConfig {
	return &config.RalphConfig{
		Model: "deepseek/deepseek-chat",
		Review: config.ReviewConfig{
			Items: []config.ReviewItem{
				{Text: "test review item"},
			},
		},
	}
}

func configWithModel(model string) *config.RalphConfig {
	cfg := anyConfig()
	cfg.Model = model
	return cfg
}

func configWithReviewModel(model string) *config.RalphConfig {
	cfg := anyConfig()
	cfg.Review.Model = model
	return cfg
}

func configWithItems(items []config.ReviewItem) *config.RalphConfig {
	return &config.RalphConfig{
		Model: "deepseek/deepseek-chat",
		Review: config.ReviewConfig{
			Items: items,
		},
	}
}

var errMock = &mockError{"mock error"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
