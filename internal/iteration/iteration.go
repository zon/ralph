package iteration

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/once"
)

// RunIterationLoop runs multiple development iterations until completion or max iterations
// Each iteration:
// 1. Runs the once/develop command internally
// 2. Commits the changes
// 3. Checks project completion status
// 4. Stops when all requirements pass OR max iterations reached
//
// Returns the final iteration count and any error encountered
func RunIterationLoop(ctx *context.Context, projectFile string, maxIters int, cleanupRegistrar func(func())) (int, error) {
	logger.Infof("Starting iteration loop (max: %d)", maxIters)
	logger.Info("==========================================")

	iterationCount := 0

	for i := 1; i <= maxIters; i++ {
		iterationCount = i

		logger.Info("")
		logger.Infof("=== Iteration %d/%d ===", i, maxIters)

		// Run single development iteration (once command logic)
		logger.Info("Running development iteration...")
		if err := once.Execute(ctx, projectFile, cleanupRegistrar); err != nil {
			return iterationCount, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		// Commit changes after iteration
		logger.Infof("Committing changes from iteration %d...", i)
		if err := CommitChanges(ctx, i); err != nil {
			// If there are no changes, it's not fatal - continue to next iteration
			logger.Warningf("Commit failed (may be no changes): %v", err)
		} else {
			logger.Successf("Committed changes from iteration %d", i)
		}

		// Load and check project completion status
		project, err := config.LoadProject(projectFile)
		if err != nil {
			return iterationCount, fmt.Errorf("failed to reload project after iteration %d: %w", i, err)
		}

		allComplete, passingCount, failingCount := config.CheckCompletion(project)
		logger.Infof("Status after iteration %d: %d passing, %d failing", i, passingCount, failingCount)

		// Stop if all requirements are passing
		if allComplete {
			logger.Success("All requirements passing! Stopping iteration loop.")
			break
		}

		// Continue to next iteration if not at max
		if i < maxIters {
			logger.Info("Requirements not complete, continuing to next iteration...")
		}
	}

	logger.Info("==========================================")
	logger.Successf("Iteration loop completed after %d iteration(s)", iterationCount)

	return iterationCount, nil
}

// CommitChanges stages all changes and commits them with an iteration-based message
func CommitChanges(ctx *context.Context, iteration int) error {
	if ctx.IsDryRun() {
		logger.Infof("[DRY-RUN] Would commit changes from iteration %d", iteration)
		return nil
	}

	// Stage all changes
	if err := git.StageAll(ctx); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are any staged changes to commit
	if !git.HasStagedChanges(ctx) {
		return fmt.Errorf("no changes to commit")
	}

	// Generate commit message based on iteration
	commitMsg := fmt.Sprintf("Development iteration %d", iteration)

	// Commit the staged changes
	if err := git.Commit(ctx, commitMsg); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if ctx.IsVerbose() {
		logger.Infof("Committed with message: %s", commitMsg)
	}

	return nil
}
