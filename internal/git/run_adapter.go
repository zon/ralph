package git

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/context"
)

type RunAdapter struct {
	ctx *context.Context
}

func NewRunAdapter(ctx *context.Context) *RunAdapter {
	return &RunAdapter{ctx: ctx}
}

func (a *RunAdapter) SwitchToBranch(slug string) error {
	return ValidateGitStateAndSwitchBranch(a.ctx, slug)
}

func (a *RunAdapter) BlockedFileExists() bool {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(repoRoot, "blocked.md"))
	return err == nil
}

func (a *RunAdapter) WriteBlockedFile(err error) {
	repoRoot, repoErr := FindRepoRoot()
	if repoErr != nil {
		return
	}
	content := fmt.Sprintf("# Blocked\n\nError: %s\n", err.Error())
	_ = os.WriteFile(filepath.Join(repoRoot, "blocked.md"), []byte(content), 0644)
}

func (a *RunAdapter) HasChanges() bool {
	return HasUncommittedChanges()
}

func (a *RunAdapter) ReportExists() bool {
	_, err := os.Stat("report.md")
	return err == nil
}

func (a *RunAdapter) CommitFromReport(slug string) error {
	data, err := os.ReadFile("report.md")
	if err != nil {
		return fmt.Errorf("failed to read report.md: %w", err)
	}
	message := string(data)
	owner, repo := a.ctx.RepoOwnerAndName()
	if err := CommitChanges(a.ctx.IsWorkflowExecution(), owner, repo, message); err != nil {
		return err
	}
	if err := os.Remove("report.md"); err != nil {
		return fmt.Errorf("failed to remove report.md: %w", err)
	}
	return nil
}
