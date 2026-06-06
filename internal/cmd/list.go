package cmd

import (
	"context"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	orchestrationArgo "github.com/zon/ralph/internal/orchestration/argo"
)

type ListCmd struct {
	Context   string `help:"Kubernetes context to use" name:"context" optional:""`
	Namespace string `help:"Kubernetes namespace to use" short:"n" optional:""`
}

func (l *ListCmd) Run() error {
	ctx := context.Background()

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	k8sClient := k8s.NewClient()
	cmd := newOrchestrationArgoCmd(ctx, k8sClient, ralphConfig)
	return cmd.List(orchestrationArgo.ListFlags{
		Context:   l.Context,
		Namespace: l.Namespace,
	})
}

type argoContextClient struct {
	ctx         context.Context
	k8sClient   k8s.Client
	ralphConfig *config.RalphConfig
}

func (a *argoContextClient) Resolve(flagContext, flagNamespace string) (orchestrationArgo.K8sContext, error) {
	k8sCtx, err := resolveKubeContext(a.ctx, a.k8sClient, a.ralphConfig, nil, flagContext, flagNamespace)
	if err != nil {
		return orchestrationArgo.K8sContext{}, err
	}
	return orchestrationArgo.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

type argoClientAdapter struct {
	client argo.Client
}

func (a *argoClientAdapter) List(ctx orchestrationArgo.K8sContext) error {
	return a.client.ListWorkflows(argo.K8sContext{Name: ctx.Name, Namespace: ctx.Namespace})
}

func (a *argoClientAdapter) Stop(ctx orchestrationArgo.K8sContext, workflowName string) error {
	return a.client.StopWorkflow(argo.K8sContext{Name: ctx.Name, Namespace: ctx.Namespace}, workflowName)
}

func newOrchestrationArgoCmd(ctx context.Context, k8sClient k8s.Client, ralphConfig *config.RalphConfig) *orchestrationArgo.ArgoCmd {
	return &orchestrationArgo.ArgoCmd{
		Argo: &argoClientAdapter{client: argo.NewClient()},
		Ctx:  &argoContextClient{ctx: ctx, k8sClient: k8sClient, ralphConfig: ralphConfig},
	}
}
