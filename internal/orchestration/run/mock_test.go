package run

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
)

// mockAI implements AIClient with configurable behaviors and recorded call
// history for the item-based run flow.
type mockAI struct {
	runPickerFunc          func(proj *project.Project, incomplete []project.Item) (project.Item, error)
	runDeveloperFunc       func(proj *project.Project, item project.Item) error
	isFatalFunc            func(err error) bool
	changelogFunc          func() error
	fixServiceFunc         func(*config.RalphConfig, error) error
	writeOrchestrationFunc func(input *project.InputFile) error
	writeProjectFunc       func(input *project.InputFile) (string, error)

	statsPrinted             bool
	pickCalls                int
	developCalls             int
	changelogCalls           int
	fixServiceCalled         bool
	writeOrchestrationCalled bool
	writeProjectCalled       bool
	lastPickerIndices        []int
	lastPickerItems          []project.Item
	lastDevelopedIndex       int
	lastDevelopedValue       any
}

func (m *mockAI) RunPicker(proj *project.Project, incomplete []project.Item) (project.Item, error) {
	m.pickCalls++
	m.lastPickerIndices = itemIndices(incomplete)
	m.lastPickerItems = cloneProjectItems(incomplete)
	if m.runPickerFunc != nil {
		return m.runPickerFunc(proj, incomplete)
	}
	if len(incomplete) > 0 {
		return incomplete[0], nil
	}
	return project.Item{}, nil
}

func (m *mockAI) RunDeveloper(proj *project.Project, item project.Item) error {
	m.developCalls++
	m.lastDevelopedIndex = item.Index
	m.lastDevelopedValue = item.Value
	if m.runDeveloperFunc != nil {
		return m.runDeveloperFunc(proj, item)
	}
	return nil
}

func (m *mockAI) IsFatal(err error) bool {
	if m.isFatalFunc != nil {
		return m.isFatalFunc(err)
	}
	return false
}

func (m *mockAI) GenerateChangelog(proj *project.Project) error {
	m.changelogCalls++
	if m.changelogFunc != nil {
		return m.changelogFunc()
	}
	return nil
}

func (m *mockAI) PrintStats() {
	m.statsPrinted = true
}

func (m *mockAI) FixServiceStartup(cfg *config.RalphConfig, err error) error {
	m.fixServiceCalled = true
	if m.fixServiceFunc != nil {
		return m.fixServiceFunc(cfg, err)
	}
	return nil
}

func (m *mockAI) WriteOrchestration(input *project.InputFile) error {
	m.writeOrchestrationCalled = true
	if m.writeOrchestrationFunc != nil {
		return m.writeOrchestrationFunc(input)
	}
	return nil
}

func (m *mockAI) WriteProject(input *project.InputFile) (string, error) {
	m.writeProjectCalled = true
	if m.writeProjectFunc != nil {
		return m.writeProjectFunc(input)
	}
	return "projects/generated.yaml", nil
}

func itemIndices(items []project.Item) []int {
	indices := make([]int, len(items))
	for i, it := range items {
		indices[i] = it.Index
	}
	return indices
}

func cloneProjectItems(items []project.Item) []project.Item {
	cloned := make([]project.Item, len(items))
	copy(cloned, items)
	return cloned
}

// mockGit implements GitClient with configurable behaviors, a call-order log,
// and recorded commit history.
type mockGit struct {
	commitsAhead  bool
	blockedFile   bool
	hasChanges    bool
	reportExists  bool
	reportMessage string
	order         []string

	switchToBranchCalled             bool
	writeBlockedFileCalled           bool
	commitFromReportCalled           bool
	commitOrchestrationRemovalCalled bool
	commitGeneratedArtifactsCalled   bool
	commitProjectRemovalCalled       bool
	lastCommitMessage                string
}

func gitNewMock() *mockGit {
	return &mockGit{}
}

func (m *mockGit) SwitchToBranch(slug string) error {
	m.switchToBranchCalled = true
	m.order = append(m.order, "switch")
	return nil
}

func (m *mockGit) BlockedFileExists() bool {
	return m.blockedFile
}

func (m *mockGit) WriteBlockedFile(err error) {
	m.writeBlockedFileCalled = true
}

func (m *mockGit) HasChanges() bool {
	return m.hasChanges
}

