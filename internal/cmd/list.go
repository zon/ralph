package cmd

import (
	"context"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/list"
)

type ListCmd struct {
	Context string `help:"Kubernetes context to use" name:"context" optional:""`
}

func (l *ListCmd) Run() error {
	orchestrator := newListOrchestrator()
	return orchestrator.Run(context.Background(), l.Context)
}

type listConfigLoaderAdapter struct{}

func (a *listConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type listK8sClientAdapter struct{}

func (a *listK8sClientAdapter) GetCurrentContext(ctx context.Context) (k8s.Context, error) {
	return k8s.NewClient().GetCurrentContext(ctx)
}

type listArgoClientAdapter struct{}

func (a *listArgoClientAdapter) ListWorkflows(ctx list.KubeContext) error {
	return argo.NewClient().ListWorkflows(argo.K8sContext{
		Name:      ctx.Name,
		Namespace: ctx.Namespace,
	})
}

func newListOrchestrator() *list.List {
	return list.New(
		&listConfigLoaderAdapter{},
		&listK8sClientAdapter{},
		&listArgoClientAdapter{},
	)
}
