package github

import (
	gocontext "context"
	"errors"
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"

	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/project"
)

// GitAuthConfigurer configures git authentication for GitHub operations.
type GitAuthConfigurer interface {
	ConfigureGitAuth(ctx gocontext.Context, owner, repo, secretsDir string) error
}

type realGitAuthConfigurer struct{}

func (r *realGitAuthConfigurer) ConfigureGitAuth(ctx gocontext.Context, owner, repo, secretsDir string) error {
	return ConfigureGitAuth(ctx, owner, repo, secretsDir)
}

// CommitLogGetter retrieves the commit log of the commits on the current branch
// that are not on the base branch.
type CommitLogGetter interface {
	GetCommitLog(base string, limit int) (string, error)
}

type realCommitLog struct{}

func (realCommitLog) GetCommitLog(base string, limit int) (string, error) {
	return git.GetCommitLog(base, limit)
}

type Client struct {
	ctx               *context.Context
	baseBranch        string
	gh                GHClient
	oc                opencode.OCClient
	gitAuthConfigurer GitAuthConfigurer
	commitLog         CommitLogGetter
}

func NewClient(ctx *context.Context, baseBranch string, gh GHClient, oc opencode.OCClient) *Client {
	return &Client{
		ctx:               ctx,
		baseBranch:        baseBranch,
		gh:                gh,
		oc:                oc,
		gitAuthConfigurer: &realGitAuthConfigurer{},
		commitLog:         realCommitLog{},
	}
}

func (a *Client) CreatePR(proj *project.Project) error {
	commitLog, err := a.commitLog.GetCommitLog(a.baseBranch, 100)
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	if strings.TrimSpace(commitLog) == "" {
		a.ctx.Output().Debug("No commits ahead of base branch; skipping PR creation")
		return nil
	}

	prSummary, err := ai.GeneratePRSummary(a.ctx, a.oc, proj.Title, a.baseBranch, commitLog)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}

	return a.openPullRequest(proj, git.SanitizeBranchName(proj.Slug), a.baseBranch, prSummary)
}

// OpenLoopPullRequest opens a pull request from loop-<slug> to the base branch
// when the loop branch has commits ahead of the base. When there are no commits
// ahead of the base, no pull request is opened and the call succeeds.
func (a *Client) OpenLoopPullRequest(slug string) error {
	commitLog, err := a.commitLog.GetCommitLog(a.baseBranch, 100)
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}
	if strings.TrimSpace(commitLog) == "" {
		a.ctx.Output().Debug("No commits ahead of base branch; skipping PR creation")
		return nil
	}
	return a.openPullRequest(&project.Project{Slug: slug}, git.LoopBranch(slug), a.baseBranch, commitLog)
}

func (a *Client) openPullRequest(proj *project.Project, head, base, body string) error {
	if a.ctx.IsWorkflowExecution() {
		owner, repoName := a.ctx.RepoOwnerAndName()
		if err := a.gitAuthConfigurer.ConfigureGitAuth(gocontext.Background(), owner, repoName, DefaultSecretsDir); err != nil {
			return fmt.Errorf("failed to refresh GitHub credentials before PR creation: %w", err)
		}
	}

	prURL, err := CreatePullRequest(a.ctx.Output(), a.gh, proj, head, base, body)
	if err != nil {
		if errors.Is(err, ErrNoCommitsBetweenBranches) {
			a.ctx.Output().Debug("No commits ahead of base branch; skipping PR creation")
			return nil
		}
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	a.ctx.Output().Info(prURL)
	return nil
}
