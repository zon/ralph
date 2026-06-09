package cmd

import (
	"os"

	"github.com/zon/ralph/internal/argo"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/output"
	orchestrationCommand "github.com/zon/ralph/internal/orchestration/command"
	internalwf "github.com/zon/ralph/internal/workflow"
)

type CommandCmd struct {
	Command  []string `arg:"" name:"command" help:"Command to run" optional:""`
	NoFollow bool     `help:"Skip following workflow logs" name:"no-follow" default:"false"`
	Verbose  bool     `help:"Enable verbose logging" default:"false"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (c *CommandCmd) Run() error {
	ctx := c.newExecutionContext()
	client := &commandWorkflowClient{
		ctx:        ctx,
		argoClient: argo.NewClient(),
	}
	cmd := orchestrationCommand.NewCommandCmd(client)
	flags := orchestrationCommand.CommandFlags{
		Command:  c.Command,
		NoFollow: c.NoFollow,
	}
	return cmd.Run(flags)
}

func (c *CommandCmd) newExecutionContext() *execcontext.Context {
	ctx := createExecutionContext()
	ctx.SetCommand(c.Command)
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, c.Verbose))
	return ctx
}

type commandWorkflowClient struct {
	ctx         *execcontext.Context
	argoClient  argo.Client
	namespace   string
	kubeContext string
}

func (c *commandWorkflowClient) Submit(command []string) (string, error) {
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	if err := git.IsBranchSyncedWithRemote(currentBranch); err != nil {
		return "", err
	}

	var remoteURL string
	owner, name := c.ctx.RepoOwnerAndName()
	if owner != "" {
		remoteURL = githubpkg.CloneURL(owner, name)
	} else {
		remoteURL, err = git.RemoteURL()
		if err != nil {
			return "", err
		}
	}

	wf, err := internalwf.GenerateCommandWorkflow(c.ctx, currentBranch, remoteURL)
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

func (c *commandWorkflowClient) StreamLogs(workflowName string) error {
	return c.argoClient.FollowLogs(argo.K8sContext{Name: c.kubeContext, Namespace: c.namespace}, workflowName)
}
