package cmd

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
)

type ValidateCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file"`
}

func (v *ValidateCmd) Run() error {
	projectData, err := config.LoadProject(v.ProjectFile)
	if err != nil {
		return err
	}

	fmt.Printf("Project '%s' is valid (%d requirements)\n", projectData.Name, len(projectData.Requirements))
	return nil
}
