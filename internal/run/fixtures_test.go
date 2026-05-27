package run

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	proj "github.com/zon/ralph/internal/project"
)

// ---- Mock implementations ----

type mockProjectClient struct {
	allPassingFunc func() bool
}

func (m *mockProjectClient) AllRequirementsPassing(p *proj.Project) bool {
	if m.allPassingFunc != nil {
		return m.allPassingFunc()
	}
	return false
}

func (m *mockProjectClient) MaxIterationsError(p *proj.Project) error {
	return fmt.Errorf("%w: 0 requirements still failing", ErrMaxIterationsReached)
}

type mockAgentClient struct {
	iterateFunc   func() error
	isFatalFunc   func(err error) bool
	changelogFunc func() error

	iterateCalls      int
	changelogCalls    int
	iterateProjects   []*proj.Project
	changelogProjects []*proj.Project
}

func newMockAgentClient() *mockAgentClient {
	return &mockAgentClient{
		iterateFunc:   func() error { return nil },
		isFatalFunc:   func(err error) bool { return false },
		changelogFunc: func() error { return nil },
	}
}

func (m *mockAgentClient) Iterate(p *proj.Project) error {
	m.iterateCalls++
	m.iterateProjects = append(m.iterateProjects, p)
	return m.iterateFunc()
}

func (m *mockAgentClient) IsFatal(err error) bool {
	return m.isFatalFunc(err)
}

func (m *mockAgentClient) GenerateChangelog(p *proj.Project) error {
	m.changelogCalls++
	m.changelogProjects = append(m.changelogProjects, p)
	return m.changelogFunc()
}

type mockGitClient struct {
	hasChangesVal        bool
	reportExistsVal      bool
	blockedFileExistsVal bool

	committedFromReport int
	branchSwitched      int
	blockedFileWritten  bool
}

func (m *mockGitClient) SwitchToBranch(slug string) error {
	m.branchSwitched++
	return nil
}

func (m *mockGitClient) BlockedFileExists() bool {
	return m.blockedFileExistsVal
}

func (m *mockGitClient) WriteBlockedFile(err error) {
	m.blockedFileWritten = true
}

func (m *mockGitClient) HasChanges() bool {
	return m.hasChangesVal
}

func (m *mockGitClient) ReportExists() bool {
	return m.reportExistsVal
}

func (m *mockGitClient) CommitFromReport(slug string) error {
	m.committedFromReport++
	return nil
}

type mockGitHubClient struct {
	prCreated int
}

func (m *mockGitHubClient) CreatePR(p *proj.Project) error {
	m.prCreated++
	return nil
}

type mockServicesClient struct {
	beforeErr error
}

func (m *mockServicesClient) RunBeforeCommands(cfg *config.RalphConfig) error {
	return m.beforeErr
}

type mockNotifyClient struct {
	errorSlugs   []string
	successSlugs []string
}

func (m *mockNotifyClient) Error(slug string) {
	m.errorSlugs = append(m.errorSlugs, slug)
}

func (m *mockNotifyClient) Success(slug string) {
	m.successSlugs = append(m.successSlugs, slug)
}

// ---- withMocks infrastructure ----

type mockSet struct {
	project  ProjectClient
	git      GitClient
	ai       AgentClient
	github   GitHubClient
	services ServicesClient
	notify   NotifyClient
}

func withMocks(opts ...func(*mockSet)) *Runner {
	m := &mockSet{}
	for _, opt := range opts {
		opt(m)
	}
	if m.ai == nil {
		m.ai = newMockAgentClient()
	}
	if m.git == nil {
		m.git = &mockGitClient{}
	}
	if m.project == nil {
		m.project = &mockProjectClient{}
	}
	if m.github == nil {
		m.github = &mockGitHubClient{}
	}
	if m.services == nil {
		m.services = &mockServicesClient{}
	}
	if m.notify == nil {
		m.notify = &mockNotifyClient{}
	}
	return &Runner{
		project:  m.project,
		ai:       m.ai,
		git:      m.git,
		github:   m.github,
		services: m.services,
		notify:   m.notify,
	}
}

func withProject(pc ProjectClient) func(*mockSet) {
	return func(m *mockSet) {
		m.project = pc
	}
}

func withAI(ac AgentClient) func(*mockSet) {
	return func(m *mockSet) {
		m.ai = ac
	}
}

func withGit(gc GitClient) func(*mockSet) {
	return func(m *mockSet) {
		m.git = gc
	}
}

func withServices(sc ServicesClient) func(*mockSet) {
	return func(m *mockSet) {
		m.services = sc
	}
}

// ---- Helper to extract mock from Runner ----

func agentCalls(r *Runner) (iterateCount, changelogCount int) {
	m, ok := r.ai.(*mockAgentClient)
	if !ok {
		return 0, 0
	}
	return m.iterateCalls, m.changelogCalls
}

func iterateCalls(r *Runner) []*proj.Project {
	m, ok := r.ai.(*mockAgentClient)
	if !ok {
		return nil
	}
	return m.iterateProjects
}

func changelogCalls(r *Runner) []*proj.Project {
	m, ok := r.ai.(*mockAgentClient)
	if !ok {
		return nil
	}
	return m.changelogProjects
}

