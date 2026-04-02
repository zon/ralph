package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
)

type StopCmd struct {
	WorkflowName string `arg:"" help:"Name of the workflow to stop"`
	Context      string `help:"Kubernetes context to use" name:"context" optional:""`
}

func (s *StopCmd) Run() error {
	ctx := context.Background()

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := resolveKubeContext(ctx, ralphConfig, s.Context, "")
	if err != nil {
		return err
	}

	return argo.Stop(s.WorkflowName, k8sCtx.Name, k8sCtx.Namespace)
}
