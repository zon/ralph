package cmd

import (
	"context"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
	orchestrationComment "github.com/zon/ralph/internal/orchestration/comment"
	orchestrationMerge "github.com/zon/ralph/internal/orchestration/merge"
)

// ---------------------------------------------------------------------------
// Comment adapters
// ---------------------------------------------------------------------------

type commentAIClient struct {
	ctx *execcontext.Context
}

func (c *commentAIClient) RunAgent(prompt string) error {
	return ai.RunAgent(c.ctx, prompt)
}

type commentServicesClient struct {
	manager *services.Manager
}

func (c *commentServicesClient) Start(svcs []config.Service) error {
	mgr := services.NewManager()
	if _, err := mgr.Start(svcs); err != nil {
		return err
	}
	c.manager = mgr
	return nil
}

func (c *commentServicesClient) Stop() {
	if c.manager != nil {
		c.manager.Stop()
	}
}

func newOrchestrationCommentCmd(ctx *execcontext.Context) *orchestrationComment.CommentCmd {
	return orchestrationComment.NewCommentCmd(
		&commentAIClient{ctx: ctx},
		&commentServicesClient{},
	)
}

// ---------------------------------------------------------------------------
// Merge adapters
// ---------------------------------------------------------------------------

type mergeGitClient struct{}

func (c *mergeGitClient) CurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

func (c *mergeGitClient) RevParse(rev string) (string, error) {
	return git.RevParse(rev)
}

func (c *mergeGitClient) Push(branch string) error {
	_, err := git.Push(nil, branch)
	return err
}

type mergeGitHubClient struct {
	gh github.GHClient
}

func (c *mergeGitHubClient) MergePR(pr, repo string) error {
	return c.gh.MergePR(pr, repo)
}

func (c *mergeGitHubClient) GetPRHeadRefOid(pr string) (string, error) {
	return c.gh.GetPRHeadRefOid(pr)
}

type mergeProjectClient struct{}

func (c *mergeProjectClient) FindCompleteProjects(dir string) ([]string, error) {
	return project.FindCompleteProjects(dir)
}

func (c *mergeProjectClient) RemoveAndCommit(files []string) error {
	return project.RemoveAndCommit(nil, files)
}

type mergeWorkflowClient struct {
	argoClient argo.Client
}

func (c *mergeWorkflowClient) SubmitMergeWorkflow(branch string) (string, error) {
	mw, err := workflow.GenerateMergeWorkflow(branch)
	if err != nil {
		return "", err
	}
	return mw.Submit(context.Background(), c.argoClient)
}

func newMergeWorkflowClient() *mergeWorkflowClient {
	return &mergeWorkflowClient{argoClient: argo.NewClient()}
}

func newOrchestrationMergeCmd() *orchestrationMerge.MergeCmd {
	return orchestrationMerge.NewMergeCmd(
		&mergeGitClient{},
		&mergeGitHubClient{gh: &github.GH{}},
		&mergeProjectClient{},
		newMergeWorkflowClient(),
	)
}


