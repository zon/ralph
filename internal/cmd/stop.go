package cmd

import (
	orchestrationArgo "github.com/zon/ralph/internal/orchestration/argo"
)

type StopCmd struct {
	WorkflowName string `arg:"" help:"Name of the workflow to stop"`
	Context      string `help:"The name of the Kubernetes context to use"`
	Namespace    string `help:"The name of the Kubernetes namespace to use" short:"n"`
}

func (s *StopCmd) Run() error {
	return runArgoCmd(func(cmd *orchestrationArgo.ArgoCmd) error {
		return cmd.Stop(orchestrationArgo.StopFlags{
			Context:      s.Context,
			Namespace:    s.Namespace,
			WorkflowName: s.WorkflowName,
		})
	})
}
