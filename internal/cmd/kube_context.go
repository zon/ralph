package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
)

// resolveKubeContext resolves the Kubernetes context and namespace with the following priority:
// Context priority:
// 1. Command-line flags (if provided)
// 2. .ralph/config.yaml (workflow.context)
// 3. kubectl configuration (current context)
//
// Namespace priority:
// 1. Command-line flags (if provided)
// 2. .ralph/config.yaml (workflow.namespace)
// 3. "config" namespace (if .ralph/config.yaml is found)
// 4. kubectl configuration (context namespace)
// 5. Default namespace ("default")
func resolveKubeContext(ctx context.Context, ralphConfig *config.RalphConfig, flagContext, flagNamespace string) (k8s.Context, error) {
	var k8sCtx k8s.Context

	// 1. Resolve Context Name
	if flagContext != "" {
		logger.Verbosef("Using Kubernetes context: %s", flagContext)
		k8sCtx.Name = flagContext
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		logger.Verbosef("Using context from .ralph/config.yaml: %s", ralphConfig.Workflow.Context)
		k8sCtx.Name = ralphConfig.Workflow.Context
	} else {
		current, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return k8s.Context{}, fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		logger.Verbosef("Using current Kubernetes context: %s", current.Name)
		k8sCtx.Name = current.Name
		// Store the default namespace from kubectl as a fallback
		k8sCtx.Namespace = current.Namespace
	}

	// 2. Resolve Namespace
	if flagNamespace != "" {
		logger.Verbosef("Using namespace: %s", flagNamespace)
		k8sCtx.Namespace = flagNamespace
	} else if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		logger.Verbosef("Using namespace from .ralph/config.yaml: %s", ralphConfig.Workflow.Namespace)
		k8sCtx.Namespace = ralphConfig.Workflow.Namespace
	} else if ralphConfig != nil && ralphConfig.ConfigPath != "" {
		logger.Verbosef("Using default namespace: %s (config found)", "config")
		k8sCtx.Namespace = "config"
	}

	// 3. Final fallback to "default" if still empty
	if k8sCtx.Namespace == "" {
		logger.Verbosef("Using namespace: %s (default)", "default")
		k8sCtx.Namespace = "default"
	}

	return k8sCtx, nil
}
