package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
)

func newProjectThatReportsAllPassing() *project.MockClient {
	return &project.MockClient{
		AllPassingFunc: func() bool { return true },
	}
}

func newProjectThatReportsPassingAfterIterations(n int) *project.MockClient {
	calls := 0
	return &project.MockClient{
		AllPassingFunc: func() bool {
			calls++
			return calls > n
		},
	}
}

func newProjectThatAlwaysReportsFailures() *project.MockClient {
	return &project.MockClient{
		AllPassingFunc: func() bool { return false },
	}
}

// mockAgentClient cannot move to internal/run because that package already
// imports internal/orchestration/run, which would create an import cycle.
type mockAgentClient struct {
	pickFunc       func() (string, error)
	developFunc    func(string) error
	isFatalFunc    func(err error) bool
	changelogFunc  func() error
	pickCalls      []*project.Project
	developCalls   []*project.Project
	changelogCalls []*project.Project
}

func (m *mockAgentClient) Pick(proj *project.Project) (string, error) {
	m.pickCalls = append(m.pickCalls, proj)
	if m.pickFunc != nil {
		return m.pickFunc()
	}
	return "mock-requirement", nil
}

func (m *mockAgentClient) Develop(proj *project.Project, req string) error {
	m.developCalls = append(m.developCalls, proj)
	if m.developFunc != nil {
		return m.developFunc(req)
	}
	return nil
}

func (m *mockAgentClient) IsFatal(err error) bool {
	if m.isFatalFunc != nil {
		return m.isFatalFunc(err)
	}
	return false
}

func (m *mockAgentClient) GenerateChangelog(proj *project.Project) error {
	m.changelogCalls = append(m.changelogCalls, proj)
	if m.changelogFunc != nil {
		return m.changelogFunc()
	}
	return nil
}

func newAIThatAlwaysFails() *mockAgentClient {
	return &mockAgentClient{
		pickFunc:    func() (string, error) { return "", errNonFatal },
		isFatalFunc: func(err error) bool { return false },
	}
}

func newAIThatReturnsFatalError() *mockAgentClient {
	return &mockAgentClient{
		pickFunc:    func() (string, error) { return "", errFatal },
		isFatalFunc: func(err error) bool { return err == errFatal },
	}
}

func newAIThatReturnsNonFatalError() *mockAgentClient {
	return &mockAgentClient{
		pickFunc:    func() (string, error) { return "", errNonFatal },
		isFatalFunc: func(err error) bool { return false },
	}
}

func newGitWithChangesAndReport() *git.MockClient {
	return &git.MockClient{
		HasChangesFunc:   func() bool { return true },
		ReportExistsFunc: func() bool { return true },
	}
}

func newGitWithChangesButNoReport() *git.MockClient {
	return &git.MockClient{
		HasChangesFunc:   func() bool { return true },
		ReportExistsFunc: func() bool { return false },
	}
}

func newGitWithNoChanges() *git.MockClient {
	return &git.MockClient{
		HasChangesFunc:   func() bool { return false },
		ReportExistsFunc: func() bool { return false },
	}
}

func newGitWithBlockedFile() *git.MockClient {
	return &git.MockClient{
		BlockedFileExistsFunc: func() bool { return true },
	}
}

func newGitHubWithCommitsAhead() *github.MockClient {
	return &github.MockClient{
		CreatePRFunc: func(_ *project.Project) error { return nil },
	}
}

func newServicesThatFailBeforeCommands() *services.MockClient {
	return &services.MockClient{
		RunBeforeFunc: func(_ *config.RalphConfig) error { return errServiceFailure },
	}
}

var errNonFatal = &mockError{"non-fatal error"}
var errFatal = &mockError{"billing limit exceeded"}
var errServiceFailure = &mockError{"service failure"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

type runnerOption func(*Runner)

func withProject(pc ProjectClient) runnerOption {
	return func(r *Runner) {
		r.project = pc
	}
}

func withAI(ac AgentClient) runnerOption {
	return func(r *Runner) {
		r.ai = ac
	}
}

func withGit(gc GitClient) runnerOption {
	return func(r *Runner) {
		r.git = gc
	}
}

func withGitHub(gc GitHubClient) runnerOption {
	return func(r *Runner) {
		r.github = gc
	}
}

func withServices(sc ServicesClient) runnerOption {
	return func(r *Runner) {
		r.services = sc
	}
}

func withMocks(opts ...runnerOption) *Runner {
	r := &Runner{
		project:  newProjectThatAlwaysReportsFailures(),
		ai:       &mockAgentClient{},
		git:      &git.MockClient{},
		github:   &github.MockClient{},
		services: &services.MockClient{},
		notify:   &notify.MockClient{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func aiPickCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAgentClient); ok {
		return m.pickCalls
	}
	return nil
}

func aiDevelopCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAgentClient); ok {
		return m.developCalls
	}
	return nil
}

func aiChangelogCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAgentClient); ok {
		return m.changelogCalls
	}
	return nil
}

func gitBranchSwitched(r *Runner) bool {
	if m, ok := r.git.(*git.MockClient); ok {
		return m.SwitchToBranchCalled
	}
	return false
}

func gitBlockedFileWritten(r *Runner) bool {
	if m, ok := r.git.(*git.MockClient); ok {
		return m.WriteBlockedFileCalled
	}
	return false
}

func gitCommittedFromReport(r *Runner) bool {
	if m, ok := r.git.(*git.MockClient); ok {
		return m.CommitFromReportCalled
	}
	return false
}

func githubPRCreated(r *Runner) bool {
	if m, ok := r.github.(*github.MockClient); ok {
		return m.CreatePRCalled && m.CreatePRReturnedNil
	}
	return false
}

func notifyErrors(r *Runner) []string {
	if m, ok := r.notify.(*notify.MockClient); ok {
		return m.ErrorsSlice
	}
	return nil
}

func notifySuccesses(r *Runner) []string {
	if m, ok := r.notify.(*notify.MockClient); ok {
		return m.SuccessesSlice
	}
	return nil
}

type remoteRunnerOption func(*RemoteRunner)

func withRemoteGit(gc GitClient) remoteRunnerOption {
	return func(r *RemoteRunner) { r.git = gc }
}

func withRemoteWorkflow(wc WorkflowClient) remoteRunnerOption {
	return func(r *RemoteRunner) { r.workflow = wc }
}

func withRemoteMocks(opts ...remoteRunnerOption) *RemoteRunner {
	r := &RemoteRunner{
		git:      &git.MockClient{},
		workflow: &workflow.MockClient{},
		notify:   &notify.MockClient{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func remoteWorkflowSubmitted(runner *RemoteRunner) bool {
	if m, ok := runner.workflow.(*workflow.MockClient); ok {
		return m.SubmitCalled
	}
	return false
}

func remoteWorkflowLogHintPrinted(runner *RemoteRunner) bool {
	if m, ok := runner.workflow.(*workflow.MockClient); ok {
		return m.PrintLogHintCalled
	}
	return false
}

func remoteNotifySuccesses(runner *RemoteRunner) []string {
	if m, ok := runner.notify.(*notify.MockClient); ok {
		return m.SuccessesSlice
	}
	return nil
}

func remoteNotifyErrors(runner *RemoteRunner) []string {
	if m, ok := runner.notify.(*notify.MockClient); ok {
		return m.ErrorsSlice
	}
	return nil
}
