package cmd

import (
	"context"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	orchestrationArgo "github.com/zon/ralph/internal/orchestration/argo"
)

type StopCmd struct {
	WorkflowName string `arg:"" help:"Name of the workflow to stop"`
	Context      string `help:"Kubernetes context to use" name:"context" optional:""`
	Namespace    string `help:"Kubernetes namespace to use" short:"n" optional:""`
}

func (s *StopCmd) Run() error {
	ctx := context.Background()

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	k8sClient := k8s.NewClient()
	cmd := newOrchestrationArgoCmd(ctx, k8sClient, ralphConfig)
	return cmd.Stop(orchestrationArgo.StopFlags{
		Context:      s.Context,
		Namespace:    s.Namespace,
		WorkflowName: s.WorkflowName,
	})
}
