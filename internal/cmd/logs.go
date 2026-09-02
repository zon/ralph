package cmd

import (
	orchestrationArgo "github.com/zon/ralph/internal/orchestration/argo"
)

type LogsCmd struct {
	WorkflowName string `arg:"" help:"Name of the workflow to get logs for (default: most recently created Ralph workflow)" optional:""`
	Follow       bool   `help:"Stream logs as they are produced" short:"f"`
	Context      string `help:"The name of the Kubernetes context to use"`
	Namespace    string `help:"The name of the Kubernetes namespace to use" short:"n"`
}

func (l *LogsCmd) Run() error {
	return runArgoCmd(func(cmd *orchestrationArgo.ArgoCmd) error {
		return cmd.Logs(orchestrationArgo.LogsFlags{
			Context:      l.Context,
			Namespace:    l.Namespace,
			WorkflowName: l.WorkflowName,
			Follow:       l.Follow,
		})
	})
}
