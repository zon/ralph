package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

// MergeCmd is the command for merging a completed PR
type MergeCmd struct {
	Branch  string `arg:"" help:"PR branch name to merge"`
	DryRun  bool   `help:"Simulate execution without making changes" default:"false"`
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

	if m.DryRun {
		logger.Infof("Dry run: would submit merge workflow for branch %s", m.Branch)
		if m.Verbose {
			workflowYAML, _ := mw.Render()
			fmt.Println(workflowYAML)
		}
		return nil
	}

	// Submit the workflow (does not wait for completion)
	workflowName, err := mw.Submit(mw.RalphConfig.Workflow.Namespace)
	if err != nil {
		return fmt.Errorf("failed to submit merge workflow: %w", err)
	}

	logger.Successf("Merge workflow submitted: %s", workflowName)
	return nil
}

// runLocal merges the PR locally using the gh CLI
func (m *MergeCmd) runLocal() error {
	// Create context for project operations
	ctx := &context.Context{}
	ctx.SetDryRun(m.DryRun)
	ctx.SetVerbose(m.Verbose)

	// Scan the projects/ directory for complete projects
	projectsDir := "projects"
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		// projects directory doesn't exist, proceed with merge
		logger.Verbose("Projects directory not found, skipping complete project cleanup")
	} else {
		completeProjects, err := project.FindCompleteProjects(projectsDir)
		if err != nil {
			return fmt.Errorf("failed to scan for complete projects: %w", err)
		}

		if len(completeProjects) > 0 {
			logger.Infof("Found %d complete project(s) to clean up", len(completeProjects))
			for _, file := range completeProjects {
				relPath, err := filepath.Rel(".", file)
				if err != nil {
					relPath = file
				}
				logger.Infof("  - %s", relPath)
			}

			// Remove and commit complete project files
			if err := project.RemoveAndCommit(ctx, completeProjects); err != nil {
				return fmt.Errorf("failed to remove complete projects: %w", err)
			}

			// Push the removal commit so it is included in the merge
			if err := git.PushCurrentBranch(ctx); err != nil {
				return fmt.Errorf("failed to push after removing complete projects: %w", err)
			}

			// Wait for GitHub to recognize the push before merging
			if !m.DryRun {
				if err := waitForGitHubHead(m.PR); err != nil {
					return fmt.Errorf("failed waiting for GitHub to sync push: %w", err)
				}
			}
		} else {
			logger.Verbose("No complete projects found")
		}
	}

	if m.DryRun {
		logger.Infof("Dry run: would merge PR #%s and delete branch %s", m.PR, m.Branch)
		return nil
	}

	merger := m.ghMerger
	if merger == nil {
		merger = ghMerge
	}
	return merger(m.PR, m.Repo)
}

// ghMerge is the default implementation that calls the gh CLI to merge a PR.
// It first attempts to enable auto-merge; if GitHub rejects that because the PR is
// already in a "clean" (immediately mergeable) state, it falls back to a direct merge.
func ghMerge(pr, repo string) error {
	autoArgs := []string{"pr", "merge", pr, "--merge", "--delete-branch", "--auto"}
	if repo != "" {
		autoArgs = append(autoArgs, "--repo", repo)
	}
	var autoOut bytes.Buffer
	autoCmd := exec.Command("gh", autoArgs...)
	autoCmd.Stdout = os.Stdout
	autoCmd.Stderr = &autoOut
	if err := autoCmd.Run(); err != nil {
		// GitHub rejects enablePullRequestAutoMerge when the PR is already in a "clean"
		// state (all checks passed, ready to merge). In that case, merge immediately
		// without --auto instead.
		if strings.Contains(autoOut.String(), "clean status") {
			logger.Verbosef("PR #%s is already mergeable, merging immediately", pr)
			immediateArgs := []string{"pr", "merge", pr, "--merge", "--delete-branch"}
			if repo != "" {
				immediateArgs = append(immediateArgs, "--repo", repo)
			}
			var immediateOut bytes.Buffer
			immediateCmd := exec.Command("gh", immediateArgs...)
			immediateCmd.Stdout = os.Stdout
			immediateCmd.Stderr = &immediateOut
			if err := immediateCmd.Run(); err != nil {
				fmt.Fprint(os.Stderr, immediateOut.String())
				return fmt.Errorf("failed to merge PR #%s: %w", pr, err)
			}
			logger.Successf("Merged PR #%s", pr)
			return nil
		}
		fmt.Fprint(os.Stderr, autoOut.String())
		return fmt.Errorf("failed to merge PR #%s: %w", pr, err)
	}

	logger.Successf("Auto-merge enabled for PR #%s", pr)
	return nil
}

// waitForGitHubHead polls until GitHub's view of the PR head SHA matches the local HEAD.
// This prevents "Head branch is out of date" errors when merging immediately after a push.
func waitForGitHubHead(pr string) error {
	// Get local HEAD SHA
	var localOut bytes.Buffer
	localCmd := exec.Command("git", "rev-parse", "HEAD")
	localCmd.Stdout = &localOut
	localCmd.Stderr = &localOut
	if err := localCmd.Run(); err != nil {
		return fmt.Errorf("failed to get local HEAD: %w", err)
	}
	localSHA := strings.TrimSpace(localOut.String())

	const maxAttempts = 20
	const pollInterval = 3 * time.Second

	for i := range maxAttempts {
		var ghOut bytes.Buffer
		ghCmd := exec.Command("gh", "pr", "view", pr, "--json", "headRefOid")
		ghCmd.Stdout = &ghOut
		ghCmd.Stderr = &ghOut
		if err := ghCmd.Run(); err != nil {
			return fmt.Errorf("failed to query PR head: %w (output: %s)", err, ghOut.String())
		}

		var result struct {
			HeadRefOid string `json:"headRefOid"`
		}
		if err := json.Unmarshal(ghOut.Bytes(), &result); err != nil {
			return fmt.Errorf("failed to parse PR head response: %w", err)
		}

		if strings.HasPrefix(result.HeadRefOid, localSHA) || strings.HasPrefix(localSHA, result.HeadRefOid) {
			logger.Verbosef("GitHub head SHA matches local HEAD (%s)", localSHA[:8])
			return nil
		}

		if i < maxAttempts-1 {
			logger.Verbosef("Waiting for GitHub to sync push (attempt %d/%d, local=%s, remote=%s)...",
				i+1, maxAttempts, localSHA[:8], result.HeadRefOid[:8])
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("timed out waiting for GitHub to sync push (local HEAD: %s)", localSHA[:8])
}
