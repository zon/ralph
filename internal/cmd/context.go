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
// 3. kubectl configuration (current context and context namespace)
// 4. Default namespace ("default")
// Returns: kubeContext, namespace, error
func loadContextAndNamespace(ctx context.Context, flagContext, flagNamespace string) (string, string, error) {
	// Try to load .ralph/config.yaml for defaults
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		logger.Verbosef("Failed to load .ralph/config.yaml: %v (using kubectl config)", err)
	}

	// Determine the Kubernetes context
	var kubeContext string
	var contextSource string

	if flagContext != "" {
		// Command-line flag takes highest priority
		kubeContext = flagContext
		contextSource = "flag"
		logger.Verbosef("Using Kubernetes context: %s", kubeContext)
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		// .ralph/config.yaml is second priority
		kubeContext = ralphConfig.Workflow.Context
		contextSource = ".ralph/config.yaml"
		logger.Verbosef("Using context from .ralph/config.yaml: %s", kubeContext)
	} else {
		// Fall back to kubectl current context
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		kubeContext = currentCtx
		contextSource = "kubectl"
		logger.Verbosef("Using current Kubernetes context: %s", kubeContext)
	}

	// Determine the namespace
	var namespace string

	if flagNamespace != "" {
		// Command-line flag takes highest priority
		namespace = flagNamespace
		logger.Verbosef("Using namespace: %s", namespace)
	} else if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		// .ralph/config.yaml is second priority
		namespace = ralphConfig.Workflow.Namespace
		if contextSource == ".ralph/config.yaml" {
			logger.Verbosef("Using namespace from .ralph/config.yaml: %s", namespace)
		} else {
			logger.Verbosef("Using namespace from .ralph/config.yaml: %s (context from %s)", namespace, contextSource)
		}
	} else {
		// Fall back to kubectl context namespace
		ns, err := k8s.GetNamespaceForContext(ctx, kubeContext)
		if err != nil {
			logger.Verbosef("Failed to get namespace for context: %v", err)
		}
		if ns == "" {
			namespace = "default"
			logger.Verbosef("Using namespace: %s (default)", namespace)
		} else {
			namespace = ns
			logger.Verbosef("Using namespace: %s (from kubectl context)", namespace)
		}
	}

	return kubeContext, namespace, nil
}
