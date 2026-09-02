package get

import (
	"errors"
	"fmt"
	"io"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// ProjectClient is the read-only surface of the project client the get command
// needs: validate the project file, resolve its item array, and read completion
// from the branch commit log. None of the methods write, commit, or switch
// branches.
type ProjectClient interface {
	Resolve(path string, query string) (*project.Project, error)
	Complete(proj *project.Project, base string) ([]string, error)
	Incomplete(proj *project.Project, base string) ([]project.Item, error)
	ValidateFile(path string) error
}

// Cmd orchestrates the get complete and get incomplete subcommands by
// resolving the item array and subtracting the hashes recorded in the commit
// log, printing the result to out.
type Cmd struct {
	project ProjectClient
	out     io.Writer
}

func NewCmd(project ProjectClient, out io.Writer) *Cmd {
	return &Cmd{project: project, out: out}
}

// Flags carries the flags shared by both get subcommands plus the per-command
// output options exposed on the CLI. JSON, exposed only by complete, requests
// a JSON array of hashes instead of one hash per line.
type Flags struct {
	ProjectFile string
	Items       string
	Base        string
	JSON        bool
}

// Complete prints the hashes recorded complete in the branch commit log, one
// per line, or as a JSON array when flags.JSON is set. When a project file is
// given, its resolved item array bounds the reported hashes; without one every
// trailer found in the log is reported without a range check.
func (c *Cmd) Complete(cfg *config.RalphConfig, flags Flags) error {
	var proj *project.Project
	if flags.ProjectFile != "" {
		p, err := c.resolveProject(cfg, flags)
		if err != nil {
			return err
		}
		proj = p
	}
	complete, err := c.project.Complete(proj, resolveBase(flags.Base, cfg.DefaultBranch))
	if err != nil {
		return err
	}
	if flags.JSON {
		return output.PrintJSON(c.out, complete)
	}
	for _, hash := range complete {
		if _, err := fmt.Fprintln(c.out, hash); err != nil {
			return err
		}
	}
	return nil
}

// Incomplete requires a project file and prints the items whose indices are
// not recorded complete, in array order.
func (c *Cmd) Incomplete(cfg *config.RalphConfig, flags Flags) error {
	if flags.ProjectFile == "" {
		return errors.New("project file path is required")
	}
	proj, err := c.resolveProject(cfg, flags)
	if err != nil {
		return err
	}
	incomplete, err := c.project.Incomplete(proj, resolveBase(flags.Base, cfg.DefaultBranch))
	if err != nil {
		return err
	}
	return output.PrintJSON(c.out, project.ItemValues(incomplete))
}

// resolveProject validates the project file on disk and resolves its item
// array with the item query resolved as flag, then config, then ".".
func (c *Cmd) resolveProject(cfg *config.RalphConfig, flags Flags) (*project.Project, error) {
	if err := c.project.ValidateFile(flags.ProjectFile); err != nil {
		return nil, err
	}
	return c.project.Resolve(flags.ProjectFile, cfg.ResolveItems(flags.Items))
}

// resolveBase returns the base branch bounding the commit log: the --base flag
// when passed, otherwise the configured default branch.
func resolveBase(flag, defaultBranch string) string {
	if flag != "" {
		return flag
	}
	return defaultBranch
}
