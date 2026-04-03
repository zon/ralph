package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/prompt"
	"github.com/zon/ralph/internal/services"
)

// Project represents a project YAML file with requirements
type Project struct {
	Name         string        `yaml:"name"`
	Description  string        `yaml:"description,omitempty"`
	Requirements []Requirement `yaml:"requirements"`
}

// Requirement represents a single requirement in a project
type Requirement struct {
	ID          string   `yaml:"id,omitempty"`
	Category    string   `yaml:"category,omitempty"`
	Name        string   `yaml:"name,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Items       []string `yaml:"items,omitempty"`
	Passing     bool     `yaml:"passing"`
}

// LoadProject loads and validates a project YAML file
func LoadProject(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	var project Project
	if err := yaml.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project YAML: %w", err)
	}

	if err := ValidateProject(&project); err != nil {
		return nil, err
	}

	return &project, nil
}

// ValidateProject validates a project structure
func ValidateProject(p *Project) error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if len(p.Requirements) == 0 {
		return fmt.Errorf("project must have at least one requirement")
	}

	return nil
}

// SaveProject saves a project to a YAML file
func SaveProject(path string, p *Project) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write project file: %w", err)
	}

	return nil
}

// CheckCompletion checks project completion status
// Returns: allComplete, passingCount, failingCount
func CheckCompletion(p *Project) (bool, int, int) {
	passingCount := 0
	failingCount := 0

	for _, req := range p.Requirements {
		if req.Passing {
			passingCount++
		} else {
			failingCount++
		}
	}

	allComplete := failingCount == 0

	return allComplete, passingCount, failingCount
}

// UpdateRequirementStatus updates the passing status of a specific requirement
func UpdateRequirementStatus(p *Project, reqID string, passing bool) error {
	found := false

	for i := range p.Requirements {
		// Match by ID if provided, otherwise match by index
		if p.Requirements[i].ID == reqID {
			p.Requirements[i].Passing = passing
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("requirement not found: %s", reqID)
	}

	return nil
}

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
	proj, err := LoadProject(absProjectFile)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}
	if proj.Description != "" && ctx.IsVerbose() {
		logger.Verbosef("Description: %s", proj.Description)
	}

	// Show project status
	allComplete, passingCount, failingCount := CheckCompletion(proj)
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

	projectContent, err := marshalProjectToString(proj)
	if err != nil {
		return fmt.Errorf("failed to serialize project: %w", err)
	}

	commitLog, err := getCommitLogForPrompt(ctx, ralphConfig.DefaultBranch)
	if err != nil {
		logger.Verbosef("Failed to get commit log: %v", err)
		commitLog = ""
	}

	pickPromptData := prompt.PickPromptData{
		Notes:          ctx.Notes(),
		CommitLog:      commitLog,
		ProjectContent: projectContent,
		PickedReqPath:  pickedReqPath,
	}

	pickPrompt, err := prompt.BuildPickPrompt(pickPromptData)
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

	devPromptData := prompt.DevelopPromptData{
		Notes:               ctx.Notes(),
		CommitLog:           commitLog,
		ProjectContent:      projectContent,
		SelectedRequirement: selectedRequirement,
		ProjectFilePath:     absProjectFile,
		Services:            ralphConfig.Services,
		Instructions:        ralphConfig.Instructions,
	}

	devPrompt, err := prompt.BuildDevelopPrompt(devPromptData)
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

			fixPrompt, promptErr := prompt.BuildFixServicePrompt(ctx, failedSvc, err)
			if promptErr != nil {
				return nil, fmt.Errorf("failed to build fix service prompt: %w", promptErr)
			}

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

func marshalProjectToString(proj *Project) (string, error) {
	data, err := yaml.Marshal(proj)
	if err != nil {
		return "", fmt.Errorf("failed to marshal project: %w", err)
	}
	return string(data), nil
}

func getCommitLogForPrompt(ctx *context.Context, defaultBranch string) (string, error) {
	baseBranch := defaultBranch
	if ctx.BaseBranch() != "" {
		baseBranch = ctx.BaseBranch()
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	if currentBranch == baseBranch {
		return "", nil
	}

	return git.GetCommitLog(baseBranch, 10)
}
