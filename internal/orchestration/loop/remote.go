package loop

// WorkflowSubmitter submits a loop workflow and returns the workflow name.
type WorkflowSubmitter interface {
	Submit(slug string, steps []string, max int) (string, error)
	// PrintLogHint prints the argo logs command the user can run to follow the
	// submitted workflow.
	PrintLogHint(workflowName string)
}

// RemoteRunnerClient submits a loop workflow for remote execution.
type RemoteRunnerClient interface {
	Run(slug string, steps []string, max int) error
}

// BranchSyncClient resolves the current branch and verifies it is in sync with
// the remote before the loop workflow is submitted.
type BranchSyncClient interface {
	CurrentBranch() (string, error)
	IsBranchSyncedWithRemote(branch string) error
}

// RemoteRunner orchestrates the remote loop execution path: it verifies the
// current branch is in sync with the remote, submits a loop workflow to Argo,
// and prints the argo logs command the user can run to follow it.
type RemoteRunner struct {
	git      BranchSyncClient
	workflow WorkflowSubmitter
}

func NewRemoteRunner(git BranchSyncClient, workflow WorkflowSubmitter) *RemoteRunner {
	return &RemoteRunner{git: git, workflow: workflow}
}

// Run verifies the current branch is in sync with the remote before submitting
// the loop workflow carrying the slug, steps, and maximum iterations, then
// prints the workflow name and the argo logs command.
func (r *RemoteRunner) Run(slug string, steps []string, max int) error {
	branch, err := r.git.CurrentBranch()
	if err != nil {
		return err
	}
	if err := r.git.IsBranchSyncedWithRemote(branch); err != nil {
		return err
	}
	workflowName, err := r.workflow.Submit(slug, steps, max)
	if err != nil {
		return err
	}
	r.workflow.PrintLogHint(workflowName)
	return nil
}
