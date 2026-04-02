package run

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
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/prompt"
	"github.com/zon/ralph/internal/services"
)

// ExecuteDevelopmentIteration runs a single development iteration
// It performs the following steps:
// 1. Validates and loads the project file
// 2. Starts configured services (unless disabled)
// 3. Generates a development prompt with context
// 4. Runs the AI agent with the prompt
// 5. Stages the project file after completion
// Note: Build commands should be run once at the project level, not per iteration
func ExecuteDevelopmentIteration(ctx *context.Context, cleanupRegistrar func(func())) error {
	// Enable verbose logging if requested
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	// Validate project file exists
	absProjectFile, err := filepath.Abs(ctx.ProjectFile())
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}

	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absProjectFile)
	}

	// Check if blocked.md exists from a previous blocked run
	if err := checkBlockedFile(absProjectFile); err != nil {
		return err
	}

	logger.Verbosef("Loading project file: %s", absProjectFile)

	// Load and validate project
	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}
	if proj.Description != "" && ctx.IsVerbose() {
		logger.Verbosef("Description: %s", proj.Description)
	}

	// Show project status
	allComplete, passingCount, failingCount := project.CheckCompletion(proj)
	logger.Verbosef("Requirements: %d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	// Load configuration
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle service startup and failure recovery
	svcMgr, err := handleServiceStartup(ctx, cleanupRegistrar, ralphConfig)
	if err != nil {
		return err
	}
	if svcMgr != nil {
		defer svcMgr.Stop()
	}

	// Generate pick prompt and run picker agent
	pickedReqPath := filepath.Join(filepath.Dir(absProjectFile), "picked-requirement.yaml")
	logger.Verbose("Generating pick prompt...")
	pickPrompt, err := prompt.BuildPickPrompt(ctx, absProjectFile, pickedReqPath)
	if err != nil {
		return fmt.Errorf("failed to build pick prompt: %w", err)
	}
	logger.Verbose("Pick prompt generated")

	logger.Verbose("Running picker agent...")
	if err := ai.RunAgent(ctx, pickPrompt); err != nil {
		if writeBlockedMD(absProjectFile, err) == nil {
			logger.Verbosef("Wrote blocked.md due to picker agent failure")
			return fmt.Errorf("picker agent execution failed: %w", err)
		}
		logger.Verbosef("Failed to write blocked.md: %v", err)
		return fmt.Errorf("picker agent execution failed: %w", err)
	}
	logger.Verbose("Picker agent execution completed")

	// Read the selected requirement from picked-requirement.yaml
	pickedReqData, err := os.ReadFile(pickedReqPath)
	if err != nil {
		return fmt.Errorf("failed to read picked requirement: %w", err)
	}
	selectedRequirement := string(pickedReqData)

	// Clean up picked-requirement.yaml
	if err := os.Remove(pickedReqPath); err != nil {
		logger.Verbosef("Failed to remove picked-requirement.yaml: %v", err)
	} else {
		logger.Verbose("Cleaned up picked-requirement.yaml")
	}

	// Generate development prompt with selected requirement
	logger.Verbose("Generating development prompt...")
	devPrompt, err := prompt.BuildDevelopPrompt(ctx, absProjectFile, selectedRequirement)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}
	logger.Verbose("Development prompt generated")

	// Run AI agent with prompt
	logger.Verbose("Running AI agent...")
	if err := ai.RunAgent(ctx, devPrompt); err != nil {
		if writeBlockedMD(absProjectFile, err) == nil {
			logger.Verbosef("Wrote blocked.md due to agent failure")
		} else {
			logger.Verbosef("Failed to write blocked.md: %v", err)
		}
		return fmt.Errorf("agent execution failed: %w", err)
	}
	logger.Verbose("AI agent execution completed")

	// Post-agent cleanup
	if err := performPostAgentCleanup(ctx, absProjectFile, ralphConfig.Services); err != nil {
		return err
	}

	logger.Verbose("Single iteration completed successfully")

	return nil
}

// checkBlockedFile checks if blocked.md exists and returns an error if it does
func checkBlockedFile(absProjectFile string) error {
	blockedPath := filepath.Join(filepath.Dir(absProjectFile), "blocked.md")
	if _, err := os.Stat(blockedPath); err == nil {
		blockedContent, readErr := os.ReadFile(blockedPath)
		if readErr != nil {
			return fmt.Errorf("agent is blocked (blocked.md exists but could not read): %w", readErr)
		}
		return fmt.Errorf("agent is blocked:\n%s", string(blockedContent))
	}
	return nil
}

func writeBlockedMD(absProjectFile string, err error) error {
	blockedPath := filepath.Join(filepath.Dir(absProjectFile), "blocked.md")
	content := fmt.Sprintf("# Blocked\n\nError: %s\n", err.Error())
	return os.WriteFile(blockedPath, []byte(content), 0644)
}

// handleServiceStartup starts services if not disabled, and handles failure recovery.
// Returns the service manager if services were started successfully (caller must stop it),
// or nil if services were not started or a failure was handled.
func handleServiceStartup(ctx *context.Context, cleanupRegistrar func(func()), ralphConfig *config.RalphConfig) (*services.Manager, error) {
	svcMgr := services.NewManager()

	// Start services if not disabled
	if !ctx.NoServices() && len(ralphConfig.Services) > 0 {
		if failedSvc, err := svcMgr.Start(ralphConfig.Services); err != nil {
			logger.Warningf("Service startup failed: %v", err)

			// Build a prompt focused only on fixing the failed service
			fixPrompt := prompt.BuildServiceFixPrompt(ctx, failedSvc, err)

			if agentErr := ai.RunAgent(ctx, fixPrompt); agentErr != nil {
				return nil, fmt.Errorf("agent execution failed while fixing service: %w", agentErr)
			}
			return nil, nil
		}

		// Services started successfully
		// Register cleanup handler for signal interrupts (SIGINT/SIGTERM)
		if cleanupRegistrar != nil {
			cleanupRegistrar(func() {
				svcMgr.Stop()
			})
		}
		return svcMgr, nil
	}
	return nil, nil
}

// performPostAgentCleanup removes service logs, normalizes project file, and stages it
func performPostAgentCleanup(ctx *context.Context, absProjectFile string, servicesList []config.Service) error {
	// Remove service log files now that the iteration is complete
	for _, svc := range servicesList {
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
	if git.HasFileChanges(absProjectFile) {
		logger.Verbose("Staging project file...")
		if err := git.StageFile(absProjectFile); err != nil {
			logger.Verbosef("Failed to stage project file: %v", err)
		} else {
			logger.Verbose("Project file staged")
		}
	}
	return nil
}
