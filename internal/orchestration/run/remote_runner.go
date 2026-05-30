package run

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
