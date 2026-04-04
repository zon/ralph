package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/run"
	"github.com/zon/ralph/internal/workflow"
)

// MergeCmd is the command for merging a completed PR
type MergeCmd struct {
	Branch  string `arg:"" help:"PR branch name to merge"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	Local   bool   `help:"Run merge locally instead of submitting an Argo workflow" default:"false"`
	PR      string `help:"Pull request number" required:""`
	Repo    string `help:"GitHub repository (owner/repo); defaults to repo detected from git remote" default:""`

	cleanupRegistrar func(func()) `kong:"-"`
	// ghMerger is called to merge the PR; defaults to the real gh CLI implementation.
	// Tests inject a fake to avoid invoking gh.
	ghMerger func(pr, repo string) error `kong:"-"`
}

// Run executes the merge command (implements kong.Run interface)
func (m *MergeCmd) Run() error {
	if m.Local {
		return m.runLocal()
	}

	// Generate the merge workflow
	mw, err := workflow.GenerateMergeWorkflow(m.Branch)
	if err != nil {
		return fmt.Errorf("failed to generate merge workflow: %w", err)
	}

	// Submit the workflow (does not wait for completion)
	workflowName, err := mw.Submit()
	if err != nil {
		return fmt.Errorf("failed to submit merge workflow: %w", err)
	}

	logger.Successf("Merge workflow submitted: %s", workflowName)
	return nil
}

// runLocal merges the PR locally using the gh CLI
func (m *MergeCmd) runLocal() error {
	ctx := m.createExecutionContext()

	if err := m.scanAndCleanupProjects(ctx); err != nil {
		return err
	}

	merger := m.ghMerger
	if merger == nil {
		merger = github.MergePR
	}
	return merger(m.PR, m.Repo)
}

func (m *MergeCmd) createExecutionContext() *context.Context {
	ctx := createExecutionContext()
	ctx.SetVerbose(m.Verbose)
	return ctx
}

func (m *MergeCmd) scanAndCleanupProjects(ctx *context.Context) error {
	projectsDir := "projects"
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		logger.Verbose("Projects directory not found, skipping complete project cleanup")
		return nil
	}

	completeProjects, err := run.FindCompleteProjects(projectsDir)
	if err != nil {
		return fmt.Errorf("failed to scan for complete projects: %w", err)
	}

	if len(completeProjects) == 0 {
		logger.Verbose("No complete projects found")
		return nil
	}

	logger.Infof("Found %d complete project(s) to clean up", len(completeProjects))
	for _, file := range completeProjects {
		relPath, err := filepath.Rel(".", file)
		if err != nil {
			relPath = file
		}
		logger.Infof("  - %s", relPath)
	}

	if err := run.RemoveAndCommit(ctx, completeProjects); err != nil {
		return fmt.Errorf("failed to remove complete projects: %w", err)
	}

	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	branch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if _, err := git.Push(auth, branch); err != nil {
		return fmt.Errorf("failed to push after removing complete projects: %w", err)
	}

	if err := waitForGitHubHead(m.PR); err != nil {
		return fmt.Errorf("failed waiting for GitHub to sync push: %w", err)
	}

	return nil
}

// waitForGitHubHead polls until GitHub's view of the PR head SHA matches the local HEAD.
// This prevents "Head branch is out of date" errors when merging immediately after a push.
func waitForGitHubHead(pr string) error {
	localSHA, err := git.RevParse("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get local HEAD: %w", err)
	}

	const maxAttempts = 20
	const pollInterval = 3 * time.Second

	for i := range maxAttempts {
		headRefOid, err := github.GetPRHeadRefOid(pr)
		if err != nil {
			return fmt.Errorf("failed to query PR head: %w", err)
		}

		if strings.HasPrefix(headRefOid, localSHA) || strings.HasPrefix(localSHA, headRefOid) {
			logger.Verbosef("GitHub head SHA matches local HEAD (%s)", localSHA[:8])
			return nil
		}

		if i < maxAttempts-1 {
			logger.Verbosef("Waiting for GitHub to sync push (attempt %d/%d, local=%s, remote=%s)...",
				i+1, maxAttempts, localSHA[:8], headRefOid[:8])
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("timed out waiting for GitHub to sync push (local HEAD: %s)", localSHA[:8])
}
