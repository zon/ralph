package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/services"
)

// Project represents a project YAML file with requirements
type Project struct {
	Slug         string        `yaml:"slug"`
	Title        string        `yaml:"title,omitempty"`
	Feature      string        `yaml:"feature,omitempty"`
	Requirements []Requirement `yaml:"requirements"`
	Path         string        `yaml:"-"`
}

// Requirement represents a single requirement in a project
type Requirement struct {
	Slug        string      `yaml:"slug"`
	Description string      `yaml:"description,omitempty"`
	Items       []string    `yaml:"items,omitempty"`
	Scenarios   []Scenario  `yaml:"scenarios,omitempty"`
	Code        []CodeEntry `yaml:"code,omitempty"`
	Tests       []CodeEntry `yaml:"tests,omitempty"`
	Passing     bool        `yaml:"passing"`
}

// Scenario is a GWT scenario copied from the spec document.
type Scenario struct {
	Title string   `yaml:"title"`
	Items []string `yaml:"items"`
}

// CodeEntry describes a function or test the project should implement.
// Used for both `code` (production code from orchestration.md) and `tests` (specific
// tests the project must write).
type CodeEntry struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Module      string `yaml:"module"`
	Body        string `yaml:"body"`
}

// LoadProject loads and validates a project YAML file
func LoadProject(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	var proj Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		return nil, fmt.Errorf("failed to parse project YAML: %w", err)
	}

	if err := ValidateProject(&proj); err != nil {
		return nil, err
	}

	proj.Path = path
	return &proj, nil
}

