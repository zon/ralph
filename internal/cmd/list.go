package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
)

type ListCmd struct {
	Context string `help:"Kubernetes context to use" name:"context" optional:""`
}

func (l *ListCmd) Run() error {
	ctx := context.Background()

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := resolveKubeContext(ctx, ralphConfig, l.Context, "")
	if err != nil {
		return err
	}

	return argo.ListWorkflows(argo.K8sContext{
		Name:      k8sCtx.Name,
		Namespace: k8sCtx.Namespace,
	})
}
