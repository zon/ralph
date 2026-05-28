package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// mockProjectClient implements ProjectClient with configurable behavior.
type mockProjectClient struct {
	allPassingFunc func() bool
}

func (m *mockProjectClient) AllRequirementsPassing(_ *project.Project) bool {
	return m.allPassingFunc()
}

func (m *mockProjectClient) MaxIterationsError(_ *project.Project) error {
	return project.ErrMaxIterationsReached
}

func newProjectThatReportsAllPassing() *mockProjectClient {
	return &mockProjectClient{
		allPassingFunc: func() bool { return true },
	}
}

func newProjectThatReportsPassingAfterIterations(n int) *mockProjectClient {
	calls := 0
	return &mockProjectClient{
		allPassingFunc: func() bool {
			calls++
			return calls > n
		},
	}
}

func newProjectThatAlwaysReportsFailures() *mockProjectClient {
	return &mockProjectClient{
		allPassingFunc: func() bool { return false },
	}
}

// mockAgentClient implements AgentClient with configurable behavior and call recording.
type mockAgentClient struct {
	iterateFunc     func() error
	isFatalFunc     func(err error) bool
	changelogFunc   func() error
	iterateCalls    []*project.Project
	changelogCalls  []*project.Project
}

func (m *mockAgentClient) Iterate(proj *project.Project) error {
	m.iterateCalls = append(m.iterateCalls, proj)
	if m.iterateFunc != nil {
		return m.iterateFunc()
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
		iterateFunc: func() error { return errNonFatal },
		isFatalFunc: func(err error) bool { return false },
	}
}

func newAIThatReturnsFatalError() *mockAgentClient {
	return &mockAgentClient{
		iterateFunc: func() error { return errFatal },
		isFatalFunc: func(err error) bool { return err == errFatal },
	}
}

func newAIThatReturnsNonFatalError() *mockAgentClient {
	return &mockAgentClient{
		iterateFunc: func() error { return errNonFatal },
		isFatalFunc: func(err error) bool { return false },
	}
}

// mockGitClient implements GitClient with configurable behavior and call recording.
type mockGitClient struct {
	switchToBranchFunc    func(slug string) error
	blockedFileExistsFunc func() bool
	writeBlockedFileFunc  func(err error)
	hasChangesFunc        func() bool
	reportExistsFunc      func() bool
	commitFromReportFunc  func(slug string) error

	switchToBranchCalled   bool
	writeBlockedFileCalled bool
	commitFromReportCalled bool
}

func (m *mockGitClient) SwitchToBranch(slug string) error {
	m.switchToBranchCalled = true
	if m.switchToBranchFunc != nil {
		return m.switchToBranchFunc(slug)
	}
	return nil
}

func (m *mockGitClient) BlockedFileExists() bool {
	if m.blockedFileExistsFunc != nil {
		return m.blockedFileExistsFunc()
	}
	return false
}

func (m *mockGitClient) WriteBlockedFile(err error) {
	m.writeBlockedFileCalled = true
	if m.writeBlockedFileFunc != nil {
		m.writeBlockedFileFunc(err)
	}
}

func (m *mockGitClient) HasChanges() bool {
	if m.hasChangesFunc != nil {
		return m.hasChangesFunc()
	}
	return false
}

func (m *mockGitClient) ReportExists() bool {
	if m.reportExistsFunc != nil {
		return m.reportExistsFunc()
	}
	return false
}

func (m *mockGitClient) CommitFromReport(slug string) error {
	m.commitFromReportCalled = true
	if m.commitFromReportFunc != nil {
		return m.commitFromReportFunc(slug)
	}
	return nil
}

func newGitWithChangesAndReport() *mockGitClient {
	return &mockGitClient{
		hasChangesFunc:   func() bool { return true },
		reportExistsFunc: func() bool { return true },
	}
}

func newGitWithChangesButNoReport() *mockGitClient {
	return &mockGitClient{
		hasChangesFunc:   func() bool { return true },
		reportExistsFunc: func() bool { return false },
	}
}

