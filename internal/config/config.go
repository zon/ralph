package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed default-instructions.md
var defaultInstructions string

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

// Build represents a build command to run before starting services
type Build struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
}

// Service represents a service to be started/stopped
type Service struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"ctx,omitempty"`
	Port    int      `yaml:"port,omitempty"`    // Optional, for health checking
	Timeout int      `yaml:"timeout,omitempty"` // Optional, health check timeout in seconds (default: 30)
}

// ImageConfig represents container image configuration
type ImageConfig struct {
	Repository string `yaml:"repository,omitempty"`
	Tag        string `yaml:"tag,omitempty"`
}

// GitUserConfig represents git user configuration
type GitUserConfig struct {
	Name  string `yaml:"name,omitempty"`
	Email string `yaml:"email,omitempty"`
}

// WorkflowConfig represents Argo Workflow configuration options
type WorkflowConfig struct {
	Image      ImageConfig       `yaml:"image,omitempty"`
	ConfigMaps []string          `yaml:"configMaps,omitempty"`
	Secrets    []string          `yaml:"secrets,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	Context    string            `yaml:"context,omitempty"`
	Namespace  string            `yaml:"namespace,omitempty"`
	GitUser    GitUserConfig     `yaml:"gitUser,omitempty"`
}

// RalphConfig represents the .ralph/config.yaml structure
type RalphConfig struct {
	MaxIterations int            `yaml:"maxIterations,omitempty"`
	BaseBranch    string         `yaml:"baseBranch,omitempty"`
	Model         string         `yaml:"model,omitempty"` // AI model to use for coding and PR summary (default: anthropic/claude-sonnet-4-5)
	Builds        []Build        `yaml:"builds,omitempty"`
	Services      []Service      `yaml:"services,omitempty"`
	Workflow      WorkflowConfig `yaml:"workflow,omitempty"`
	Instructions  string         `yaml:"-"` // Not persisted in YAML, loaded from .ralph/instructions.md
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

// LoadConfig loads .ralph/config.yaml from the current working directory
// Returns default config if file doesn't exist (not an error)
func LoadConfig() (*RalphConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(cwd, ".ralph", "config.yaml")
	var config RalphConfig

	// If config file doesn't exist, use defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config = RalphConfig{
			MaxIterations: 10,
			BaseBranch:    "main",
			Model:         "anthropic/claude-sonnet-4-5",
			Services:      []Service{},
		}
	} else {
		// Load config file
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config YAML: %w", err)
		}

		// Apply defaults for missing values
		if config.MaxIterations == 0 {
			config.MaxIterations = 10
		}
		if config.BaseBranch == "" {
			config.BaseBranch = "main"
		}
		if config.Model == "" {
			config.Model = "anthropic/claude-sonnet-4-5"
		}
		// Apply default timeout for services
		for i := range config.Services {
			if config.Services[i].Timeout == 0 {
				config.Services[i].Timeout = 30
			}
		}
	}

	// Load instructions from .ralph/instructions.md or use default
	instructionsPath := filepath.Join(cwd, ".ralph", "instructions.md")
	if instructionsData, err := os.ReadFile(instructionsPath); err == nil {
		config.Instructions = string(instructionsData)
	} else {
		config.Instructions = defaultInstructions
	}

	return &config, nil
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
