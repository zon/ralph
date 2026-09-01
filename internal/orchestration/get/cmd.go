package get

import (
	"errors"
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
// log, printing the result to out as a JSON array.
type Cmd struct {
	project ProjectClient
	out     io.Writer
}

func NewCmd(project ProjectClient, out io.Writer) *Cmd {
	return &Cmd{project: project, out: out}
}

// Flags carries the flags shared by both get subcommands.
type Flags struct {
	ProjectFile string
	Items       string
	Base        string
}

// Complete prints the hashes recorded complete in the branch commit log as a
// JSON array. When a project file is given, its resolved item array bounds the
// reported hashes; without one every trailer found in the log is reported
// without a range check.
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
	return output.PrintJSON(c.out, complete)
}

// Incomplete requires a project file and prints the items whose indices are
// not recorded complete, in array order. When indexOnly is true it prints the
// indices of those items instead of the items themselves.
func (c *Cmd) Incomplete(cfg *config.RalphConfig, flags Flags, indexOnly bool) error {
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
	if indexOnly {
		return output.PrintJSON(c.out, project.ItemIndices(incomplete))
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
