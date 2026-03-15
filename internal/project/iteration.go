package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/ai"
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
	repoRoot, err := git.FindRepoRoot()
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
	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations())

	var previousProject *config.Project
	iterationCount := 0

	for i := 1; i <= ctx.MaxIterations(); i++ {
		iterationCount = i

		logger.Verbose("")
		logger.Verbosef("=== Iteration %d/%d ===", i, ctx.MaxIterations())

		// Check for blocked.md from repo root
		if blocked, err := isBlocked(ctx); err != nil {
			return iterationCount, fmt.Errorf("failed to check for blocked.md: %w", err)
		} else if blocked {
			return iterationCount, ErrBlocked
		}

		// Load project state before this iteration to track which requirements were already passing
		if previousProject == nil {
			var err error
			previousProject, err = config.LoadProject(ctx.ProjectFile())
			if err != nil {
				return 0, fmt.Errorf("failed to load initial project state: %w", err)
			}
		}

		// Run single iteration: execute, commit, check completion
		if err := runSingleIteration(ctx, cleanupRegistrar, previousProject, i); err != nil {
			return iterationCount, err
		}

		// Update previous state for next iteration
		currentProject, err := config.LoadProject(ctx.ProjectFile())
		if err != nil {
			return iterationCount, fmt.Errorf("failed to load project after iteration %d: %w", i, err)
		}
		previousProject = currentProject

		// Check if all requirements are complete
		allComplete, _, _ := config.CheckCompletion(currentProject)
		if allComplete {
			logger.Success("All requirements complete")
			break
		}

		// Continue to next iteration if not at max
		if i < ctx.MaxIterations() {
			logger.Verbose("Requirements not complete, continuing to next iteration...")
		}
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterationCount)

	// Check if we reached max iterations without completing requirements
	currentProject, err := config.LoadProject(ctx.ProjectFile())
	if err != nil {
		return iterationCount, fmt.Errorf("failed to load project state: %w", err)
	}
	allComplete, _, failingCount := config.CheckCompletion(currentProject)
	if !allComplete {
		return iterationCount, fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
	}

	return iterationCount, nil
}

// runSingleIteration executes one iteration: runs requirement.Execute, commits changes, and reports completion
func runSingleIteration(ctx *context.Context, cleanupRegistrar func(func()), previousProject *config.Project, iteration int) error {
	// Run single development iteration
	logger.Verbose("Running development iteration...")
	if err := requirement.Execute(ctx, cleanupRegistrar); err != nil {
		if isFatalOpenCodeError(err) {
			return fmt.Errorf("%w: %v", ErrFatalOpenCodeError, err)
		}
		return fmt.Errorf("iteration %d failed: %w", iteration, err)
	}

	// Commit changes after iteration
	logger.Verbosef("Committing changes from iteration %d...", iteration)
	if err := CommitChanges(ctx, iteration); err != nil {
		if errors.Is(err, ErrNoChanges) {
			logger.Verbosef("No changes to commit after iteration %d", iteration)
		} else if errors.Is(err, ErrFatalPushError) {
			return err
		} else {
			return fmt.Errorf("iteration %d commit failed: %w", iteration, err)
		}
	} else {
		logger.Verbosef("Committed changes from iteration %d", iteration)
	}

	// Load and check project completion status
	currentProject, err := config.LoadProject(ctx.ProjectFile())
	if err != nil {
		return fmt.Errorf("failed to reload project after iteration %d: %w", iteration, err)
	}

	allComplete, passingCount, failingCount := config.CheckCompletion(currentProject)
	logger.Verbosef("Status after iteration %d: %d passing, %d failing", iteration, passingCount, failingCount)

	// Report newly passing requirements
	reportNewlyPassingRequirements(previousProject, currentProject)

	// Stop if all requirements are passing
	if allComplete {
		logger.Success("All requirements complete")
	}

	return nil
}

// reportNewlyPassingRequirements logs any requirements that just became passing
func reportNewlyPassingRequirements(previousProject, currentProject *config.Project) {
	for idx, req := range currentProject.Requirements {
		if req.Passing && !previousProject.Requirements[idx].Passing {
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
}

// CommitChanges stages all changes and commits them using report.md as the commit message
func CommitChanges(ctx *context.Context, iteration int) error {
	// If there are no uncommitted changes and no report.md, there is nothing to commit
	_, reportErr := os.Stat("report.md")
	if !git.HasUncommittedChanges() && os.IsNotExist(reportErr) {
		return ErrNoChanges
	}

	// If there are uncommitted changes but no report.md, prompt opencode to write one
	if err := generateChangelogIfNeeded(ctx); err != nil {
		logger.Warningf("Failed to generate changelog: %v", err)
	}

	// Read commit message from report.md
	commitMsg, err := getCommitMessage(iteration)
	if err != nil {
		return fmt.Errorf("cannot commit without a descriptive changelog: %w", err)
	}

	// Stage all changes
	if err := git.StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit staged changes
	if err := performCommit(ctx, commitMsg, iteration); err != nil {
		return err
	}

	// Pull and push
	if err := pullAndPush(ctx); err != nil {
		return err
	}

	return nil
}

// generateChangelogIfNeeded calls opencode to write report.md when the working tree
// has uncommitted changes but the agent did not produce a report.md itself.
func generateChangelogIfNeeded(ctx *context.Context) error {
	if !git.HasUncommittedChanges() {
		return nil
	}

	if _, err := os.Stat("report.md"); err == nil {
		// report.md already exists; nothing to do
		return nil
	}

	logger.Verbose("Uncommitted changes detected without report.md; generating changelog...")
	return ai.GenerateChangelog(ctx)
}

// getCommitMessage reads report.md and returns it as the commit message.
// Returns an error if report.md does not exist.
func getCommitMessage(iteration int) ([]byte, error) {
	reportPath := "report.md"
	commitMsg, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("report.md not found: %w", err)
	}
	if err := os.Remove(reportPath); err != nil {
		logger.Warningf("Failed to remove report.md: %v", err)
	}
	return commitMsg, nil
}

// performCommit commits the staged changes
func performCommit(ctx *context.Context, commitMsg []byte, iteration int) error {
	message := strings.TrimSpace(string(commitMsg))
	if message == "" {
		return fmt.Errorf("empty commit message: cannot proceed without a descriptive message")
	}

	if git.HasStagedChanges() {
		if err := git.Commit(message); err != nil {
			return fmt.Errorf("failed to commit: %w", err)
		}
	} else {
		logger.Verbosef("No file changes after iteration %d; skipping empty commit", iteration)
		if err := os.Remove("report.md"); err != nil && !os.IsNotExist(err) {
			logger.Warningf("Failed to remove report.md: %v", err)
		}
		return ErrNoChanges
	}

	if ctx.IsVerbose() {
		logger.Infof("Committed with message: %s", message)
	}
	return nil
}

// pullAndPush pulls remote changes and pushes the current branch
func pullAndPush(ctx *context.Context) error {
	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	logger.Verbose("Pulling remote changes before push...")
	if err := git.PullRebase(auth); err != nil {
		return fmt.Errorf("failed to pull before push: %w", err)
	}

	branch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	logger.Verbose("Pushing commit to origin...")
	if _, err := git.Push(auth, branch); err != nil {
		if errors.Is(err, git.ErrWorkflowPermission) {
			return fmt.Errorf("%w: %v", ErrFatalPushError, err)
		}
		return fmt.Errorf("failed to push commit: %w", err)
	}
	logger.Verbose("Pushed commit to origin")

	return nil
}
