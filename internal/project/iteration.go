package project

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/requirement"
)

// ErrNoChanges is returned by CommitChanges when there are no staged changes to commit
var ErrNoChanges = errors.New("no changes to commit")

// RunIterationLoop runs multiple development iterations until completion or max iterations
// Each iteration:
// 1. Runs a single development iteration (requirement.Execute)
// 2. Commits the changes
// 3. Checks project completion status
// 4. Stops when all requirements pass OR max iterations reached
//
// Returns the final iteration count and any error encountered
func RunIterationLoop(ctx *context.Context, cleanupRegistrar func(func())) (int, error) {
	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations)

	// Load initial project state to track requirement completions
	previousProject, err := config.LoadProject(ctx.ProjectFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load initial project state: %w", err)
	}

	iterationCount := 0

	for i := 1; i <= ctx.MaxIterations; i++ {
		iterationCount = i

		logger.Verbose("")
		logger.Verbosef("=== Iteration %d/%d ===", i, ctx.MaxIterations)

		// Run single development iteration
		logger.Verbose("Running development iteration...")
		if err := requirement.Execute(ctx, cleanupRegistrar); err != nil {
			return iterationCount, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		// Commit changes after iteration
		logger.Verbosef("Committing changes from iteration %d...", i)
		if err := CommitChanges(ctx, i); err != nil {
			if errors.Is(err, ErrNoChanges) {
				logger.Verbosef("No changes to commit after iteration %d", i)
			} else {
				return iterationCount, fmt.Errorf("iteration %d commit failed: %w", i, err)
			}
		} else {
			logger.Verbosef("Committed changes from iteration %d", i)
		}

		// Load and check project completion status
		currentProject, err := config.LoadProject(ctx.ProjectFile)
		if err != nil {
			return iterationCount, fmt.Errorf("failed to reload project after iteration %d: %w", i, err)
		}

		allComplete, passingCount, failingCount := config.CheckCompletion(currentProject)
		logger.Verbosef("Status after iteration %d: %d passing, %d failing", i, passingCount, failingCount)

		// Check for newly completed requirements and show them immediately after AI completes
		for idx, req := range currentProject.Requirements {
			if req.Passing && !previousProject.Requirements[idx].Passing {
				// This requirement just passed
				description := req.Description
				if description == "" {
					description = req.Name
				}
				if description == "" {
					description = req.Category
				}
				logger.Successf("Requirement complete: %s", description)
			}
		}

		// Stop if all requirements are passing
		if allComplete {
			logger.Success("All requirements complete")
			break
		}

		// Update previous state for next iteration
		previousProject = currentProject

		// Continue to next iteration if not at max
		if i < ctx.MaxIterations {
			logger.Verbose("Requirements not complete, continuing to next iteration...")
		}
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterationCount)

	return iterationCount, nil
}

// CommitChanges stages all changes and commits them using report.md as the commit message
func CommitChanges(ctx *context.Context, iteration int) error {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would commit changes from iteration %d", iteration)
		return nil
	}

	// Read and remove report.md before staging so it is not included in the commit
	reportPath := "report.md"
	commitMsg, err := os.ReadFile(reportPath)
	if err != nil {
		// If report.md doesn't exist, fall back to iteration-based message
		logger.Warningf("Failed to read report.md: %v, using fallback message", err)
		commitMsg = []byte(fmt.Sprintf("Development iteration %d", iteration))
	} else {
		if err := os.Remove(reportPath); err != nil {
			logger.Warningf("Failed to remove report.md: %v", err)
		} else if ctx.IsVerbose() {
			logger.Verbose("Removed report.md")
		}
	}

	// Stage all changes
	if err := git.StageAll(ctx); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are any staged changes to commit
	if !git.HasStagedChanges(ctx) {
		return ErrNoChanges
	}

	// Clean up and validate commit message
	message := strings.TrimSpace(string(commitMsg))
	if message == "" {
		message = fmt.Sprintf("Development iteration %d", iteration)
	}

	// Commit the staged changes
	if err := git.Commit(ctx, message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Infof("Committed with message: %s", message)
	}

	// Push commit to origin
	logger.Verbose("Pushing commit to origin...")
	if err := git.PushCurrentBranch(ctx); err != nil {
		return fmt.Errorf("failed to push commit: %w", err)
	}
	logger.Verbose("Pushed commit to origin")

	return nil
}
