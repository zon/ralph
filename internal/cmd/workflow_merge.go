package cmd

import (
	"os"
	"strconv"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	orchestrationMerge "github.com/zon/ralph/internal/orchestration/merge"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

type WorkflowMergeCmd struct {
	Repo        string `help:"GitHub repository (owner/repo)" required:""`
	CloneBranch string `help:"Branch to clone"`
	PRBranch    string `help:"PR branch name" name:"pr-branch" required:""`
	PRNumber    int    `help:"Pull request number" name:"pr" required:""`
	BotName     string `help:"Git user name for commits" default:"ralph-zon[bot]"`
	BotEmail    string `help:"Git user email for commits" default:"ralph-zon[bot]@users.noreply.github.com"`

	cleanupRegistrar func(func()) `kong:"-"`
}

func (w *WorkflowMergeCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	ctx.SetNoNotify(true)
	ctx.SetWorkflowExecution(true)

	cloneBranch := w.CloneBranch
	if cloneBranch == "" {
		cloneBranch = os.Getenv("GIT_BRANCH")
	}

	cmd := newOrchestrationWorkflowMergeCmd(ctx)
	flags := orchestrationMerge.WorkflowMergeFlags{
		Repo:        w.Repo,
		CloneBranch: cloneBranch,
		PRBranch:    w.PRBranch,
		PRNumber:    w.PRNumber,
		BotName:     w.BotName,
		BotEmail:    w.BotEmail,
	}
	return cmd.Merge(flags)
}

func newOrchestrationWorkflowMergeCmd(ctx *execcontext.Context) *orchestrationMerge.WorkflowMergeCmd {
	return orchestrationMerge.NewWorkflowMergeCmd(
		&workspaceSetupAdapter{ctx: ctx},
		&workflowMergeGitClient{},
		&workflowMergeGitHubClient{},
		&workflowMergeProjectClient{},
	)
}

// ---------------------------------------------------------------------------
// workflowMergeGitClient implements orchestration/merge.GitClient
// ---------------------------------------------------------------------------

type workflowMergeGitClient struct{}

func (c *workflowMergeGitClient) CommitAndPush(message string) error {
	return git.CommitChanges(true, "", "", message)
}

// ---------------------------------------------------------------------------
// workflowMergeGitHubClient implements orchestration/merge.GitHubClient
// ---------------------------------------------------------------------------

type workflowMergeGitHubClient struct{}

func (c *workflowMergeGitHubClient) WaitForHeadSync(prBranch string) error {
	gh := github.NewGH(output.NewClient(os.Stdout, os.Stderr, false))
	return github.WaitForHeadSync(gh, prBranch)
}

func (c *workflowMergeGitHubClient) MergePR(prNumber int) error {
	gh := github.NewGH(output.NewClient(os.Stdout, os.Stderr, false))
	return gh.MergePR(strconv.Itoa(prNumber), "")
}

// ---------------------------------------------------------------------------
// workflowMergeProjectClient implements orchestration/merge.ProjectClient
// ---------------------------------------------------------------------------

type workflowMergeProjectClient struct{}

func (c *workflowMergeProjectClient) LoadAll() ([]*project.Project, error) {
	return project.LoadAll("projects")
}

func (c *workflowMergeProjectClient) FilterPassing(projects []*project.Project) []*project.Project {
	return project.FilterPassing(projects)
}

func (c *workflowMergeProjectClient) DeleteAll(projects []*project.Project) error {
	return project.DeleteAll(projects)
}
