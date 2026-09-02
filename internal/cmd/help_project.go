package cmd

import (
	"os"

	"github.com/zon/ralph/internal/projectfile"
)

// HelpProjectCmd displays a guide to writing a project file.
type HelpProjectCmd struct{}

// Run prints the project file guide on stdout.
func (c *HelpProjectCmd) Run() error {
	return printDocumentation(os.Stdout, "project", projectfile.ProjectDocumentation())
}
