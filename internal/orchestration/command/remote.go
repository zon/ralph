package command

import (
	"fmt"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	internalwf "github.com/zon/ralph/internal/workflow"
)

func ExecuteRemoteCommand(ctx *context.Context, argoClient argo.Client) error {
	ctx.Output().Debug("Submitting Argo Workflow for command...")

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	ctx.Output().Debugf("Checking branch '%s' is in sync with remote...", currentBranch)
	if err := git.IsBranchSyncedWithRemote(currentBranch); err != nil {
		return err
	}

	ctx.Output().Debug("Generating command workflow...")
	var remoteURL string
	owner, name := ctx.RepoOwnerAndName()
	if owner != "" {
		remoteURL = githubpkg.CloneURL(owner, name)
	} else {
		remoteURL, err = git.RemoteURL()
		if err != nil {
			return fmt.Errorf("failed to get remote URL: %w", err)
		}
	}

	wf, err := internalwf.GenerateCommandWorkflow(ctx, currentBranch, remoteURL)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	if ctx.IsVerbose() {
		workflowYAML, _ := wf.Render()
		ctx.Output().Debugf("Generated workflow YAML:\n%s", workflowYAML)
	}

	workflowName, err := wf.Submit(ctx.GoContext(), argoClient)
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %w", err)
	}

	ctx.Output().Successf("Workflow submitted: %s", workflowName)

	if ctx.ShouldFollow() {
		if err := argoClient.FollowLogs(argo.K8sContext{Name: wf.KubeContext, Namespace: wf.Namespace}, workflowName); err != nil {
			notify.NewClient(ctx).Error("command")
			return fmt.Errorf("argo logs failed: %w", err)
		}
		notify.NewClient(ctx).Success("command")
	} else {
		ctx.Output().Infof("To follow logs, run: argo logs -n %s %s -f", wf.Namespace, workflowName)
	}

	return nil
}
