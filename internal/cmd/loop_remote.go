package cmd

import (
	"github.com/zon/ralph/internal/argo"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/orchestration/loop"
	internalwf "github.com/zon/ralph/internal/workflow"
)

// NewLoopRemoteRunner wires the remote loop execution path with the real git,
// argo, and notify clients.
func NewLoopRemoteRunner(ctx *execcontext.Context) *loop.RemoteRunner {
	return loop.NewRemoteRunner(git.NewClient(ctx), &loopWorkflowClientAdapter{ctx: ctx, argoClient: argo.NewClient()}, notify.NewClient(ctx))
}

// loopWorkflowClientAdapter implements loop.WorkflowSubmitter and submits a
// loop workflow that runs the loop inside the container.
type loopWorkflowClientAdapter struct {
	ctx           *execcontext.Context
	argoClient    argo.Client
	namespace     string
	kubeContext   string
	currentBranch func() (string, error)
}

func (a *loopWorkflowClientAdapter) Submit(slug string, steps []string, max int) (string, error) {
	currentBranch := a.currentBranch
	if currentBranch == nil {
		currentBranch = git.GetCurrentBranch
	}
	cloneBranch, err := currentBranch()
	if err != nil {
		return "", err
	}

	var remoteURL string
	owner, name := a.ctx.RepoOwnerAndName()
	if owner != "" {
		remoteURL = githubpkg.CloneURL(owner, name)
	} else {
		repo, err := githubpkg.GetRepo(a.ctx.GoContext())
		if err != nil {
			return "", err
		}
		remoteURL = repo.CloneURL()
	}

	wf, err := internalwf.GenerateLoopWorkflow(a.ctx, slug, steps, max, cloneBranch, remoteURL)
	if err != nil {
		return "", err
	}
	a.namespace = wf.Namespace
	a.kubeContext = wf.KubeContext

	workflowName, err := wf.Submit(a.ctx.GoContext(), a.argoClient)
	if err != nil {
		return "", err
	}
	a.ctx.Output().Successf("Workflow submitted: %s", workflowName)
	return workflowName, nil
}

// PrintLogHint prints the argo logs command the user can run to follow the
// submitted loop workflow.
func (a *loopWorkflowClientAdapter) PrintLogHint(workflowName string) {
	a.ctx.Output().Infof("To follow logs, run: argo logs -n %s %s -f", a.namespace, workflowName)
}

// FollowLogs streams the submitted loop workflow logs and waits for the
// workflow to finish.
func (a *loopWorkflowClientAdapter) FollowLogs(workflowName string) error {
	return a.argoClient.Logs(argo.K8sContext{Name: a.kubeContext, Namespace: a.namespace}, workflowName, true)
}
