package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

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
	for i := 0; i < 30; i++ {
		oid, err := gh.GetPRHeadRefOid(prBranch)
		if err == nil && oid != "" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("head sync timeout for branch %s", prBranch)
}

func (c *workflowMergeGitHubClient) MergePR(prNumber int) error {
	gh := github.NewGH(output.NewClient(os.Stdout, os.Stderr, false))
	return gh.MergePR(fmt.Sprintf("%d", prNumber), "")
}

// ---------------------------------------------------------------------------
// workflowMergeProjectClient implements orchestration/merge.ProjectClient
// ---------------------------------------------------------------------------

type workflowMergeProjectClient struct{}

func (c *workflowMergeProjectClient) LoadAll() ([]*project.Project, error) {
	var projects []*project.Project
	err := filepath.WalkDir("projects", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		proj, err := project.LoadProject(path)
		if err != nil {
			return nil
		}
		projects = append(projects, proj)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk projects directory: %w", err)
	}
	return projects, nil
}

func (c *workflowMergeProjectClient) FilterPassing(projects []*project.Project) []*project.Project {
	var passing []*project.Project
	for _, p := range projects {
		if project.IsProjectComplete(p) {
			passing = append(passing, p)
		}
	}
	return passing
}

func (c *workflowMergeProjectClient) DeleteAll(projects []*project.Project) error {
	for _, p := range projects {
		if err := os.Remove(p.Path); err != nil {
			return fmt.Errorf("failed to delete project file %s: %w", p.Path, err)
		}
	}
	return nil
}
