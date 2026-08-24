package loop

// WorkflowSubmitter submits a loop workflow, streams its logs when following,
// and prints the argo logs command when not following.
type WorkflowSubmitter interface {
	Submit(slug string, steps []string, max int) (string, error)
	// FollowLogs streams the submitted workflow logs and waits for the workflow
	// to finish.
	FollowLogs(workflowName string) error
	// PrintLogHint prints the argo logs command the user can run to follow the
	// submitted workflow.
	PrintLogHint(workflowName string)
}

// RemoteRunnerClient submits a loop workflow for remote execution.
type RemoteRunnerClient interface {
	Run(slug string, steps []string, max int, follow bool) error
}

// BranchSyncClient resolves the current branch and verifies it is in sync with
// the remote before the loop workflow is submitted.
type BranchSyncClient interface {
	CurrentBranch() (string, error)
	IsBranchSyncedWithRemote(branch string) error
}

// NotifyClient sends desktop notifications for a slug.
type NotifyClient interface {
	Error(slug string)
	Success(slug string)
}

// RemoteRunner orchestrates the remote loop execution path: it verifies the
// current branch is in sync with the remote, submits a loop workflow to Argo,
// and either prints the argo logs command the user can run to follow it or
// streams the workflow logs and waits for the workflow to finish, sending a
// success or error notification for the slug on completion.
type RemoteRunner struct {
	git      BranchSyncClient
	workflow WorkflowSubmitter
	notify   NotifyClient
}

func NewRemoteRunner(git BranchSyncClient, workflow WorkflowSubmitter, notify NotifyClient) *RemoteRunner {
	return &RemoteRunner{git: git, workflow: workflow, notify: notify}
}

// Run verifies the current branch is in sync with the remote before submitting
// the loop workflow carrying the slug, steps, and maximum iterations. Without
// --follow it prints the workflow name and the argo logs command. With --follow
// it streams the workflow logs and waits for the workflow to finish, sending a
// success notification for the slug on completion or an error notification when
// streaming fails.
func (r *RemoteRunner) Run(slug string, steps []string, max int, follow bool) error {
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
	if !follow {
		r.workflow.PrintLogHint(workflowName)
		return nil
	}
	if err := r.workflow.FollowLogs(workflowName); err != nil {
		r.notify.Error(slug)
		return err
	}
	r.notify.Success(slug)
	return nil
}