func hasCommitted(r *Runner) bool {
	m, ok := r.git.(*mockGitClient)
	if !ok {
		return false
	}
	return m.committedFromReport > 0
}

func hasSwitchedBranch(r *Runner) bool {
	m, ok := r.git.(*mockGitClient)
	if !ok {
		return false
	}
	return m.branchSwitched > 0
}

func hasWrittenBlocked(r *Runner) bool {
	m, ok := r.git.(*mockGitClient)
	if !ok {
		return false
	}
	return m.blockedFileWritten
}

func hasCreatedPR(r *Runner) bool {
	m, ok := r.github.(*mockGitHubClient)
	if !ok {
		return false
	}
	return m.prCreated > 0
}

func notifyErrors(r *Runner) []string {
	m, ok := r.notify.(*mockNotifyClient)
	if !ok {
		return nil
	}
	return m.errorSlugs
}

func notifySuccesses(r *Runner) []string {
	m, ok := r.notify.(*mockNotifyClient)
	if !ok {
		return nil
	}
	return m.successSlugs
}

// ---- Project mock factories ----

func thatReportsAllPassing() ProjectClient {
	return &mockProjectClient{
		allPassingFunc: func() bool { return true },
	}
}

func thatReportsPassingAfterIterations(n int) ProjectClient {
	callCount := 0
	return &mockProjectClient{
		allPassingFunc: func() bool {
			callCount++
			return callCount > n
		},
	}
}

func thatAlwaysReportsFailures() ProjectClient {
	return &mockProjectClient{
		allPassingFunc: func() bool { return false },
	}
}

func anyProject() *proj.Project {
	return &proj.Project{
		Slug: "test-project",
		Requirements: []proj.Requirement{
			{Slug: "req-1", Items: []string{"item-1"}, Passing: false},
		},
		MaxIterations: 1,
	}
}

func withAllPassing() *proj.Project {
	return &proj.Project{
		Slug: "test-project",
		Requirements: []proj.Requirement{
			{Slug: "req-1", Items: []string{"item-1"}, Passing: true},
		},
		MaxIterations: 10,
	}
}

func withFailingRequirements() *proj.Project {
	return &proj.Project{
		Slug: "test-project",
		Requirements: []proj.Requirement{
			{Slug: "req-1", Items: []string{"item-1"}, Passing: false},
		},
		MaxIterations: 10,
	}
}

func withMaxIterations(n int) *proj.Project {
	return &proj.Project{
		Slug: "test-project",
		Requirements: []proj.Requirement{
			{Slug: "req-1", Items: []string{"item-1"}, Passing: false},
		},
		MaxIterations: n,
	}
}

// ---- Config ----

func anyConfig() *config.RalphConfig {
	return &config.RalphConfig{}
}

// ---- Git mock factories ----

func withCommitsAhead() GitClient {
	return &mockGitClient{
		hasChangesVal:   true,
		reportExistsVal: false,
	}
}

func withBlockedFile() GitClient {
	return &mockGitClient{
		blockedFileExistsVal: true,
	}
}

func withChangesAndReport() GitClient {
	return &mockGitClient{
		hasChangesVal:   true,
		reportExistsVal: true,
	}
}

func withChangesButNoReport() GitClient {
	return &mockGitClient{
		hasChangesVal:   true,
		reportExistsVal: false,
	}
}

func withNoChanges() GitClient {
	return &mockGitClient{
		hasChangesVal:   false,
		reportExistsVal: false,
	}
}

// ---- AI mock factories ----

func thatAlwaysFails() AgentClient {
	m := newMockAgentClient()
	m.iterateFunc = func() error { return errors.New("iteration failed") }
	m.isFatalFunc = func(err error) bool { return false }
	return m
}

func thatReturnsFatalError() AgentClient {
	m := newMockAgentClient()
	m.iterateFunc = func() error { return errors.New("Insufficient Balance") }
	m.isFatalFunc = func(err error) bool { return true }
	return m
}

func thatReturnsNonFatalError() AgentClient {
	m := newMockAgentClient()
	m.iterateFunc = func() error { return errors.New("non-fatal error") }
	m.isFatalFunc = func(err error) bool { return false }
	return m
}

// ---- Services mock factory ----

func thatFailBeforeCommands() ServicesClient {
	return &mockServicesClient{beforeErr: errors.New("before command failed")}
}

// ---- Tests for mock fixture behaviors ----

func TestMockIterateCallsReturnsProjects(t *testing.T) {
	runner := withMocks()
	proj := withMaxIterations(3)
	_ = runner.RunLocal(proj, anyConfig())
	calls := iterateCalls(runner)
	require.Len(t, calls, 3)
	require.Same(t, proj, calls[0])
	require.Same(t, proj, calls[2])
}

func TestMockChangelogCallsReturnsProjects(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsPassingAfterIterations(1)),
		withGit(withChangesButNoReport()),
	)
	proj := withFailingRequirements()
	_ = runner.RunLocal(proj, anyConfig())
	calls := changelogCalls(runner)
	require.Len(t, calls, 1)
	require.Same(t, proj, calls[0])
}
