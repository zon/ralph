package requirement

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/prompt"
	"github.com/zon/ralph/internal/services"
)

// Execute runs a single development iteration
// It performs the following steps:
// 1. Validates and loads the project file
// 2. Starts configured services (unless disabled)
// 3. Generates a development prompt with context
// 4. Runs the AI agent with the prompt
// 5. Stages the project file after completion
// Note: Build commands should be run once at the project level, not per iteration
func Execute(ctx *context.Context, cleanupRegistrar func(func())) error {
	// Enable verbose logging if requested
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	if ctx.IsDryRun() {
		logger.Verbose("=== DRY-RUN MODE: No changes will be made ===")
	}

	// Validate project file exists
	absProjectFile, err := filepath.Abs(ctx.ProjectFile)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absProjectFile)
	}

	// Check if blocked.md exists from a previous blocked run
	blockedPath := filepath.Join(filepath.Dir(absProjectFile), "blocked.md")
	if _, err := os.Stat(blockedPath); err == nil {
		blockedContent, readErr := os.ReadFile(blockedPath)
		if readErr != nil {
			return fmt.Errorf("agent is blocked (blocked.md exists but could not read): %w", readErr)
		}
		return fmt.Errorf("agent is blocked:\n%s", string(blockedContent))
	}

	logger.Verbosef("Loading project file: %s", absProjectFile)

	// Load and validate project
	project, err := config.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}
	if project.Description != "" && ctx.IsVerbose() {
		logger.Verbosef("Description: %s", project.Description)
	}

	// Show project status
	allComplete, passingCount, failingCount := config.CheckCompletion(project)
	logger.Verbosef("Requirements: %d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create service manager for this requirement run
	svcMgr := services.NewManager()

	// Start services if not disabled
	if ctx.ShouldStartServices() && len(ralphConfig.Services) > 0 {
		if failedSvc, err := svcMgr.Start(ralphConfig.Services, ctx.IsDryRun()); err != nil {
			logger.Warningf("Service startup failed: %v", err)

			// Build a prompt focused only on fixing the failed service
			fixPrompt := prompt.BuildServiceFixPrompt(ctx, failedSvc, err)

			if agentErr := ai.RunAgent(ctx, fixPrompt); agentErr != nil {
				return fmt.Errorf("agent execution failed while fixing service: %w", agentErr)
			}
			return nil
		} else {
			// Services started successfully
			// Register cleanup handler for signal interrupts (SIGINT/SIGTERM)
			if cleanupRegistrar != nil {
				cleanupRegistrar(func() {
					svcMgr.Stop()
				})
			}

			// Ensure services are stopped when this function exits (success or error)
			defer svcMgr.Stop()
		}
	}

	// Generate development prompt
	logger.Verbose("Generating development prompt...")
	devPrompt, err := prompt.BuildDevelopPrompt(ctx, absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}
	logger.Verbose("Development prompt generated")

	// Run AI agent with prompt
	logger.Verbose("Running AI agent...")
	if err := ai.RunAgent(ctx, devPrompt); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}
	logger.Verbose("AI agent execution completed")

	// Remove service log files now that the iteration is complete
	for _, svc := range ralphConfig.Services {
		logPath := services.LogFileName(svc.Name)
		if err := os.Remove(logPath); err == nil {
			logger.Verbosef("Removed service log: %s", logPath)
		}
	}

	// Normalize project file: strip excess trailing newlines added by the agent
	if data, err := os.ReadFile(absProjectFile); err == nil {
		normalized := []byte(strings.TrimRight(string(data), "\n") + "\n")
		if len(normalized) != len(data) {
			if writeErr := os.WriteFile(absProjectFile, normalized, 0644); writeErr != nil {
				logger.Verbosef("Failed to normalize project file: %v", writeErr)
			}
		}
	}

	// Stage project file after agent completes, only if it has changes
	if git.HasFileChanges(ctx, absProjectFile) {
		logger.Verbose("Staging project file...")
		if err := git.StageFile(ctx, absProjectFile); err != nil {
			logger.Verbosef("Failed to stage project file: %v", err)
		} else {
			logger.Verbose("Project file staged")
		}
	}

	logger.Verbose("Single iteration completed successfully")

	return nil
}
