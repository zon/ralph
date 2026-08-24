package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWorkflowSubmitter records the submission arguments and returns an
// injected workflow name or error.
type mockWorkflowSubmitter struct {
	slug           string
	steps          []string
	max            int
	workflow       string
	err            error
	calls          int
	hintWorkflow   string
	hintCalls      int
	followWorkflow string
	followCalls    int
	followErr      error
}

func (m *mockWorkflowSubmitter) Submit(slug string, steps []string, max int) (string, error) {
	m.calls++
	m.slug = slug
	m.steps = steps
	m.max = max
	if m.err != nil {
		return "", m.err
	}
	return m.workflow, nil
}

func (m *mockWorkflowSubmitter) PrintLogHint(workflowName string) {
	m.hintCalls++
	m.hintWorkflow = workflowName
}

func (m *mockWorkflowSubmitter) FollowLogs(workflowName string) error {
	m.followCalls++
	m.followWorkflow = workflowName
	return m.followErr
}

// mockNotifyClient records the success and error notifications sent for slugs,
// so tests never touch the real desktop notifier.
type mockNotifyClient struct {
	successes []string
	errors    []string
}

func (m *mockNotifyClient) Error(slug string) {
	m.errors = append(m.errors, slug)
}

func (m *mockNotifyClient) Success(slug string) {
	m.successes = append(m.successes, slug)
}

var _ NotifyClient = (*mockNotifyClient)(nil)

// mockBranchSyncClient resolves the current branch and verifies it is in sync
// with the remote, recording the branch it was asked to check and returning an
// injected error when set.
type mockBranchSyncClient struct {
	branch string
	err    error
	checks int
	last   string
}

func (m *mockBranchSyncClient) CurrentBranch() (string, error) {
	return m.branch, nil
}

func (m *mockBranchSyncClient) IsBranchSyncedWithRemote(branch string) error {
	m.checks++
	m.last = branch
	return m.err
}

// TestRemoteRunnerVerifiesBranchSyncedBeforeSubmit asserts the remote runner
// checks the current branch is in sync with the remote before submitting the
// loop workflow, then prints the argo logs hint for the submitted workflow
// when not following.
func TestRemoteRunnerVerifiesBranchSyncedBeforeSubmit(t *testing.T) {
	gitClient := &mockBranchSyncClient{branch: "feature"}
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow"}
	runner := NewRemoteRunner(gitClient, submitter, &mockNotifyClient{})

	err := runner.Run("fmt", []string{"run gofmt"}, 3, false)

	require.NoError(t, err)
	assert.Equal(t, 1, gitClient.checks, "the branch sync is verified before submission")
	assert.Equal(t, "feature", gitClient.last, "the current branch is the branch checked against the remote")
	assert.Equal(t, 1, submitter.calls, "the loop workflow is submitted exactly once after the branch is in sync")
	assert.Equal(t, "fmt", submitter.slug, "the workflow is submitted with the slug")
	assert.Equal(t, []string{"run gofmt"}, submitter.steps, "the workflow is submitted with the steps")
	assert.Equal(t, 3, submitter.max, "the workflow is submitted with the max iterations")
	assert.Equal(t, 1, submitter.hintCalls, "the argo logs hint is printed once after submission")
	assert.Equal(t, "loop-workflow", submitter.hintWorkflow, "the hint names the submitted workflow")
	assert.Equal(t, 0, submitter.followCalls, "the workflow logs are not followed without --follow")
}

// TestRemoteRunnerAbortsWhenBranchNotPushed asserts the remote runner returns
// the not-pushed error and never submits the loop workflow when the branch has
// no remote counterpart.
func TestRemoteRunnerAbortsWhenBranchNotPushed(t *testing.T) {
	syncErr := errors.New("branch 'feature' has not been pushed to remote - please push before running remotely")
	gitClient := &mockBranchSyncClient{branch: "feature", err: syncErr}
	submitter := &mockWorkflowSubmitter{}
	runner := NewRemoteRunner(gitClient, submitter, &mockNotifyClient{})

	err := runner.Run("fmt", nil, 10, false)

	require.Error(t, err)
	assert.Equal(t, syncErr, err, "the not-pushed error is returned unchanged")
	assert.Equal(t, 0, submitter.calls, "the loop workflow is not submitted when the branch has not been pushed")
	assert.Equal(t, 0, submitter.hintCalls, "no argo logs hint is printed without a submission")
}

// TestRemoteRunnerAbortsWhenBranchNotInSync asserts the remote runner returns
// the out-of-sync error and never submits the loop workflow when local and
// remote differ at the current commit.
func TestRemoteRunnerAbortsWhenBranchNotInSync(t *testing.T) {
	syncErr := errors.New("branch 'feature' is not in sync with remote - please push your changes before running remotely")
	gitClient := &mockBranchSyncClient{branch: "feature", err: syncErr}
	submitter := &mockWorkflowSubmitter{}
	runner := NewRemoteRunner(gitClient, submitter, &mockNotifyClient{})

	err := runner.Run("fmt", nil, 10, false)

	require.Error(t, err)
	assert.Equal(t, syncErr, err, "the out-of-sync error is returned unchanged")
	assert.Equal(t, 0, submitter.calls, "the loop workflow is not submitted when the branch is out of sync")
	assert.Equal(t, 0, submitter.hintCalls, "no argo logs hint is printed without a submission")
}

