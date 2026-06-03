package cmd

import (
	"os"
	"strconv"

	"github.com/zon/ralph/internal/output"
	orchestrationComment "github.com/zon/ralph/internal/orchestration/comment"
)

type CommentCmd struct {
	Body       string `arg:"" help:"Comment body text"`
	Repo       string `help:"Repository in owner/repo format, e.g. zon/ralph" required:""`
	Branch     string `help:"PR branch name" required:""`
	PR         string `help:"Pull request number" required:""`
	Verbose    bool   `help:"Enable verbose logging" default:"false"`
	NoServices bool   `help:"Skip service startup" default:"false"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (c *CommentCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, c.Verbose))
	ctx.SetNoServices(c.NoServices)
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	owner, repoName := parseRepo(c.Repo)
	prNum, _ := strconv.Atoi(c.PR)

	cmd := orchestrationComment.NewWorkflowCommentCmd(
		&workspaceSetupAdapter{ctx: ctx},
		&configOptionalAdapter{},
		&workflowCommentAIClient{ctx: ctx},
		&workflowCommentServicesClient{ctx: ctx},
		&workflowCommentGitClient{},
		&workflowCommentGitHubClient{},
	)
	flags := orchestrationComment.WorkflowCommentFlags{
		Repo:          c.Repo,
		CloneBranch:   c.Branch,
		ProjectBranch: c.Branch,
		BotName:       "ralph-zon[bot]",
		BotEmail:      "ralph-zon[bot]@users.noreply.github.com",
		CommentBody:   c.Body,
		PRNumber:      prNum,
		RepoOwner:     owner,
		RepoName:      repoName,
		NoServices:    c.NoServices,
	}
	return cmd.Run(flags)
}
