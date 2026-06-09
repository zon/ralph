package github

import (
	gocontext "context"
	"errors"
	"fmt"

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

type Client struct {
	ctx               *context.Context
	baseBranch        string
	gh                GHClient
	oc                opencode.OCClient
	gitAuthConfigurer GitAuthConfigurer
}

func NewClient(ctx *context.Context, baseBranch string, gh GHClient, oc opencode.OCClient) *Client {
	return &Client{
		ctx:               ctx,
		baseBranch:        baseBranch,
		gh:                gh,
		oc:                oc,
		gitAuthConfigurer: &realGitAuthConfigurer{},
	}
}

func (a *Client) CreatePR(proj *project.Project) error {
	commitLog, err := git.GetCommitLog(a.baseBranch, 100)
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	allComplete, passingCount, failingCount := project.CheckCompletion(proj)
	projectStatus := fmt.Sprintf("%d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	prSummary, err := ai.GeneratePRSummary(a.ctx, a.oc, proj.Title, projectStatus, a.baseBranch, commitLog)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}

	branchName := git.SanitizeBranchName(proj.Slug)

	if a.ctx.IsWorkflowExecution() {
		owner, repoName := a.ctx.RepoOwnerAndName()
		if err := a.gitAuthConfigurer.ConfigureGitAuth(gocontext.Background(), owner, repoName, DefaultSecretsDir); err != nil {
			return fmt.Errorf("failed to refresh GitHub credentials before PR creation: %w", err)
		}
	}

	prURL, err := CreatePullRequest(a.ctx.Output(), a.gh, proj, branchName, a.baseBranch, prSummary)
	if err != nil {
		if errors.Is(err, ErrNoCommitsBetweenBranches) {
			a.ctx.Output().Debug("No commits ahead of base branch — all requirements were already passing; skipping PR creation")
			return nil
		}
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	a.ctx.Output().Info(prURL)
	return nil
}