func newGitWithNoChanges() *mockGitClient {
	return &mockGitClient{
		hasChangesFunc:   func() bool { return false },
		reportExistsFunc: func() bool { return false },
	}
}

func newGitWithBlockedFile() *mockGitClient {
	return &mockGitClient{
		blockedFileExistsFunc: func() bool { return true },
	}
}

// mockGitHubClient implements GitHubClient with configurable behavior and call recording.
type mockGitHubClient struct {
	createPRFunc        func(*project.Project) error
	createPRCalled      bool
	createPRReturnedNil bool
}

func (m *mockGitHubClient) CreatePR(proj *project.Project) error {
	m.createPRCalled = true
	if m.createPRFunc != nil {
		err := m.createPRFunc(proj)
		if err == nil {
			m.createPRReturnedNil = true
		}
		return err
	}
	m.createPRReturnedNil = true
	return nil
}

func newGitHubWithCommitsAhead() *mockGitHubClient {
	return &mockGitHubClient{
		createPRFunc: func(_ *project.Project) error { return nil },
	}
}

// mockServicesClient implements ServicesClient with configurable behavior.
type mockServicesClient struct {
	runBeforeFunc func(cfg *config.RalphConfig) error
}

func (m *mockServicesClient) RunBeforeCommands(cfg *config.RalphConfig) error {
	if m.runBeforeFunc != nil {
		return m.runBeforeFunc(cfg)
	}
	return nil
}

func newServicesThatFailBeforeCommands() *mockServicesClient {
	return &mockServicesClient{
		runBeforeFunc: func(_ *config.RalphConfig) error { return errServiceFailure },
	}
}

// mockNotifyClient implements NotifyClient with call recording.
type mockNotifyClient struct {
	errorsSlice    []string
	successesSlice []string
	errorFunc      func(slug string)
	successFunc    func(slug string)
}

func (m *mockNotifyClient) Error(slug string) {
	m.errorsSlice = append(m.errorsSlice, slug)
	if m.errorFunc != nil {
		m.errorFunc(slug)
	}
}

func (m *mockNotifyClient) Success(slug string) {
	m.successesSlice = append(m.successesSlice, slug)
	if m.successFunc != nil {
		m.successFunc(slug)
	}
}

// test errors
var errNonFatal = &mockError{"non-fatal error"}
var errFatal = &mockError{"billing limit exceeded"}
var errServiceFailure = &mockError{"service failure"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// runnerOption is a function that configures a Runner's clients.
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
	mockAI := &mockAgentClient{}
	mockGit := &mockGitClient{}
	mockGH := &mockGitHubClient{}

	r := &Runner{
		project:  newProjectThatAlwaysReportsFailures(),
		ai:       mockAI,
		git:      mockGit,
		github:   mockGH,
		services: &mockServicesClient{},
		notify:   &mockNotifyClient{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// spy accessors — these work on the concrete mock types embedded in the Runner.
func aiIterateCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAgentClient); ok {
		return m.iterateCalls
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
	if m, ok := r.git.(*mockGitClient); ok {
		return m.switchToBranchCalled
	}
	return false
}

func gitBlockedFileWritten(r *Runner) bool {
	if m, ok := r.git.(*mockGitClient); ok {
		return m.writeBlockedFileCalled
	}
	return false
}

func gitCommittedFromReport(r *Runner) bool {
	if m, ok := r.git.(*mockGitClient); ok {
		return m.commitFromReportCalled
	}
	return false
}

func githubPRCreated(r *Runner) bool {
	if m, ok := r.github.(*mockGitHubClient); ok {
		return m.createPRCalled && m.createPRReturnedNil
	}
	return false
}

func notifyErrors(r *Runner) []string {
	if m, ok := r.notify.(*mockNotifyClient); ok {
		return m.errorsSlice
	}
	return nil
}

func notifySuccesses(r *Runner) []string {
	if m, ok := r.notify.(*mockNotifyClient); ok {
		return m.successesSlice
	}
	return nil
}
