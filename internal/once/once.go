package once

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
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
// 6. Sends desktop notifications on success/failure
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

	// Track started services for cleanup
	var processes []*services.Process

	// Start services if not disabled
	if ctx.ShouldStartServices() && len(ralphConfig.Services) > 0 {
		logger.Verbosef("Starting %d service(s)...", len(ralphConfig.Services))

		processes, err = services.StartAllServices(ralphConfig.Services, ctx.IsDryRun())
		if err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}

		// Register cleanup handler for services
		if cleanupRegistrar != nil {
			cleanupRegistrar(func() {
				services.StopAllServices(processes)
			})
		}

		logger.Verbose("All services started and healthy")
	} else if len(ralphConfig.Services) > 0 {
		logger.Verbose("Skipping service startup (--no-services flag)")
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
		// Send failure notification
		notify.Error(project.Name, ctx.ShouldNotify() && !ctx.IsDryRun())
		return fmt.Errorf("agent execution failed: %w", err)
	}
	logger.Verbose("AI agent execution completed")

	// Stage project file after agent completes
	logger.Verbose("Staging project file...")
	if err := git.StageFile(ctx, absProjectFile); err != nil {
		logger.Verbosef("Failed to stage project file: %v", err)
	} else {
		logger.Verbose("Project file staged")
	}

	// Send success notification
	notify.Success(project.Name, ctx.ShouldNotify() && !ctx.IsDryRun())

	logger.Verbose("Single iteration completed successfully")

	return nil
}
