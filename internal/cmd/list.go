package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
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

	k8sClient := k8s.NewClient()
	k8sCtx, err := resolveKubeContext(ctx, k8sClient, ralphConfig, nil, l.Context, "")
	if err != nil {
		return err
	}

	client := argo.NewClient()
	return client.ListWorkflows(argo.K8sContext{
		Name:      k8sCtx.Name,
		Namespace: k8sCtx.Namespace,
	})
}
