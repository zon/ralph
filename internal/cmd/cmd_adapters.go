package cmd

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/architecture"
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
	"github.com/zon/ralph/internal/workflow"
	orchestrationArchitecture "github.com/zon/ralph/internal/orchestration/architecture"
	orchestrationComment "github.com/zon/ralph/internal/orchestration/comment"
	orchestrationMerge "github.com/zon/ralph/internal/orchestration/merge"
	orchestrationReview "github.com/zon/ralph/internal/orchestration/review"
)

// ---------------------------------------------------------------------------
// Review adapters
// ---------------------------------------------------------------------------

type reviewAIClient struct {
	ctx *execcontext.Context
}

func (c *reviewAIClient) BuildReviewItemPrompt(content string) (string, error) {
	return ai.BuildReviewItemPrompt(content)
}

func (c *reviewAIClient) BuildLoopItemPrompt(content, funcName, funcPath string) (string, error) {
	return ai.BuildLoopItemPrompt(content, funcName, funcPath)
}

func (c *reviewAIClient) RunAgent(prompt string) error {
	return ai.RunAgent(c.ctx, prompt)
}

func (c *reviewAIClient) DisplayStats() error {
	return ai.DisplayStats()
}

func (c *reviewAIClient) GenerateReviewPRBody(slug, title string, requirementSummaries []string) (string, error) {
	return ai.GenerateReviewPRBody(c.ctx, slug, title, requirementSummaries)
}

func (c *reviewAIClient) SetModel(model string) {
	c.ctx.SetModel(model)
}

type reviewGitClient struct {
	ctx *execcontext.Context
}

func (c *reviewGitClient) CurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

func (c *reviewGitClient) HasUncommittedChanges() bool {
	return git.HasUncommittedChanges()
}

func (c *reviewGitClient) CommitAllAndPush(branch, commitMsg string) error {
	return git.CommitAllAndPush(nil, branch, commitMsg)
}

func (c *reviewGitClient) DetectModifiedProjectFile(dir string) (string, error) {
	return git.DetectModifiedProjectFile(dir)
}

func (c *reviewGitClient) IsBranchSyncedWithRemote(branch string) error {
	return git.IsBranchSyncedWithRemote(branch)
}

func (c *reviewGitClient) TmpPath(filename string) (string, error) {
	return git.TmpPath(filename)
}

type reviewGitHubClient struct {
	ctx *execcontext.Context
}

func (c *reviewGitHubClient) CreatePullRequest(proj *project.Project, reviewName, baseBranch, body string) (string, error) {
	return github.CreatePullRequest(c.ctx, proj, reviewName, baseBranch, body)
}

type reviewWorkflowClient struct {
	ctx          *execcontext.Context
	lastWorkflow *workflow.Workflow
}

func (c *reviewWorkflowClient) SubmitReview(cloneBranch string) (string, error) {
	wf, err := workflow.GenerateReviewWorkflow(c.ctx, cloneBranch)
	if err != nil {
		return "", err
	}
	c.lastWorkflow = wf
	return wf.Submit()
}

func (c *reviewWorkflowClient) FollowLogs(workflowName string) error {
	return argo.FollowLogs(c.lastWorkflow.Namespace, workflowName, c.lastWorkflow.KubeContext)
}

func newOrchestrationReviewCmd(ctx *execcontext.Context) *orchestrationReview.ReviewCmd {
	return orchestrationReview.NewReviewCmd(
		&reviewAIClient{ctx: ctx},
		&reviewGitClient{ctx: ctx},
		&reviewGitHubClient{ctx: ctx},
		&reviewWorkflowClient{ctx: ctx},
	)
}

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

type mergeGitHubClient struct{}

func (c *mergeGitHubClient) MergePR(pr, repo string) error {
	return github.MergePR(pr, repo)
}

func (c *mergeGitHubClient) GetPRHeadRefOid(pr string) (string, error) {
	return github.GetPRHeadRefOid(pr)
}

type mergeProjectClient struct{}

func (c *mergeProjectClient) FindCompleteProjects(dir string) ([]string, error) {
	return project.FindCompleteProjects(dir)
}

func (c *mergeProjectClient) RemoveAndCommit(files []string) error {
	return project.RemoveAndCommit(nil, files)
}

type mergeWorkflowClient struct{}

func (c *mergeWorkflowClient) SubmitMergeWorkflow(branch string) (string, error) {
	mw, err := workflow.GenerateMergeWorkflow(branch)
	if err != nil {
		return "", err
	}
	return mw.Submit()
}

func newOrchestrationMergeCmd() *orchestrationMerge.MergeCmd {
	return orchestrationMerge.NewMergeCmd(
		&mergeGitClient{},
		&mergeGitHubClient{},
		&mergeProjectClient{},
		&mergeWorkflowClient{},
	)
}

// ---------------------------------------------------------------------------
// Architecture adapters
// ---------------------------------------------------------------------------

type architectureAIClient struct {
	ctx *execcontext.Context
}

func (c *architectureAIClient) BuildArchitecturePrompt(output string) (string, error) {
	return ai.BuildArchitecturePrompt(output)
}

func (c *architectureAIClient) BuildArchitectureFixPrompt(output string, errors []string) (string, error) {
	return ai.BuildArchitectureFixPrompt(output, errors)
}

func (c *architectureAIClient) RunAgent(prompt string) error {
	return ai.RunAgent(c.ctx, prompt)
}

type architectureGitClient struct {
	ctx *execcontext.Context
}

func (c *architectureGitClient) IsFileModifiedOrNew(path string) bool {
	return git.IsFileModifiedOrNew(path)
}

func (c *architectureGitClient) CheckoutOrCreateBranch(name string) error {
	return git.CheckoutOrCreateBranch(name)
}

func (c *architectureGitClient) StageFile(path string) error {
	return git.StageFile(path)
}

func (c *architectureGitClient) CommitAllAndPush(auth *git.AuthConfig, branchName, commitMsg string) error {
	return git.CommitAllAndPush(auth, branchName, commitMsg)
}

type architectureGitHubClient struct {
	ctx *execcontext.Context
}

func (c *architectureGitHubClient) CreatePullRequest(proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
	return github.CreatePullRequest(c.ctx, proj, branchName, baseBranch, prSummary)
}

type archClient struct{}

func (c *archClient) Load(path string) (*architecture.Architecture, error) {
	return architecture.Load(path)
}

func newOrchestrationArchitectureCmd(ctx *execcontext.Context) *orchestrationArchitecture.ArchitectureCmd {
	return orchestrationArchitecture.NewArchitectureCmd(
		&architectureAIClient{ctx: ctx},
		&architectureGitClient{ctx: ctx},
		&architectureGitHubClient{ctx: ctx},
		&archClient{},
	)
}
