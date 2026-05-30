package git

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/context"
)

type Client struct {
	ctx *context.Context
}

func NewClient(ctx *context.Context) *Client {
	return &Client{ctx: ctx}
}

func (a *Client) SwitchToBranch(slug string) error {
	return ValidateGitStateAndSwitchBranch(a.ctx, slug)
}

func (a *Client) BlockedFileExists() bool {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(repoRoot, "blocked.md"))
	return err == nil
}

func (a *Client) WriteBlockedFile(err error) {
	repoRoot, repoErr := FindRepoRoot()
	if repoErr != nil {
		return
	}
	content := fmt.Sprintf("# Blocked\n\nError: %s\n", err.Error())
	_ = os.WriteFile(filepath.Join(repoRoot, "blocked.md"), []byte(content), 0644)
}

func (a *Client) HasChanges() bool {
	return HasUncommittedChanges()
}

func (a *Client) ReportExists() bool {
	_, err := os.Stat("report.md")
	return err == nil
}

func (a *Client) CommitFromReport(slug string) error {
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

func (a *Client) CurrentBranch() (string, error) {
	return GetCurrentBranch()
}

func (a *Client) IsBranchSyncedWithRemote(branch string) error {
	return IsBranchSyncedWithRemote(branch)
}
