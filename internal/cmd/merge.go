package cmd

import (
	"os"
	"strconv"

	"github.com/zon/ralph/internal/output"
	orchestrationMerge "github.com/zon/ralph/internal/orchestration/merge"
)

// MergeCmd is the command for merging a completed PR
type MergeCmd struct {
	Branch  string `arg:"" help:"PR branch name to merge"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	PR      string `help:"Pull request number" required:""`
	Repo    string `help:"GitHub repository (owner/repo); defaults to repo detected from git remote" default:""`

	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the merge command (implements kong.Run interface)
func (m *MergeCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, m.Verbose))
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	prNum, _ := strconv.Atoi(m.PR)

	cmd := orchestrationMerge.NewWorkflowMergeCmd(
		&workspaceSetupAdapter{ctx: ctx},
		&workflowMergeGitClient{},
		&workflowMergeGitHubClient{},
		&workflowMergeProjectClient{},
	)
	flags := orchestrationMerge.WorkflowMergeFlags{
		Repo:        m.Repo,
		CloneBranch: m.Branch,
		PRBranch:    m.Branch,
		PRNumber:    prNum,
		BotName:     "ralph-zon[bot]",
		BotEmail:    "ralph-zon[bot]@users.noreply.github.com",
	}
	return cmd.Merge(flags)
}
