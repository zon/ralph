package get

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// ProjectClient is the read-only surface of the project client the get command
// needs: validate the project file, resolve its item array, and read completion
// from the branch commit log. None of the methods write, commit, or switch
// branches.
type ProjectClient interface {
	Resolve(path string, query string) (*project.Project, error)
	Complete(proj *project.Project, base string) ([]int, error)
	Incomplete(proj *project.Project, base string) ([]project.Item, error)
	ValidateFile(path string) error
}

// Cmd orchestrates the get complete and get incomplete subcommands by
// resolving the item array and subtracting the indices recorded in the commit
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

// Complete prints the indices recorded complete in the branch commit log as a
// JSON array. When a project file is given, its resolved item array bounds the
// reported indices; without one every trailer found in the log is reported
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
	return printJSON(c.out, complete)
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
		indices := make([]int, len(incomplete))
		for i, it := range incomplete {
			indices[i] = it.Index
		}
		return printJSON(c.out, indices)
	}
	values := make([]any, len(incomplete))
	for i, it := range incomplete {
		values[i] = it.Value
	}
	return printJSON(c.out, values)
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

// printJSON writes v as a JSON array followed by a newline.
func printJSON(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = fmt.Fprintln(w)
	return err
}
