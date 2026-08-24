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
	slug     string
	steps    []string
	max      int
	workflow string
	err      error
	calls    int
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

// TestRemoteRunnerSubmitsLoopWorkflow asserts the remote runner submits the
// loop workflow carrying the slug, steps, and max iterations.
func TestRemoteRunnerSubmitsLoopWorkflow(t *testing.T) {
	submitter := &mockWorkflowSubmitter{workflow: "loop-workflow"}
	runner := NewRemoteRunner(submitter)

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
	runner := NewRemoteRunner(submitter)

	err := runner.Run("fmt", nil, 10)

	require.Error(t, err)
	assert.Equal(t, submitErr, err, "the workflow submission error is returned unchanged")
	assert.Equal(t, 1, submitter.calls, "the workflow submission is attempted once")
}
