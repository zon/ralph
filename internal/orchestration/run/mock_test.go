package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

func newProjectThatReportsAllPassing() *project.MockRunAdapter {
	return &project.MockRunAdapter{
		AllPassingFunc: func() bool { return true },
	}
}

func newProjectThatReportsPassingAfterIterations(n int) *project.MockRunAdapter {
	calls := 0
	return &project.MockRunAdapter{
		AllPassingFunc: func() bool {
			calls++
			return calls > n
		},
	}
}

func newProjectThatAlwaysReportsFailures() *project.MockRunAdapter {
	return &project.MockRunAdapter{
		AllPassingFunc: func() bool { return false },
	}
}

// mockAgentClient cannot move to internal/run because that package already
// imports internal/orchestration/run, which would create an import cycle.
type mockAgentClient struct {
	iterateFunc    func() error
	isFatalFunc    func(err error) bool
	changelogFunc  func() error
	iterateCalls   []*project.Project
	changelogCalls []*project.Project
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

func newGitWithChangesAndReport() *git.MockRunAdapter {
	return &git.MockRunAdapter{
		HasChangesFunc:   func() bool { return true },
		ReportExistsFunc: func() bool { return true },
	}
}

func newGitWithChangesButNoReport() *git.MockRunAdapter {
	return &git.MockRunAdapter{
		HasChangesFunc:   func() bool { return true },
		ReportExistsFunc: func() bool { return false },
	}
}

func newGitWithNoChanges() *git.MockRunAdapter {
	return &git.MockRunAdapter{
		HasChangesFunc:   func() bool { return false },
		ReportExistsFunc: func() bool { return false },
	}
}

func newGitWithBlockedFile() *git.MockRunAdapter {
	return &git.MockRunAdapter{
		BlockedFileExistsFunc: func() bool { return true },
	}
}

func newGitHubWithCommitsAhead() *github.MockRunAdapter {
	return &github.MockRunAdapter{
		CreatePRFunc: func(_ *project.Project) error { return nil },
	}
}

func newServicesThatFailBeforeCommands() *services.MockRunAdapter {
	return &services.MockRunAdapter{
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
		git:      &git.MockRunAdapter{},
		github:   &github.MockRunAdapter{},
		services: &services.MockRunAdapter{},
		notify:   &notify.MockRunAdapter{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

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
	if m, ok := r.git.(*git.MockRunAdapter); ok {
		return m.SwitchToBranchCalled
	}
	return false
}

func gitBlockedFileWritten(r *Runner) bool {
	if m, ok := r.git.(*git.MockRunAdapter); ok {
		return m.WriteBlockedFileCalled
	}
	return false
}

func gitCommittedFromReport(r *Runner) bool {
	if m, ok := r.git.(*git.MockRunAdapter); ok {
		return m.CommitFromReportCalled
	}
	return false
}

func githubPRCreated(r *Runner) bool {
	if m, ok := r.github.(*github.MockRunAdapter); ok {
		return m.CreatePRCalled && m.CreatePRReturnedNil
	}
	return false
}

func notifyErrors(r *Runner) []string {
	if m, ok := r.notify.(*notify.MockRunAdapter); ok {
		return m.ErrorsSlice
	}
	return nil
}

func notifySuccesses(r *Runner) []string {
	if m, ok := r.notify.(*notify.MockRunAdapter); ok {
		return m.SuccessesSlice
	}
	return nil
}
