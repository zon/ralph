package project

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/fileutil"
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

// readProjectFile reads the project file at the given path.
func readProjectFile(path string) ([]byte, error) {
	return fileutil.ReadFile(path)
}

// parseProjectYAML unmarshals YAML data into a Project.
func parseProjectYAML(data []byte) (*Project, error) {
	var project Project
	if err := yaml.Unmarshal(data, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// writeProjectFile writes the project data to the given path.
func writeProjectFile(path string, data []byte) error {
	return fileutil.WriteFile(path, data, 0644)
}

// LoadProject loads and validates a project YAML file
func LoadProject(path string) (*Project, error) {
	data, err := readProjectFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	project, err := parseProjectYAML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project YAML: %w", err)
	}

	if err := ValidateProject(project); err != nil {
		return nil, err
	}

	return project, nil
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

	if err := writeProjectFile(path, data); err != nil {
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
