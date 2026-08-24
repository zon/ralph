package loop

// WorkflowSubmitter submits a loop workflow and returns the workflow name.
type WorkflowSubmitter interface {
	Submit(slug string, steps []string, max int) (string, error)
}

// RemoteRunnerClient submits a loop workflow for remote execution.
type RemoteRunnerClient interface {
	Run(slug string, steps []string, max int) error
}

// RemoteRunner orchestrates the remote loop execution path: it submits a loop
// workflow to Argo and returns after submission, leaving the loop to run inside
// the workflow container.
type RemoteRunner struct {
	workflow WorkflowSubmitter
}

func NewRemoteRunner(workflow WorkflowSubmitter) *RemoteRunner {
	return &RemoteRunner{workflow: workflow}
}

// Run submits the loop workflow carrying the slug, steps, and maximum
// iterations.
func (r *RemoteRunner) Run(slug string, steps []string, max int) error {
	if _, err := r.workflow.Submit(slug, steps, max); err != nil {
		return err
	}
	return nil
}
