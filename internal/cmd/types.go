package cmd

import (
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

type ExecutionSetup struct {
	ProjectFile   string
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
}
