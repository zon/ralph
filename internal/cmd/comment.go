package cmd

import (
	"os"

	"github.com/zon/ralph/internal/output"
	orchestrationComment "github.com/zon/ralph/internal/orchestration/comment"
)

// CommentCmd is the command for running a comment-triggered development iteration
type CommentCmd struct {
	Body       string `arg:"" help:"Comment body text"`
	Repo       string `help:"Repository in owner/repo format, e.g. zon/ralph" required:""`
	Branch     string `help:"PR branch name" required:""`
	PR         string `help:"Pull request number" required:""`
	Verbose    bool   `help:"Enable verbose logging" default:"false"`
	NoServices bool   `help:"Skip service startup" default:"false"`

	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the comment command (implements kong.Run interface)
func (c *CommentCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetVerbose(c.Verbose)
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, c.Verbose))
	ctx.SetNoServices(c.NoServices)
	ctx.SetNoNotify(true)

	flags := orchestrationComment.CommentFlags{
		Body:       c.Body,
		Repo:       c.Repo,
		Branch:     c.Branch,
		PR:         c.PR,
		NoServices: c.NoServices,
		Verbose:    c.Verbose,
	}

	cmd := newOrchestrationCommentCmd(ctx)
	return cmd.Run(flags)
}
