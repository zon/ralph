package run

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
)

// GitClientAdapter adapts git functions to the GitClient interface.
type GitClientAdapter struct {
	ctx *context.Context
}

func NewGitClientAdapter(ctx *context.Context) *GitClientAdapter {
	return &GitClientAdapter{ctx: ctx}
}

func (a *GitClientAdapter) SwitchToBranch(slug string) error {
	return git.ValidateGitStateAndSwitchBranch(a.ctx, slug)
}

func (a *GitClientAdapter) BlockedFileExists() bool {
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(repoRoot, "blocked.md"))
	return err == nil
}

func (a *GitClientAdapter) WriteBlockedFile(err error) {
	repoRoot, repoErr := git.FindRepoRoot()
	if repoErr != nil {
		return
	}
	content := fmt.Sprintf("# Blocked\n\nError: %s\n", err.Error())
	_ = os.WriteFile(filepath.Join(repoRoot, "blocked.md"), []byte(content), 0644)
}

func (a *GitClientAdapter) HasChanges() bool {
	return git.HasUncommittedChanges()
}

func (a *GitClientAdapter) ReportExists() bool {
	_, err := os.Stat("report.md")
	return err == nil
}

func (a *GitClientAdapter) CommitFromReport(slug string) error {
	data, err := os.ReadFile("report.md")
	if err != nil {
		return fmt.Errorf("failed to read report.md: %w", err)
	}
	message := string(data)
	owner, repo := a.ctx.RepoOwnerAndName()
	if err := git.CommitChanges(a.ctx.IsWorkflowExecution(), owner, repo, message); err != nil {
		return err
	}
	if err := os.Remove("report.md"); err != nil {
		return fmt.Errorf("failed to remove report.md: %w", err)
	}
	return nil
}
