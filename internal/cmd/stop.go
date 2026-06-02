package cmd

import (
	"context"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/stop"
)

type StopCmd struct {
	WorkflowName string `arg:"" help:"Name of the workflow to stop"`
	Context      string `help:"Kubernetes context to use" name:"context" optional:""`
}

func (s *StopCmd) Run() error {
	orchestrator := newStopOrchestrator()
	return orchestrator.Run(context.Background(), s.Context, s.WorkflowName)
}

type stopConfigLoaderAdapter struct{}

func (a *stopConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type stopK8sClientAdapter struct{}

func (a *stopK8sClientAdapter) GetCurrentContext(ctx context.Context) (k8s.Context, error) {
	return k8s.NewClient().GetCurrentContext(ctx)
}

type stopArgoClientAdapter struct{}

func (a *stopArgoClientAdapter) StopWorkflow(ctx stop.KubeContext, workflowName string) error {
	return argo.NewClient().StopWorkflow(argo.K8sContext{
		Name:      ctx.Name,
		Namespace: ctx.Namespace,
	}, workflowName)
}

func newStopOrchestrator() *stop.Stop {
	return stop.New(
		&stopConfigLoaderAdapter{},
		&stopK8sClientAdapter{},
		&stopArgoClientAdapter{},
	)
}