// ValidateProject validates a project structure
func ValidateProject(p *Project) error {
	if p.Slug == "" {
		return fmt.Errorf("project slug is required")
	}

	if len(p.Requirements) == 0 {
		return fmt.Errorf("project must have at least one requirement")
	}

	seen := make(map[string]struct{}, len(p.Requirements))
	for i, req := range p.Requirements {
		if req.Slug == "" {
			return fmt.Errorf("requirement[%d] slug is required", i)
		}
		if _, dup := seen[req.Slug]; dup {
			return fmt.Errorf("requirement slug %q is not unique", req.Slug)
		}
		seen[req.Slug] = struct{}{}

		if len(req.Items) == 0 && len(req.Scenarios) == 0 && len(req.Code) == 0 && len(req.Tests) == 0 {
			return fmt.Errorf("requirement %q must define at least one of items, scenarios, code, or tests", req.Slug)
		}

		for j, c := range req.Code {
			if err := validateCodeEntry(req.Slug, "code", j, c); err != nil {
				return err
			}
		}
		for j, c := range req.Tests {
			if err := validateCodeEntry(req.Slug, "tests", j, c); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateCodeEntry(reqSlug, field string, idx int, c CodeEntry) error {
	missing := []string{}
	if c.Name == "" {
		missing = append(missing, "name")
	}
	if c.Description == "" {
		missing = append(missing, "description")
	}
	if c.Module == "" {
		missing = append(missing, "module")
	}
	if c.Body == "" {
		missing = append(missing, "body")
	}
	if len(missing) > 0 {
		return fmt.Errorf("requirement %q %s[%d] is missing required field(s): %s", reqSlug, field, idx, strings.Join(missing, ", "))
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

// UpdateRequirementStatus updates the passing status of the requirement
// identified by its slug.
func UpdateRequirementStatus(p *Project, reqSlug string, passing bool) error {
	for i := range p.Requirements {
		if p.Requirements[i].Slug == reqSlug {
			p.Requirements[i].Passing = passing
			return nil
		}
	}
	return fmt.Errorf("requirement not found: %s", reqSlug)
}

type IterationSetup struct {
	Project       *Project
	Config        *config.RalphConfig
	PickedReqPath string
	CommitLog     string
	ServiceMgr    *services.Manager
}

func PrepareIteration(ctx *context.Context, cleanupRegistrar func(func())) (*IterationSetup, error) {
	if ctx.IsVerbose() {
		logger.SetVerbose(true)
	}

	absProjectFile, err := filepath.Abs(ctx.ProjectFile())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project file path: %w", err)
	}

	if _, err := os.Stat(absProjectFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("project file not found: %s", absProjectFile)
	}

	if err := checkBlockedFile(absProjectFile); err != nil {
		return nil, err
	}

	proj, err := LoadProject(absProjectFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}
	if proj.Title != "" && ctx.IsVerbose() {
		logger.Verbosef("Title: %s", proj.Title)
	}

	allComplete, passingCount, failingCount := CheckCompletion(proj)
	logger.Verbosef("Requirements: %d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	svcMgr, err := handleServiceStartup(ctx, cleanupRegistrar, ralphConfig)
	if err != nil {
		return nil, err
	}

	commitLog, err := getCommitLogForPrompt(ctx, ralphConfig.DefaultBranch)
	if err != nil {
		logger.Verbosef("Failed to get commit log: %v", err)
		commitLog = ""
	}

	return &IterationSetup{
		Project:       proj,
		Config:        ralphConfig,
		PickedReqPath: filepath.Join(filepath.Dir(absProjectFile), "picked-requirement.yaml"),
		CommitLog:     commitLog,
		ServiceMgr:    svcMgr,
	}, nil
}

func ExecuteDevelopmentIteration(ctx *context.Context, cleanupRegistrar func(func()) /* DEPRECATED: Use ExecuteDevelopmentIterationWithSetup */) error {
	setup, err := PrepareIteration(ctx, cleanupRegistrar)
	if err != nil {
		return err
	}
	defer func() {
		if setup.ServiceMgr != nil {
			setup.ServiceMgr.Stop()
		}
	}()
	return ExecuteDevelopmentIterationWithSetup(ctx, setup)
}

func ExecuteDevelopmentIterationWithSetup(ctx *context.Context, setup *IterationSetup) error {
	logger.Verbosef("Loading project file: %s", setup.Project.Path)

	projectContent, err := marshalProjectToString(setup.Project)
	if err != nil {
		return fmt.Errorf("failed to serialize project: %w", err)
	}

	pickPromptData := ai.PickPromptData{
		Notes:          ctx.Notes(),
		CommitLog:      setup.CommitLog,
		ProjectContent: projectContent,
		PickedReqPath:  setup.PickedReqPath,
	}

	pickPrompt, err := ai.BuildPickPrompt(pickPromptData)
	if err != nil {
		return fmt.Errorf("failed to build pick prompt: %w", err)
	}
	logger.Verbose("Pick prompt generated")

	logger.Verbose("Running picker agent...")
	if err := ai.RunAgent(ctx, pickPrompt); err != nil {
		if writeBlockedMD(setup.Project.Path, err) == nil {
			logger.Verbosef("Wrote blocked.md due to picker agent failure")
			return fmt.Errorf("picker agent execution failed: %w", err)
		}
		logger.Verbosef("Failed to write blocked.md: %v", err)
		return fmt.Errorf("picker agent execution failed: %w", err)
	}
	logger.Verbose("Picker agent execution completed")

	pickedReqData, err := os.ReadFile(setup.PickedReqPath)
	if err != nil {
		return fmt.Errorf("failed to read picked requirement: %w", err)
	}
	selectedRequirement := string(pickedReqData)

	if err := os.Remove(setup.PickedReqPath); err != nil {
		logger.Verbosef("Failed to remove picked-requirement.yaml: %v", err)
	} else {
		logger.Verbose("Cleaned up picked-requirement.yaml")
	}

	logger.Verbose("Generating development prompt...")

	devPromptData := ai.DevelopPromptData{
		Notes:               ctx.Notes(),
		CommitLog:           setup.CommitLog,
		ProjectContent:      projectContent,
		SelectedRequirement: selectedRequirement,
		ProjectFilePath:     setup.Project.Path,
		Services:            setup.Config.Services,
		Instructions:        setup.Config.Instructions,
	}

	devPrompt, err := ai.BuildDevelopPrompt(devPromptData)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}
	logger.Verbose("Development prompt generated")

	logger.Verbose("Running AI agent...")
	if err := ai.RunAgent(ctx, devPrompt); err != nil {
		if writeBlockedMD(setup.Project.Path, err) == nil {
			logger.Verbosef("Wrote blocked.md due to agent failure")
		} else {
			logger.Verbosef("Failed to write blocked.md: %v", err)
		}
		return fmt.Errorf("agent execution failed: %w", err)
	}
	logger.Verbose("AI agent execution completed")

	if err := performPostAgentCleanup(ctx, setup.Project.Path, setup.Config.Services); err != nil {
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

			fixPrompt, buildErr := ai.BuildFixServicePrompt(ctx, failedSvc, err)
			if buildErr != nil {
				return nil, fmt.Errorf("failed to build fix service prompt: %w", buildErr)
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

func FindCompleteProjects(dir string) ([]string, error) {
	var completeProjects []string

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	var allFiles []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	for _, filePath := range allFiles {
		proj, err := LoadProject(filePath)
		if err != nil {
			continue
		}

		if IsProjectComplete(proj) {
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				continue
			}
			completeProjects = append(completeProjects, absPath)
		}
	}

	return completeProjects, nil
}

func IsProjectComplete(proj *Project) bool {
	if len(proj.Requirements) == 0 {
		return false
	}

	for _, req := range proj.Requirements {
		if !req.Passing {
			return false
		}
	}

	return true
}

func RemoveAndCommit(ctx *context.Context, files []string) error {
	if len(files) == 0 {
		return nil
	}

	for _, filePath := range files {
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove project file %s: %w", filePath, err)
		}
		logger.Infof("Removed complete project file: %s", filePath)
	}

	for _, filePath := range files {
		if err := git.StageFile(filePath); err != nil {
			return fmt.Errorf("failed to stage deleted file %s: %w", filePath, err)
		}
	}

	commitMessage := "chore: remove complete project files"
	if err := git.Commit(commitMessage); err != nil {
		return fmt.Errorf("failed to commit deleted files: %w", err)
	}

	logger.Successf("Committed removal of %d complete project file(s)", len(files))
	return nil
}
