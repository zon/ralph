package run

import "github.com/zon/ralph/internal/project"

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

func (r *RemoteRunner) RunRemote(proj *project.Project, follow bool) error {
	branch, err := r.git.CurrentBranch()
	if err != nil {
		return err
	}
	if err := r.git.IsBranchSyncedWithRemote(branch); err != nil {
		return err
	}
	workflowName, err := r.workflow.Submit(proj, branch)
	if err != nil {
		return err
	}
	if !follow {
		r.workflow.PrintLogHint(workflowName)
		return nil
	}
	if err := r.workflow.FollowLogs(workflowName); err != nil {
		r.notify.Error(proj.Slug)
		return err
	}
	r.notify.Success(proj.Slug)
	return nil
}
