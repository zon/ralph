package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

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

// iterateWhile loops while worker returns continue=true, up to max iterations.
// worker receives iteration number and returns (continue, error).
func iterateWhile(max int, worker func(iteration int) (bool, error)) (int, error) {
	for i := 1; i <= max; i++ {
		continueLoop, err := worker(i)
		if err != nil {
			return i, err
		}
		if !continueLoop {
			return i, nil
		}
	}
	return max, nil
}

// RunIterationLoop runs multiple development iterations until completion or max iterations
// Each iteration:
// 1. Runs a single development iteration (requirement.Execute)
// 2. Commits the changes
// 3. Checks project completion status
// 4. Stops when all requirements pass OR max iterations reached
//
// Returns the final iteration count and any error encountered
func RunIterationLoop(ctx *context.Context, cleanupRegistrar func(func()), proj *project.Project) (int, error) {
	logger.Verbosef("Starting iteration loop (max: %d)", ctx.MaxIterations())

	var previousProject *project.Project

	worker := func(iteration int) (bool, error) {
		logger.Verbose("")
		logger.Verbosef("=== Iteration %d/%d ===", iteration, ctx.MaxIterations())

		if blocked, err := isBlocked(ctx); err != nil {
			return false, fmt.Errorf("failed to check for blocked.md: %w", err)
		} else if blocked {
			return false, ErrBlocked
		}

		if previousProject == nil {
			previousProject = proj
		}

		if err := runSingleIteration(ctx, cleanupRegistrar, previousProject, iteration); err != nil {
			return false, err
		}

		currentProject, err := project.LoadProject(proj.Path)
		if err != nil {
			return false, fmt.Errorf("failed to load project after iteration %d: %w", iteration, err)
		}
		previousProject = currentProject

		allComplete, _, _ := project.CheckCompletion(currentProject)
		if allComplete {
			logger.Success("All requirements complete")
			return false, nil
		}

		if iteration < ctx.MaxIterations() {
			logger.Verbose("Requirements not complete, continuing to next iteration...")
		}
		return true, nil
	}

	iterationCount, err := iterateWhile(ctx.MaxIterations(), worker)
	if err != nil {
		return iterationCount, err
	}

	logger.Verbosef("Iteration loop completed after %d iteration(s)", iterationCount)

	currentProject, err := project.LoadProject(proj.Path)
	if err != nil {
		return iterationCount, fmt.Errorf("failed to load project state: %w", err)
	}
	allComplete, _, failingCount := project.CheckCompletion(currentProject)
	if !allComplete {
		return iterationCount, fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
	}

	return iterationCount, nil
}

// runSingleIteration executes one iteration: runs requirement.Execute, commits changes, and reports completion
func runSingleIteration(ctx *context.Context, cleanupRegistrar func(func()), previousProject *project.Project, iteration int) error {
	// Run single development iteration
	logger.Verbose("Running development iteration...")
	if err := project.ExecuteDevelopmentIteration(ctx, cleanupRegistrar); err != nil {
		if isFatalOpenCodeError(err) {
			return fmt.Errorf("%w: %v", ErrFatalOpenCodeError, err)
		}
		return fmt.Errorf("iteration %d failed: %w", iteration, err)
	}

	// Commit changes after iteration
	if err := commitIterationChanges(ctx, iteration); err != nil {
		return err
	}

	// Load and check project completion status
	currentProject, err := project.LoadProject(ctx.ProjectFile())
	if err != nil {
		return fmt.Errorf("failed to reload project after iteration %d: %w", iteration, err)
	}

	allComplete, passingCount, failingCount := project.CheckCompletion(currentProject)
	logger.Verbosef("Status after iteration %d: %d passing, %d failing", iteration, passingCount, failingCount)

	// Report newly passing requirements
	reportNewlyPassingRequirements(previousProject, currentProject)

	// Stop if all requirements are passing
	if allComplete {
		logger.Success("All requirements complete")
	}

	return nil
}

// commitIterationChanges handles the commit workflow for an iteration.
// It detects report.md, generates changelog if needed, reads the commit message,
// and commits changes via git. Returns nil if there are no changes to commit.
func commitIterationChanges(ctx *context.Context, iteration int) error {
	_, reportErr := os.Stat("report.md")
	if !git.HasUncommittedChanges() && os.IsNotExist(reportErr) {
		logger.Verbosef("No changes to commit")
		return nil
	}

	if err := generateChangelogIfNeeded(ctx); err != nil {
		logger.Warningf("Failed to generate changelog: %v", err)
	}

	commitMsg, err := getCommitMessage(iteration)
	if err != nil {
		return fmt.Errorf("cannot commit without a descriptive changelog: %w", err)
	}

	logger.Verbosef("Committing changes from iteration %d...", iteration)
	isWorkflow := ctx.IsWorkflowExecution()
	var owner, repo string
	if isWorkflow {
		owner, repo = ctx.RepoOwnerAndName()
	}

	message := strings.TrimSpace(string(commitMsg))
	if err := git.CommitChanges(isWorkflow, owner, repo, message); err != nil {
		if errors.Is(err, git.ErrNoChanges) {
			logger.Verbosef("No changes to commit after iteration %d", iteration)
		} else if errors.Is(err, git.ErrFatalPushError) {
			return err
		} else {
			return fmt.Errorf("iteration %d commit failed: %w", iteration, err)
		}
	} else {
		logger.Verbosef("Committed changes from iteration %d", iteration)
	}

	return nil
}

// reportNewlyPassingRequirements logs any requirements that just became passing
func reportNewlyPassingRequirements(previousProject, currentProject *project.Project) {
	for idx, req := range currentProject.Requirements {
		if req.Passing && !previousProject.Requirements[idx].Passing {
			description := req.Description
			if description == "" {
				description = req.Slug
			}
			logger.Successf("Requirement complete: %s", description)
		}
	}
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
