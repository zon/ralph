package cmd

import (
	"errors"
	"fmt"
)

// LoopCmd is the `ralph loop` command. It runs AI iterations over a set of
// steps until an iteration reports nothing to do or the loop reaches its
// limit.
type LoopCmd struct {
	Slug  string   `arg:"" optional:"" help:"Slug of the loop configuration in .ralph/config.yaml"`
	Steps []string `help:"Step to run in the loop (repeatable)" name:"step"`
	Max   int      `help:"Maximum number of iterations" name:"max" default:"10"`
}

// Validate checks the parsed command line before the command runs. Kong wraps
// its error in a usage error, printing the full usage alongside it.
func (c *LoopCmd) Validate() error {
	if c.Slug == "" && len(c.Steps) == 0 {
		return errors.New("a slug or at least one --step is required")
	}
	if c.Max < 1 {
		return fmt.Errorf("--max must be positive, got %d", c.Max)
	}
	return nil
}

// Run is a stub. It does nothing yet.
func (c *LoopCmd) Run() error {
	return nil
}
