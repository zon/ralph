package run

import (
	"fmt"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

type PrepareResult struct {
	ProjectFile   string
	Project       *project.Project
	Config        *config.RalphConfig
	BranchName    string
	CurrentBranch string
	BaseBranch    string
}

func PrepareExecution(ctx *context.Context) (*PrepareResult, error) {
	absProjectFile, err := filepath.Abs(ctx.ProjectFile())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project file path: %w", err)
	}

	proj, err := project.LoadProject(absProjectFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load project file: %w", err)
	}

	branchName := git.SanitizeBranchName(proj.Slug)
	ctx.Output().Debugf("Branch name: %s", branchName)

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	baseBranch := ctx.BaseBranch()
	if baseBranch == "" {
		baseBranch = ralphConfig.DefaultBranch
	}

	return &PrepareResult{
		ProjectFile:   absProjectFile,
		Project:       proj,
		Config:        ralphConfig,
		BranchName:    branchName,
		CurrentBranch: currentBranch,
		BaseBranch:    baseBranch,
	}, nil
}
