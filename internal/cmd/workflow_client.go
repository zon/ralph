package cmd

import (
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

type workflowClientAdapter struct {
	ctx         *context.Context
	argoClient  argo.Client
	namespace   string
	kubeContext string
}

func (a *workflowClientAdapter) Submit(proj *project.Project, cloneBranch string) (string, error) {
	projectBranch := git.SanitizeBranchName(proj.Slug)
	wf, err := workflow.GenerateWorkflow(a.ctx, proj.Slug, cloneBranch, projectBranch, a.ctx.IsVerbose())
	if err != nil {
		return "", err
	}
	a.namespace = wf.Namespace
	a.kubeContext = wf.KubeContext
	return wf.Submit(a.ctx.GoContext(), a.argoClient)
}

func (a *workflowClientAdapter) FollowLogs(workflowName string) error {
	return a.argoClient.FollowLogs(argo.K8sContext{Name: a.kubeContext, Namespace: a.namespace}, workflowName)
}

func (a *workflowClientAdapter) PrintLogHint(workflowName string) {
	logger.Infof("To follow logs, run: argo logs -n %s %s -f", a.namespace, workflowName)
}

func NewRemoteRunner(ctx *context.Context) *orchestrationRun.RemoteRunner {
	return orchestrationRun.NewRemoteRunner(
		git.NewClient(ctx),
		&workflowClientAdapter{ctx: ctx, argoClient: argo.NewClient()},
		notify.NewClient(ctx),
	)
}
