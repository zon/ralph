package stop

import (
	"context"
	"fmt"
	"io"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/output"
)

type KubeContext struct {
	Name      string
	Namespace string
}

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type K8sClient interface {
	GetCurrentContext(ctx context.Context) (k8s.Context, error)
}

type ArgoClient interface {
	StopWorkflow(ctx KubeContext, workflowName string) error
}

type Stop struct {
	configLoader ConfigLoader
	k8sClient    K8sClient
	argoClient   ArgoClient
}

func New(configLoader ConfigLoader, k8sClient K8sClient, argoClient ArgoClient) *Stop {
	return &Stop{
		configLoader: configLoader,
		k8sClient:    k8sClient,
		argoClient:   argoClient,
	}
}

func (s *Stop) Run(ctx context.Context, flagContext, workflowName string) error {
	ralphConfig, err := s.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := s.resolveKubeContext(ctx, ralphConfig, flagContext)
	if err != nil {
		return err
	}

	return s.argoClient.StopWorkflow(k8sCtx, workflowName)
}

func (s *Stop) resolveKubeContext(ctx context.Context, ralphConfig *config.RalphConfig, flagContext string) (KubeContext, error) {
	out := output.NewClient(io.Discard, io.Discard, false)

	var kc KubeContext

	if flagContext != "" {
		out.Debugf("Using Kubernetes context: %s", flagContext)
		kc.Name = flagContext
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		out.Debugf("Using context from .ralph/config.yaml: %s", ralphConfig.Workflow.Context)
		kc.Name = ralphConfig.Workflow.Context
	} else {
		current, err := s.k8sClient.GetCurrentContext(ctx)
		if err != nil {
			return KubeContext{}, fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		out.Debugf("Using current Kubernetes context: %s", current.Name)
		kc.Name = current.Name
		kc.Namespace = current.Namespace
	}

	if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		out.Debugf("Using namespace from .ralph/config.yaml: %s", ralphConfig.Workflow.Namespace)
		kc.Namespace = ralphConfig.Workflow.Namespace
	} else if ralphConfig != nil && ralphConfig.ConfigPath != "" {
		out.Debugf("Using default namespace: %s (config found)", "config")
		kc.Namespace = "config"
	}

	if kc.Namespace == "" {
		out.Debugf("Using namespace: %s (default)", "default")
		kc.Namespace = "default"
	}

	return kc, nil
}