func (m *mockGit) ReportExists() bool {
	return m.reportExists
}

func (m *mockGit) CommitFromReport(slug string) error {
	m.commitFromReportCalled = true
	m.lastCommitMessage = m.reportMessage
	return nil
}

func (m *mockGit) CurrentBranch() (string, error) {
	return "main", nil
}

func (m *mockGit) IsBranchSyncedWithRemote(branch string) error {
	return nil
}

func (m *mockGit) CommitOrchestrationRemoval(slug string) error {
	m.commitOrchestrationRemovalCalled = true
	m.order = append(m.order, "commit-orchestration-removal")
	return nil
}

func (m *mockGit) CommitGeneratedArtifacts(slug string) error {
	m.commitGeneratedArtifactsCalled = true
	m.order = append(m.order, "commit-artifacts")
	return nil
}

func (m *mockGit) CommitProjectRemoval(path string) error {
	m.commitProjectRemovalCalled = true
	m.order = append(m.order, "commit-project-removal")
	return nil
}

// mockGitHub implements GitHubClient and records whether CreatePR was called.
type mockGitHub struct {
	createPRCalled bool
}

func (m *mockGitHub) CreatePR(proj *project.Project) error {
	m.createPRCalled = true
	return nil
}

// mockServices implements ServicesClient with recorded start/stop/removeLogs
// counts and configurable failures.
type mockServices struct {
	runBeforeErr    error
	startErr        error
	startCount      int
	stopCount       int
	removeLogsCount int
}

func (m *mockServices) RunBeforeCommands(cfg *config.RalphConfig) error {
	return m.runBeforeErr
}

func (m *mockServices) Start(cfg *config.RalphConfig) (*services.Manager, error) {
	m.startCount++
	if m.startErr != nil {
		return nil, m.startErr
	}
	return &services.Manager{}, nil
}

func (m *mockServices) Stop(svc *services.Manager) {
	m.stopCount++
}

func (m *mockServices) RemoveLogs(cfg *config.RalphConfig) {
	m.removeLogsCount++
}

// mockEnv implements EnvClient.
type mockEnv struct {
	inWorkflow bool
}

func (m *mockEnv) InWorkflow() bool {
	return m.inWorkflow
}

func envInWorkflow() *mockEnv {
	return &mockEnv{inWorkflow: true}
}

func envNotInWorkflow() *mockEnv {
	return &mockEnv{inWorkflow: false}
}

