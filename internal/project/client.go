package project

import (
	"fmt"
	"os"
	"path/filepath"
)

type Client struct{}

func (c *Client) Load(path string) (*Project, error) {
	return LoadProject(path)
}

func (c *Client) ValidateFile(path string) error {
	if path == "" {
		return fmt.Errorf("project file required (see --help)")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absPath)
	}
	return nil
}

func (c *Client) Reload(proj *Project) *Project {
	if proj.Path != "" {
		if latest, err := LoadProject(proj.Path); err == nil {
			return latest
		}
	}
	return proj
}

func (c *Client) AllRequirementsPassing(proj *Project) bool {
	allComplete, _, _ := CheckCompletion(proj)
	return allComplete
}

func (c *Client) MaxIterationsError(proj *Project) error {
	_, _, failingCount := CheckCompletion(proj)
	return fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
}
