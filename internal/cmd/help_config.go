package cmd

import (
	"os"

	"github.com/zon/ralph/internal/config"
)

// HelpConfigCmd displays the .ralph/config.yaml reference documentation.
type HelpConfigCmd struct{}

// Run prints the configuration documentation on stdout.
func (c *HelpConfigCmd) Run() error {
	return printDocumentation(os.Stdout, "configuration", config.ConfigDocumentation())
}
