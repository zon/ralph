package cmd

import (
	"fmt"
	"os"
	"os/exec"

	execcontext "github.com/zon/ralph/internal/context"
	orchestrationCommand "github.com/zon/ralph/internal/orchestration/command"
	"github.com/zon/ralph/internal/output"
)

type WorkflowCommandCmd struct {
	Repo        string   `help:"GitHub repository (owner/repo)" required:""`
	CloneBranch string   `help:"Branch to clone"`
	BotName     string   `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail    string   `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`
	Command     []string `arg:"" name:"command" help:"Command to run" required:""`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowCommandCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	cloneBranch := w.CloneBranch
	if cloneBranch == "" {
		cloneBranch = os.Getenv("GIT_BRANCH")
	}

	cmd := newOrchestrationWorkflowCommandCmd(ctx)
	flags := orchestrationCommand.WorkflowCommandFlags{
		Repo:        w.Repo,
		CloneBranch: cloneBranch,
		BotName:     w.BotName,
		BotEmail:    w.BotEmail,
		Command:     w.Command,
	}
	return cmd.Run(flags)
}

func newOrchestrationWorkflowCommandCmd(ctx *execcontext.Context) *orchestrationCommand.WorkflowCommandCmd {
	return orchestrationCommand.NewWorkflowCommandCmd(
		&workspaceSetupAdapter{ctx: ctx},
		&workflowCommandExecClient{},
	)
}

// ---------------------------------------------------------------------------
// workflowCommandExecClient implements orchestration/command.ExecClient
// ---------------------------------------------------------------------------

type workflowCommandExecClient struct{}

func (c *workflowCommandExecClient) Run(tokens []string) error {
	if len(tokens) == 0 {
		return fmt.Errorf("command cannot be empty")
	}
	cmd := exec.Command(tokens[0], tokens[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
