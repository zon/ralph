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
	slug         string
	steps        []string
	max          int
	workflow     string
	err          error
	calls        int
	hintWorkflow string
	hintCalls    int
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
// loop workflow, then prints the argo logs hint for the submitted workflow.
func TestRemoteRunnerVerifiesBranchSyncedBeforeSubmit(t *testing.T) {
	gitClient := &mockBranchSyncClient{branch: "feature"}
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow"}
	runner := NewRemoteRunner(gitClient, submitter)

	err := runner.Run("fmt", []string{"run gofmt"}, 3)

	require.NoError(t, err)
	assert.Equal(t, 1, gitClient.checks, "the branch sync is verified before submission")
	assert.Equal(t, "feature", gitClient.last, "the current branch is the branch checked against the remote")
	assert.Equal(t, 1, submitter.calls, "the loop workflow is submitted exactly once after the branch is in sync")
	assert.Equal(t, "fmt", submitter.slug, "the workflow is submitted with the slug")
	assert.Equal(t, []string{"run gofmt"}, submitter.steps, "the workflow is submitted with the steps")
	assert.Equal(t, 3, submitter.max, "the workflow is submitted with the max iterations")
	assert.Equal(t, 1, submitter.hintCalls, "the argo logs hint is printed once after submission")
	assert.Equal(t, "loop-workflow", submitter.hintWorkflow, "the hint names the submitted workflow")
}

// TestRemoteRunnerAbortsWhenBranchNotPushed asserts the remote runner returns
// the not-pushed error and never submits the loop workflow when the branch has
// no remote counterpart.
func TestRemoteRunnerAbortsWhenBranchNotPushed(t *testing.T) {
	syncErr := errors.New("branch 'feature' has not been pushed to remote - please push before running remotely")
	gitClient := &mockBranchSyncClient{branch: "feature", err: syncErr}
	submitter := &mockWorkflowSubmitter{}
	runner := NewRemoteRunner(gitClient, submitter)

	err := runner.Run("fmt", nil, 10)

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
	runner := NewRemoteRunner(gitClient, submitter)

	err := runner.Run("fmt", nil, 10)

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
	runner := NewRemoteRunner(&mockBranchSyncClient{branch: "main"}, submitter)

	err := runner.Run("fmt", []string{"run gofmt", "run go vet"}, 3)

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
	runner := NewRemoteRunner(&mockBranchSyncClient{branch: "main"}, submitter)

	err := runner.Run("fmt", nil, 10)

	require.Error(t, err)
	assert.Equal(t, submitErr, err, "the workflow submission error is returned unchanged")
	assert.Equal(t, 1, submitter.calls, "the workflow submission is attempted once")
	assert.Equal(t, 0, submitter.hintCalls, "no argo logs hint is printed when submission fails")
}