// TestRemoteRunnerSubmitsLoopWorkflow asserts the remote runner submits the
// loop workflow carrying the slug, steps, and max iterations once the current
// branch is verified in sync.
func TestRemoteRunnerSubmitsLoopWorkflow(t *testing.T) {
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow"}
	runner := NewRemoteRunner(&mockBranchSyncClient{branch: "main"}, submitter, &mockNotifyClient{})

	err := runner.Run("fmt", []string{"run gofmt", "run go vet"}, 3, false)

	require.NoError(t, err)
	assert.Equal(t, 1, submitter.calls, "the loop workflow is submitted exactly once")
	assert.Equal(t, "fmt", submitter.slug, "the workflow is submitted with the slug")
	assert.Equal(t, []string{"run gofmt", "run go vet"}, submitter.steps, "the workflow is submitted with the steps")
	assert.Equal(t, 3, submitter.max, "the workflow is submitted with the max iterations")
}

// TestRemoteRunnerPropagatesSubmitError asserts a workflow submission failure
// aborts the remote runner and is returned unchanged.
func TestRemoteRunnerPropagatesSubmitError(t *testing.T) {
	submitErr := errors.New("submit boom")
	submitter := &mockWorkflowSubmitter{err: submitErr}
	runner := NewRemoteRunner(&mockBranchSyncClient{branch: "main"}, submitter, &mockNotifyClient{})

	err := runner.Run("fmt", nil, 10, false)

	require.Error(t, err)
	assert.Equal(t, submitErr, err, "the workflow submission error is returned unchanged")
	assert.Equal(t, 1, submitter.calls, "the workflow submission is attempted once")
	assert.Equal(t, 0, submitter.hintCalls, "no argo logs hint is printed when submission fails")
}

// TestRemoteRunnerFollowStreamsLogsAndNotifiesSuccess asserts that with --follow
// the remote runner streams the submitted workflow logs, waits for the workflow
// to finish, and sends a success notification for the slug on completion.
func TestRemoteRunnerFollowStreamsLogsAndNotifiesSuccess(t *testing.T) {
	gitClient := &mockBranchSyncClient{branch: "main"}
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow"}
	notifier := &mockNotifyClient{}
	runner := NewRemoteRunner(gitClient, submitter, notifier)

	err := runner.Run("fmt", []string{"run gofmt"}, 3, true)

	require.NoError(t, err)
	assert.Equal(t, 1, submitter.followCalls, "the workflow logs are streamed exactly once with --follow")
	assert.Equal(t, "loop-workflow", submitter.followWorkflow, "the logs are streamed for the submitted workflow")
	assert.Equal(t, 0, submitter.hintCalls, "no argo logs hint is printed when following the logs")
	assert.Equal(t, []string{"fmt"}, notifier.successes, "a success notification is sent for the slug on completion")
	assert.Empty(t, notifier.errors, "no error notification is sent on success")
}

// TestRemoteRunnerFollowFailureNotifiesErrorAndReturns asserts a workflow log
// streaming failure aborts the remote runner, sends an error notification for
// the slug, and returns the error unchanged.
func TestRemoteRunnerFollowFailureNotifiesErrorAndReturns(t *testing.T) {
	followErr := errors.New("follow boom")
	gitClient := &mockBranchSyncClient{branch: "main"}
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow", followErr: followErr}
	notifier := &mockNotifyClient{}
	runner := NewRemoteRunner(gitClient, submitter, notifier)

	err := runner.Run("fmt", []string{"run gofmt"}, 3, true)

	require.Error(t, err)
	assert.Equal(t, followErr, err, "the log streaming error is returned unchanged")
	assert.Equal(t, 1, submitter.followCalls, "the workflow logs are streamed once")
	assert.Equal(t, []string{"fmt"}, notifier.errors, "an error notification is sent for the slug on failure")
	assert.Empty(t, notifier.successes, "no success notification is sent on failure")
}

// TestRemoteRunnerWithoutFollowDoesNotNotify asserts that without --follow the
// remote runner prints the log hint and never sends a notification.
func TestRemoteRunnerWithoutFollowDoesNotNotify(t *testing.T) {
	gitClient := &mockBranchSyncClient{branch: "main"}
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow"}
	notifier := &mockNotifyClient{}
	runner := NewRemoteRunner(gitClient, submitter, notifier)

	err := runner.Run("fmt", []string{"run gofmt"}, 3, false)

	require.NoError(t, err)
	assert.Equal(t, 1, submitter.hintCalls, "the argo logs hint is printed without --follow")
	assert.Equal(t, 0, submitter.followCalls, "the workflow logs are not streamed without --follow")
	assert.Empty(t, notifier.successes, "no success notification is sent without --follow")
	assert.Empty(t, notifier.errors, "no error notification is sent without --follow")
}