var errFatal = &mockError{"billing limit exceeded"}
var errNonFatal = &mockError{"non-fatal error"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// ---------------------------------------------------------------------------
// AI client builders
// ---------------------------------------------------------------------------

func aiThatAlwaysFails() *mockAI {
	return &mockAI{
		runPickerFunc: func(_ *project.Project, _ []project.Item) (project.Item, error) {
			return project.Item{}, errNonFatal
		},
		isFatalFunc: func(err error) bool { return false },
	}
}

func aiThatFailsServiceFix() *mockAI {
	return &mockAI{
		fixServiceFunc: func(_ *config.RalphConfig, _ error) error { return errNonFatal },
	}
}

func aiThatFailsWriteOrchestration() *mockAI {
	return &mockAI{
		writeOrchestrationFunc: func(*project.InputFile) error { return errNonFatal },
	}
}

func aiThatFailsWriteProject() *mockAI {
	return &mockAI{
		writeProjectFunc: func(*project.InputFile) (string, error) { return "", errNonFatal },
	}
}

func aiThatPicksIndex(i int) *mockAI {
	return &mockAI{
		runPickerFunc: func(proj *project.Project, incomplete []project.Item) (project.Item, error) {
			for _, it := range incomplete {
				if it.Index == i {
					return it, nil
				}
			}
			if i >= 0 && i < len(proj.Items) {
				return proj.Items[i], nil
			}
			return project.Item{}, errNonFatal
		},
	}
}

func aiThatReturnsFatalPickError() *mockAI {
	return &mockAI{
		runPickerFunc: func(_ *project.Project, _ []project.Item) (project.Item, error) {
			return project.Item{}, errFatal
		},
		isFatalFunc: func(err error) bool { return err == errFatal },
	}
}

func aiThatReturnsNonFatalPickError() *mockAI {
	return &mockAI{
		runPickerFunc: func(_ *project.Project, _ []project.Item) (project.Item, error) {
			return project.Item{}, errNonFatal
		},
		isFatalFunc: func(err error) bool { return false },
	}
}

func aiThatReturnsFatalDevelopError() *mockAI {
	return &mockAI{
		runDeveloperFunc: func(_ *project.Project, _ project.Item) error { return errFatal },
		isFatalFunc:      func(err error) bool { return err == errFatal },
	}
}

func aiThatReturnsNonFatalDevelopError() *mockAI {
	return &mockAI{
		runDeveloperFunc: func(_ *project.Project, _ project.Item) error { return errNonFatal },
		isFatalFunc:      func(err error) bool { return false },
	}
}

// ---------------------------------------------------------------------------
// Git client builders
// ---------------------------------------------------------------------------

func gitThatCommitsAhead() *mockGit {
	return &mockGit{commitsAhead: true}
}

func gitWithBlockedFile() *mockGit {
	return &mockGit{blockedFile: true}
}

func gitWithChangesAndReport() *mockGit {
	return &mockGit{hasChanges: true, reportExists: true}
}

func gitWithReport(message string) *mockGit {
	return &mockGit{hasChanges: true, reportExists: true, reportMessage: message}
}

func gitWithChangesButNoReport() *mockGit {
	return &mockGit{hasChanges: true, reportExists: false}
}

func gitWithNoChanges() *mockGit {
	return &mockGit{hasChanges: false, reportExists: false}
}

// ---------------------------------------------------------------------------
// Services client builders
// ---------------------------------------------------------------------------

func servicesThatFailBeforeCommands() *mockServices {
	return &mockServices{runBeforeErr: errNonFatal}
}

func servicesThatFailToStart() *mockServices {
	return &mockServices{startErr: errNonFatal}
}

// ---------------------------------------------------------------------------
// Runner construction
// ---------------------------------------------------------------------------

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
		project:  project.ThatAlwaysReportsIncomplete(),
		ai:       &mockAI{},
		git:      gitNewMock(),
		github:   &mockGitHub{},
		services: &mockServices{},
		notify:   &notify.MockClient{},
		env:      envNotInWorkflow(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func aiStatsPrinted(r *Runner) bool {
	if m, ok := r.ai.(*mockAI); ok {
		return m.statsPrinted
	}
	return false
}

func aiPickCalls(r *Runner) int {
	if m, ok := r.ai.(*mockAI); ok {
		return m.pickCalls
	}
	return 0
}

func aiDevelopCalls(r *Runner) int {
	if m, ok := r.ai.(*mockAI); ok {
		return m.developCalls
	}
	return 0
}

func aiChangelogCalls(r *Runner) int {
	if m, ok := r.ai.(*mockAI); ok {
		return m.changelogCalls
	}
	return 0
}

func aiServiceFixCalled(r *Runner) bool {
	if m, ok := r.ai.(*mockAI); ok {
		return m.fixServiceCalled
	}
	return false
}

func aiWriteOrchestrationCalled(r *Runner) bool {
	if m, ok := r.ai.(*mockAI); ok {
		return m.writeOrchestrationCalled
	}
	return false
}

func aiWriteProjectCalled(r *Runner) bool {
	if m, ok := r.ai.(*mockAI); ok {
		return m.writeProjectCalled
	}
	return false
}

func aiLastPickerIndices(r *Runner) []int {
	if m, ok := r.ai.(*mockAI); ok {
		return m.lastPickerIndices
	}
	return nil
}

func aiLastPickerItems(r *Runner) []project.Item {
	if m, ok := r.ai.(*mockAI); ok {
		return m.lastPickerItems
	}
	return nil
}

func aiLastDevelopedIndex(r *Runner) int {
	if m, ok := r.ai.(*mockAI); ok {
		return m.lastDevelopedIndex
	}
	return -1
}

func aiLastDevelopedValue(r *Runner) any {
	if m, ok := r.ai.(*mockAI); ok {
		return m.lastDevelopedValue
	}
	return nil
}

func gitBranchSwitched(r *Runner) bool {
	if m, ok := r.git.(*mockGit); ok {
		return m.switchToBranchCalled
	}
	return false
}

func gitArtifactsCommitted(r *Runner) bool {
	if m, ok := r.git.(*mockGit); ok {
		return m.commitGeneratedArtifactsCalled
	}
	return false
}

func gitCommittedFromReport(r *Runner) bool {
	if m, ok := r.git.(*mockGit); ok {
		return m.commitFromReportCalled
	}
	return false
}

func gitOrchestrationRemovalCommitted(r *Runner) bool {
	if m, ok := r.git.(*mockGit); ok {
		return m.commitOrchestrationRemovalCalled
	}
	return false
}

func gitProjectRemovalCommitted(r *Runner) bool {
	if m, ok := r.git.(*mockGit); ok {
		return m.commitProjectRemovalCalled
	}
	return false
}

func gitBlockedFileWritten(r *Runner) bool {
	if m, ok := r.git.(*mockGit); ok {
		return m.writeBlockedFileCalled
	}
	return false
}

func gitLastCommitMessage(r *Runner) string {
	if m, ok := r.git.(*mockGit); ok {
		return m.lastCommitMessage
	}
	return ""
}

func gitSwitchedBeforeArtifactsCommitted(r *Runner) bool {
	m, ok := r.git.(*mockGit)
	if !ok {
		return false
	}
	switchIdx, artifactsIdx := -1, -1
	for i, event := range m.order {
		switch event {
		case "switch":
			switchIdx = i
		case "commit-artifacts":
			artifactsIdx = i
		}
	}
	return switchIdx >= 0 && artifactsIdx >= 0 && switchIdx < artifactsIdx
}

func githubPRCreated(r *Runner) bool {
	g, ok := r.git.(*mockGit)
	if !ok || !g.commitsAhead {
		return false
	}
	h, ok := r.github.(*mockGitHub)
	return ok && h.createPRCalled
}

func githubCreatePRCalled(r *Runner) bool {
	if m, ok := r.github.(*mockGitHub); ok {
		return m.createPRCalled
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

// ---------------------------------------------------------------------------
// Remote runner helpers
// ---------------------------------------------------------------------------

type remoteRunnerOption func(*RemoteRunner)

func withRemoteGit(gc GitClient) remoteRunnerOption {
	return func(r *RemoteRunner) { r.git = gc }
}

func withRemoteWorkflow(wc WorkflowClient) remoteRunnerOption {
	return func(r *RemoteRunner) { r.workflow = wc }
}

func withRemoteMocks(opts ...remoteRunnerOption) *RemoteRunner {
	r := &RemoteRunner{
		git:      gitNewMock(),
		workflow: &workflow.MockClient{},
		notify:   &notify.MockClient{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func runRemoteFlagsAny() RunRemoteFlags {
	return RunRemoteFlags{}
}

func runRemoteFlagsWithFollow() RunRemoteFlags {
	return RunRemoteFlags{Follow: true}
}

func runRemoteFlagsWithoutFollow() RunRemoteFlags {
	return RunRemoteFlags{}
}

func runRemoteFlagsWithDebug(branch string) RunRemoteFlags {
	return RunRemoteFlags{Debug: branch}
}

func runRemoteFlagsWithItems(query string) RunRemoteFlags {
	return RunRemoteFlags{Items: query}
}

func runRemoteFlagsWithCleanup() RunRemoteFlags {
	return RunRemoteFlags{Cleanup: true}
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

func remoteWorkflowFollowLogsCalled(runner *RemoteRunner) bool {
	if m, ok := runner.workflow.(*workflow.MockClient); ok {
		return m.FollowLogsCalled
	}
	return false
}

func remoteWorkflowLastDebugBranch(runner *RemoteRunner) string {
	if m, ok := runner.workflow.(*workflow.MockClient); ok {
		return m.LastDebugBranch
	}
	return ""
}

func remoteWorkflowLastItems(runner *RemoteRunner) string {
	if m, ok := runner.workflow.(*workflow.MockClient); ok {
		return m.LastItems
	}
	return ""
}

func remoteWorkflowLastCleanup(runner *RemoteRunner) bool {
	if m, ok := runner.workflow.(*workflow.MockClient); ok {
		return m.LastCleanup
	}
	return false
}

func remoteNotifySuccessSent(runner *RemoteRunner) bool {
	return len(remoteNotifySuccesses(runner)) > 0
}

func remoteNotifyErrorSent(runner *RemoteRunner) bool {
	return len(remoteNotifyErrors(runner)) > 0
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
