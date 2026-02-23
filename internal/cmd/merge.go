package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	cleanupRegistrar func(func()) `kong:"-"`
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
	ctx := &context.Context{
		DryRun:  m.DryRun,
		Verbose: m.Verbose,
	}

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
		} else {
			logger.Verbose("No complete projects found")
		}
	}

	if m.DryRun {
		logger.Infof("Dry run: would merge PR #%s and delete branch %s", m.PR, m.Branch)
		return nil
	}

	cmd := exec.Command("gh", "pr", "merge", m.PR, "--merge", "--delete-branch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to merge PR #%s: %w", m.PR, err)
	}

	logger.Successf("Merged PR #%s and deleted branch %s", m.PR, m.Branch)
	return nil
}
