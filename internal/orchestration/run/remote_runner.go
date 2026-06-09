package run

import "github.com/zon/ralph/internal/project"

type RunRemoteFlags struct {
	Follow bool
	Debug  string
}

type RemoteRunner struct {
	git      GitClient
	workflow WorkflowClient
	notify   NotifyClient
}

func NewRemoteRunner(git GitClient, workflow WorkflowClient, notify NotifyClient) *RemoteRunner {
	return &RemoteRunner{
		git:      git,
		workflow: workflow,
		notify:   notify,
	}
}

func (r *RemoteRunner) Run(input *project.InputFile, flags RunRemoteFlags) error {
	branch, err := r.git.CurrentBranch()
	if err != nil {
		return err
	}
	if err := r.git.IsBranchSyncedWithRemote(branch); err != nil {
		return err
	}
	workflowName, err := r.workflow.Submit(input, branch, flags.Debug)
	if err != nil {
		return err
	}
	if !flags.Follow {
		r.workflow.PrintLogHint(workflowName)
		return nil
	}
	if err := r.workflow.FollowLogs(workflowName); err != nil {
		r.notify.Error(input.Slug())
		return err
	}
	r.notify.Success(input.Slug())
	return nil
}
