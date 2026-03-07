package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/requirement"
)

// ErrFatalPushError is returned when a push fails with a permanent error that
// cannot be resolved by retrying (e.g. missing GitHub App permissions).
var ErrFatalPushError = errors.New("fatal push error")

// ErrNoChanges is returned by CommitChanges when there are no staged changes to commit
var ErrNoChanges = errors.New("no changes to commit")

// ErrMaxIterationsReached is returned when max iterations are reached but requirements are still failing
var ErrMaxIterationsReached = errors.New("max iteration limit reached")

// ErrBlocked is returned when blocked.md is detected in the repository
var ErrBlocked = errors.New("blocked.md detected")

// ErrFatalOpenCodeError is returned when opencode outputs a fatal error (e.g., account/billing issues)
var ErrFatalOpenCodeError = errors.New("fatal opencode error")

var fatalOpenCodePatterns = []string{
	"Insufficient Balance",
	"insufficient balance",
	"billing",
	"account",
	"payment required",
	"quota exceeded",
}

func isFatalOpenCodeError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	for _, pattern := range fatalOpenCodePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isBlocked checks if blocked.md exists in the repository root
func isBlocked(ctx *context.Context) (bool, error) {
	repoRoot, err := git.FindRepoRoot(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to find repo root: %w", err)
	}

	blockedPath := filepath.Join(repoRoot, "blocked.md")
	_, err = os.Stat(blockedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check for blocked.md: %w", err)
	}
	return true, nil
}

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

	var previousProject *config.Project
	iterationCount := 0

	for i := 1; i <= ctx.MaxIterations; i++ {
		iterationCount = i

		logger.Verbose("")
		logger.Verbosef("=== Iteration %d/%d ===", i, ctx.MaxIterations)

		// Check for blocked.md from repo root
		if blocked, err := isBlocked(ctx); err != nil {
			return iterationCount, fmt.Errorf("failed to check for blocked.md: %w", err)
		} else if blocked {
			return iterationCount, ErrBlocked
		}

		// Load project state before this iteration to track which requirements were already passing
		if previousProject == nil {
			var err error
			previousProject, err = config.LoadProject(ctx.ProjectFile)
			if err != nil {
				return 0, fmt.Errorf("failed to load initial project state: %w", err)
			}
		}

		// Run single development iteration
		logger.Verbose("Running development iteration...")
		if err := requirement.Execute(ctx, cleanupRegistrar); err != nil {
			if isFatalOpenCodeError(err) {
				return iterationCount, fmt.Errorf("%w: %v", ErrFatalOpenCodeError, err)
			}
			return iterationCount, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		// Commit changes after iteration
		logger.Verbosef("Committing changes from iteration %d...", i)
		if err := CommitChanges(ctx, i); err != nil {
			if errors.Is(err, ErrNoChanges) {
				logger.Verbosef("No changes to commit after iteration %d", i)
			} else if errors.Is(err, ErrFatalPushError) {
				return iterationCount, err
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

	// Check if we reached max iterations without completing requirements
	if !ctx.IsDryRun() {
		currentProject, err := config.LoadProject(ctx.ProjectFile)
		if err != nil {
			return iterationCount, fmt.Errorf("failed to load project state: %w", err)
		}
		allComplete, _, failingCount := config.CheckCompletion(currentProject)
		if !allComplete {
			return iterationCount, fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
		}
	}

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

	// Clean up and validate commit message
	message := strings.TrimSpace(string(commitMsg))
	if message == "" {
		message = fmt.Sprintf("Development iteration %d", iteration)
	}

	// Commit staged changes. If the AI ran but made no file changes (e.g. all
	// requirements were already passing), use --allow-empty so the branch still
	// gets a commit and gh pr create can succeed.
	if git.HasStagedChanges(ctx) {
		if err := git.Commit(ctx, message); err != nil {
			return fmt.Errorf("failed to commit: %w", err)
		}
	} else {
		logger.Verbosef("No file changes after iteration %d; creating empty commit", iteration)
		if err := git.CommitAllowEmpty(ctx, message); err != nil {
			return fmt.Errorf("failed to commit: %w", err)
		}
	}

	if ctx.IsVerbose() {
		logger.Infof("Committed with message: %s", message)
	}

	// Pull remote changes before pushing to handle resumed workflows where
	// the remote branch has advanced since we last pushed
	logger.Verbose("Pulling remote changes before push...")
	if err := git.PullRebase(ctx); err != nil {
		return fmt.Errorf("failed to pull before push: %w", err)
	}

	// Push commit to origin
	logger.Verbose("Pushing commit to origin...")
	if err := git.PushCurrentBranch(ctx); err != nil {
		if errors.Is(err, git.ErrWorkflowPermission) {
			return fmt.Errorf("%w: %v", ErrFatalPushError, err)
		}
		return fmt.Errorf("failed to push commit: %w", err)
	}
	logger.Verbose("Pushed commit to origin")

	return nil
}
