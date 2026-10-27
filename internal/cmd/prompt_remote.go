package cmd

import (
	"github.com/zon/ralph/internal/argo"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
	internalwf "github.com/zon/ralph/internal/workflow"
)

// promptRemoteSubmission carries the values resolved locally for one remote
// prompt: the branch the container clones, the prompt branch, the base branch,
// and the prompt text read locally.
type promptRemoteSubmission struct {
	CloneBranch  string
	PromptBranch string
	BaseBranch   string
	Prompt       string
}

// promptRemoteClient submits and follows the remote prompt workflow.
type promptRemoteClient interface {
	Submit(submission promptRemoteSubmission) (string, error)
	PrintLogHint(workflowName string)
	FollowLogs(workflowName string) error
}

// promptNotifyClient sends the completion notification for a followed remote
// prompt.
type promptNotifyClient interface {
	Success(slug string)
	Error(slug string)
}

// promptWorkflowClient generates and submits a ralph-prompt- workflow. It
// retains the workflow's Kubernetes target so followed logs use the same
// context and namespace as the submission.
type promptWorkflowClient struct {
	ctx         *execcontext.Context
	argoClient  argo.Client
	namespace   string
	kubeContext string
}

// Submit generates the prompt workflow from the locally resolved values and
// submits it to Argo, returning the generated workflow name.
func (c *promptWorkflowClient) Submit(submission promptRemoteSubmission) (string, error) {
	var remoteURL string
	owner, name := c.ctx.RepoOwnerAndName()
	if owner != "" {
		remoteURL = githubpkg.CloneURL(owner, name)
	} else {
		var err error
		remoteURL, err = git.RemoteURL()
		if err != nil {
			return "", err
		}
	}

	wf, err := internalwf.GeneratePromptWorkflow(
		c.ctx,
		submission.PromptBranch,
		submission.BaseBranch,
		submission.Prompt,
		submission.CloneBranch,
		remoteURL,
	)
	if err != nil {
		return "", err
	}

	if c.ctx.IsVerbose() {
		yaml, _ := wf.Render()
		c.ctx.Output().Debugf("Generated workflow YAML:\n%s", yaml)
	}

	c.namespace = wf.Namespace
	c.kubeContext = wf.KubeContext

	workflowName, err := wf.Submit(c.ctx.GoContext(), c.argoClient)
	if err != nil {
		return "", err
	}
	c.ctx.Output().Successf("Workflow submitted: %s", workflowName)
	return workflowName, nil
}

// PrintLogHint prints the argo logs command for a submitted prompt workflow.
func (c *promptWorkflowClient) PrintLogHint(workflowName string) {
	c.ctx.Output().Infof("To follow logs, run: argo logs -n %s %s -f", c.namespace, workflowName)
}

// FollowLogs streams the submitted prompt workflow logs and waits for it to
// finish.
func (c *promptWorkflowClient) FollowLogs(workflowName string) error {
	return c.argoClient.Logs(argo.K8sContext{Name: c.kubeContext, Namespace: c.namespace}, workflowName, true)
}
