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

// mockAIClient cannot move to internal/run because that package already
// imports internal/orchestration/run, which would create an import cycle.
type mockAIClient struct {
	runPickerFunc    func() (string, error)
	runDeveloperFunc func(string) error
	isFatalFunc      func(err error) bool
	changelogFunc    func() error
	fixServiceFunc   func(*config.RalphConfig, error) error
	pickCalls        []*project.Project
	developCalls     []*project.Project
	changelogCalls   []*project.Project
	fixServiceCalled bool
	statsPrinted     bool
}

func (m *mockAIClient) RunPicker(proj *project.Project) (string, error) {
	m.pickCalls = append(m.pickCalls, proj)
	if m.runPickerFunc != nil {
		return m.runPickerFunc()
	}
	return "mock-requirement", nil
}

func (m *mockAIClient) RunDeveloper(proj *project.Project, req string) error {
	m.developCalls = append(m.developCalls, proj)
	if m.runDeveloperFunc != nil {
		return m.runDeveloperFunc(req)
	}
	return nil
}

func (m *mockAIClient) IsFatal(err error) bool {
	if m.isFatalFunc != nil {
		return m.isFatalFunc(err)
	}
	return false
}

func (m *mockAIClient) GenerateChangelog(proj *project.Project) error {
	m.changelogCalls = append(m.changelogCalls, proj)
	if m.changelogFunc != nil {
		return m.changelogFunc()
	}
	return nil
}

func (m *mockAIClient) PrintStats() {
	m.statsPrinted = true
}

func (m *mockAIClient) FixServiceStartup(cfg *config.RalphConfig, err error) error {
	m.fixServiceCalled = true
	if m.fixServiceFunc != nil {
		return m.fixServiceFunc(cfg, err)
	}
	return nil
}

type mockEnvClient struct {
	inWorkflow bool
}

func (m *mockEnvClient) InWorkflow() bool {
	return m.inWorkflow
}

func newEnvInWorkflow() *mockEnvClient {
	return &mockEnvClient{inWorkflow: true}
}

func newEnvNotInWorkflow() *mockEnvClient {
	return &mockEnvClient{inWorkflow: false}
}

func newAIThatAlwaysFails() *mockAIClient {
	return &mockAIClient{
		runPickerFunc: func() (string, error) { return "", errNonFatal },
		isFatalFunc:   func(err error) bool { return false },
	}
}

func newAIThatReturnsFatalError() *mockAIClient {
	return &mockAIClient{
		runPickerFunc: func() (string, error) { return "", errFatal },
		isFatalFunc:   func(err error) bool { return err == errFatal },
	}
}

func newAIThatReturnsNonFatalError() *mockAIClient {
	return &mockAIClient{
		runPickerFunc: func() (string, error) { return "", errNonFatal },
		isFatalFunc:   func(err error) bool { return false },
	}
}

func newAIThatFailsServiceFix() *mockAIClient {
	return &mockAIClient{
		fixServiceFunc: func(_ *config.RalphConfig, _ error) error { return errServiceFailure },
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

type mockServicesClient struct {
	startCount      int
	stopCount       int
	removeLogsCount int
	startFunc       func() error
}

func (m *mockServicesClient) RunBeforeCommands(_ *config.RalphConfig) error { return nil }

func (m *mockServicesClient) Start(_ *config.RalphConfig) (*services.Manager, error) {
	m.startCount++
	if m.startFunc != nil {
		return nil, m.startFunc()
	}
	return &services.Manager{}, nil
}

func (m *mockServicesClient) Stop(_ *services.Manager) {
	m.stopCount++
}

func (m *mockServicesClient) RemoveLogs(_ *config.RalphConfig) {
	m.removeLogsCount++
}

func newServicesThatFailToStart() *mockServicesClient {
	return &mockServicesClient{
		startFunc: func() error { return errServiceFailure },
	}
}

func failingProject() *project.Project {
	return project.WithFailingRequirements()
}

func passingProject() *project.Project {
	return project.WithAllPassing()
}

func anyConfig() *config.RalphConfig {
	return config.Any()
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

func withAI(ac AIClient) runnerOption {
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

func withEnv(ec EnvClient) runnerOption {
	return func(r *Runner) {
		r.env = ec
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
		ai:       &mockAIClient{},
		git:      &git.MockClient{},
		github:   &github.MockClient{},
		services: &services.MockClient{},
		notify:   &notify.MockClient{},
		env:      newEnvNotInWorkflow(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func aiStatsPrinted(r *Runner) bool {
	if m, ok := r.ai.(*mockAIClient); ok {
		return m.statsPrinted
	}
	return false
}

func aiPickCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAIClient); ok {
		return m.pickCalls
	}
	return nil
}

func aiDevelopCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAIClient); ok {
		return m.developCalls
	}
	return nil
}

func aiChangelogCalls(r *Runner) []*project.Project {
	if m, ok := r.ai.(*mockAIClient); ok {
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
