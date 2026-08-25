package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/output"
)

func resolveKubeContext(ctx context.Context, client k8s.Client, ralphConfig *config.RalphConfig, out *output.Client, flagContext, flagNamespace string) (k8s.Context, error) {
	if out == nil {
		out = output.NewClient(io.Discard, io.Discard, false)
	}

	var k8sCtx k8s.Context

	if flagContext != "" {
		out.Debugf("Using Kubernetes context: %s", flagContext)
		k8sCtx.Name = flagContext
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		out.Debugf("Using context from .ralph/config.yaml: %s", ralphConfig.Workflow.Context)
		k8sCtx.Name = ralphConfig.Workflow.Context
	} else {
		current, err := client.GetCurrentContext(ctx)
		if err != nil {
			return k8s.Context{}, fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured", err)
		}
		out.Debugf("Using current Kubernetes context: %s", current.Name)
		k8sCtx.Name = current.Name
		k8sCtx.Namespace = current.Namespace
	}

	if flagNamespace != "" {
		out.Debugf("Using namespace: %s", flagNamespace)
		k8sCtx.Namespace = flagNamespace
	} else if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		out.Debugf("Using namespace from .ralph/config.yaml: %s", ralphConfig.Workflow.Namespace)
		k8sCtx.Namespace = ralphConfig.Workflow.Namespace
	} else if ralphConfig != nil && ralphConfig.ConfigPath != "" {
		out.Debugf("Using default namespace: %s (config found)", "config")
		k8sCtx.Namespace = "config"
	}

	if k8sCtx.Namespace == "" {
		out.Debugf("Using namespace: %s (default)", "default")
		k8sCtx.Namespace = "default"
	}

	return k8sCtx, nil
}
