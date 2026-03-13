package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
)

// loadContextAndNamespace loads the Kubernetes context and namespace with the following priority:
// 1. Command-line flags (if provided)
// 2. .ralph/config.yaml (workflow.context and workflow.namespace)
// 3. "config" namespace (if .ralph/config.yaml is found)
// 4. kubectl configuration (current context and context namespace)
// 5. Default namespace ("default")
// Returns: kubeContext, namespace, error
func loadContextAndNamespace(ctx context.Context, flagContext, flagNamespace string) (string, string, error) {
	ralphConfig, _ := config.LoadConfig()

	kubeContext, contextSource, err := determineKubeContext(ctx, flagContext, ralphConfig)
	if err != nil {
		return "", "", err
	}

	namespace := determineNamespace(ctx, flagNamespace, ralphConfig, kubeContext, contextSource)

	return kubeContext, namespace, nil
}

func determineKubeContext(ctx context.Context, flagContext string, ralphConfig *config.RalphConfig) (string, string, error) {
	if flagContext != "" {
		logger.Verbosef("Using Kubernetes context: %s", flagContext)
		return flagContext, "flag", nil
	}

	if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		logger.Verbosef("Using context from .ralph/config.yaml: %s", ralphConfig.Workflow.Context)
		return ralphConfig.Workflow.Context, ".ralph/config.yaml", nil
	}

	currentCtx, err := k8s.GetCurrentContext(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
	}
	logger.Verbosef("Using current Kubernetes context: %s", currentCtx)
	return currentCtx, "kubectl", nil
}

func determineNamespace(ctx context.Context, flagNamespace string, ralphConfig *config.RalphConfig, kubeContext, contextSource string) string {
	if flagNamespace != "" {
		logger.Verbosef("Using namespace: %s", flagNamespace)
		return flagNamespace
	}

	if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		namespace := ralphConfig.Workflow.Namespace
		if contextSource == ".ralph/config.yaml" {
			logger.Verbosef("Using namespace from .ralph/config.yaml: %s", namespace)
		} else {
			logger.Verbosef("Using namespace from .ralph/config.yaml: %s (context from %s)", namespace, contextSource)
		}
		return namespace
	}

	if ralphConfig != nil && ralphConfig.ConfigPath != "" {
		logger.Verbosef("Using default namespace: %s (config found)", "config")
		return "config"
	}

	ns, err := k8s.GetNamespaceForContext(ctx, kubeContext)
	if err != nil {
		logger.Verbosef("Failed to get namespace for context: %v", err)
	}
	if ns == "" {
		logger.Verbosef("Using namespace: %s (default)", "default")
		return "default"
	}
	logger.Verbosef("Using namespace: %s (from kubectl context)", ns)
	return ns
}
