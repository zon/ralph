package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	Steps       []string `yaml:"steps,omitempty"`
	Passing     bool     `yaml:"passing"`
}

// Service represents a service to be started/stopped
type Service struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
	Port    int      `yaml:"port,omitempty"` // Optional, for health checking
}

// RalphConfig represents the .ralph/config.yaml structure
type RalphConfig struct {
	MaxIterations int       `yaml:"maxIterations,omitempty"`
	BaseBranch    string    `yaml:"baseBranch,omitempty"`
	LLMProvider   string    `yaml:"llmProvider,omitempty"`
	LLMModel      string    `yaml:"llmModel,omitempty"`
	Services      []Service `yaml:"services,omitempty"`
}

// RalphSecrets represents the secrets.yaml structure
type RalphSecrets struct {
	APIKeys map[string]string `yaml:"apiKeys,omitempty"`
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

	// If config file doesn't exist, return defaults (not an error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &RalphConfig{
			MaxIterations: 10,
			BaseBranch:    "main",
			Services:      []Service{},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RalphConfig
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

	return &config, nil
}

// LoadRalphSecrets loads secrets from ~/.ralph/secrets.yaml or .ralph/secrets.yaml (cwd)
// Priority: .ralph/secrets.yaml (cwd) > ~/.ralph/secrets.yaml
// Returns empty secrets if neither file exists (not an error)
func LoadRalphSecrets() (*RalphSecrets, error) {
	var secretsPath string

	// Check for project-specific secrets first (highest priority)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	projectSecretsPath := filepath.Join(cwd, ".ralph", "secrets.yaml")
	if _, err := os.Stat(projectSecretsPath); err == nil {
		secretsPath = projectSecretsPath
	} else {
		// Fall back to global secrets
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		globalSecretsPath := filepath.Join(homeDir, ".ralph", "secrets.yaml")
		if _, err := os.Stat(globalSecretsPath); err == nil {
			secretsPath = globalSecretsPath
		}
	}

	// If no secrets file found, return empty secrets (not an error)
	if secretsPath == "" {
		return &RalphSecrets{
			APIKeys: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	var secrets RalphSecrets
	if err := yaml.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("failed to parse secrets YAML: %w", err)
	}

	if secrets.APIKeys == nil {
		secrets.APIKeys = make(map[string]string)
	}

	return &secrets, nil
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
