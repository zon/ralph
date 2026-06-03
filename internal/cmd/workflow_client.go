package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
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

func (a *workflowClientAdapter) Submit(proj *project.Project, cloneBranch string, debug string) (string, error) {
	projectBranch := git.SanitizeBranchName(proj.Slug)

	var repoURL string
	if a.ctx.Repo() != "" {
		owner, name := a.ctx.RepoOwnerAndName()
		repoURL = githubpkg.CloneURL(owner, name)
	} else {
		repo, err := githubpkg.GetRepo(a.ctx.GoContext())
		if err != nil {
			return "", fmt.Errorf("failed to get repository: %w", err)
		}
		repoURL = repo.CloneURL()
	}

	relProjectPath := a.ctx.ProjectFile()
	if filepath.IsAbs(relProjectPath) {
		if a.ctx.Repo() == "" {
			repoRoot, err := git.FindRepoRoot()
			if err != nil {
				return "", fmt.Errorf("failed to get repository root: %w", err)
			}
			relProjectPath, err = filepath.Rel(repoRoot, relProjectPath)
			if err != nil {
				return "", fmt.Errorf("failed to calculate relative project path: %w", err)
			}
		}
	}

	wf, err := workflow.GenerateWorkflow(a.ctx, proj.Slug, cloneBranch, projectBranch, a.ctx.IsVerbose(), repoURL, relProjectPath)
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
	a.ctx.Output().Infof("To follow logs, run: argo logs -n %s %s -f", a.namespace, workflowName)
}

func NewRemoteRunner(ctx *context.Context) *orchestrationRun.RemoteRunner {
	return orchestrationRun.NewRemoteRunner(
		git.NewClient(ctx),
		&workflowClientAdapter{ctx: ctx, argoClient: argo.NewClient()},
		notify.NewClient(ctx),
	)
}
