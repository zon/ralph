package cmd

import (
	orchestrationMerge "github.com/zon/ralph/internal/orchestration/merge"
)

// MergeCmd is the command for merging a completed PR
type MergeCmd struct {
	Branch  string `arg:"" help:"PR branch name to merge"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	Local   bool   `help:"Run merge locally instead of submitting an Argo workflow" default:"false"`
	PR      string `help:"Pull request number" required:""`
	Repo    string `help:"GitHub repository (owner/repo); defaults to repo detected from git remote" default:""`

	cleanupRegistrar func(func()) `kong:"-"`
}

// Run executes the merge command (implements kong.Run interface)
func (m *MergeCmd) Run() error {
	flags := orchestrationMerge.MergeFlags{
		Branch:  m.Branch,
		PR:      m.PR,
		Repo:    m.Repo,
		Local:   m.Local,
		Verbose: m.Verbose,
	}

	cmd := newOrchestrationMergeCmd()
	return cmd.Run(flags)
}
