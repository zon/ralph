package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

// ---------------------------------------------------------------------------
// Merge adapters (used by the legacy top-level merge command)
// ---------------------------------------------------------------------------

type legacyMergeGitClient struct{}

func (c *legacyMergeGitClient) CurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

func (c *legacyMergeGitClient) RevParse(rev string) (string, error) {
	return git.RevParse(rev)
}

func (c *legacyMergeGitClient) Push(branch string) error {
	_, err := git.Push(nil, branch)
	return err
}

type legacyMergeGitHubClient struct {
	gh github.GHClient
}

func (c *legacyMergeGitHubClient) MergePR(pr, repo string) error {
	return c.gh.MergePR(pr, repo)
}

func (c *legacyMergeGitHubClient) GetPRHeadRefOid(pr string) (string, error) {
	return c.gh.GetPRHeadRefOid(pr)
}

type legacyMergeProjectClient struct{}

func (c *legacyMergeProjectClient) FindCompleteProjects(dir string) ([]string, error) {
	return project.FindCompleteProjects(dir)
}

func (c *legacyMergeProjectClient) RemoveAndCommit(files []string) error {
	return project.RemoveAndCommit(nil, files)
}

type legacyMergeWorkflowClient struct {
	argoClient argo.Client
}

func (c *legacyMergeWorkflowClient) SubmitMergeWorkflow(branch string) (string, error) {
	repo, err := github.GetRepo(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	mw, err := workflow.GenerateMergeWorkflow(branch, repo.CloneURL(), currentBranch)
	if err != nil {
		return "", err
	}
	return mw.Submit(context.Background(), c.argoClient)
}

func newLegacyMergeWorkflowClient() *legacyMergeWorkflowClient {
	return &legacyMergeWorkflowClient{argoClient: argo.NewClient()}
}

type legacyMergeCmd struct {
	git     *legacyMergeGitClient
	github  *legacyMergeGitHubClient
	project *legacyMergeProjectClient
	wf      *legacyMergeWorkflowClient
}

func newLegacyMergeCmd() *legacyMergeCmd {
	return &legacyMergeCmd{
		git:     &legacyMergeGitClient{},
		github:  &legacyMergeGitHubClient{gh: github.NewGH(output.NewClient(os.Stdout, os.Stderr, false))},
		project: &legacyMergeProjectClient{},
		wf:      newLegacyMergeWorkflowClient(),
	}
}
