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

// SwitchToLoopBranch switches to the loop-<slug> branch before the loop's
// agent passes run, creating it from the current branch when it does not
// exist. It mirrors SwitchToBranch for project runs: the branch is resolved
// before any AI work begins, so the agent reviews the loop branch's own state
// and iteration commits never have to move edits across a diverged branch.
func (a *Client) SwitchToLoopBranch(slug string) error {
	return ValidateGitStateAndSwitchBranch(a.ctx, LoopBranch(slug))
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

// CommitIterationAndPush commits the iteration's changes with the report
// content as the commit message and pushes the loop branch. When the working
// tree has no changes it creates an empty commit, so a completion trailer in
// the report is recorded even when no code was written. The caller switches to
// the loop branch first via SwitchToLoopBranch, so this method never switches
// branches while the agent's edits are uncommitted. It removes report.md
// before the commit so it never lands on the loop branch and the working tree
// stays clean.
func (a *Client) CommitIterationAndPush(slug string) error {
	data, err := os.ReadFile("report.md")
	if err != nil {
		return fmt.Errorf("failed to read report.md: %w", err)
	}
	if err := os.Remove("report.md"); err != nil {
		return fmt.Errorf("failed to remove report.md: %w", err)
	}
	owner, repo := a.ctx.RepoOwnerAndName()
	return CommitChangesAllowEmpty(a.ctx.IsWorkflowExecution(), owner, repo, string(data))
}

func (a *Client) CommitFromReport(slug string) error {
	data, err := os.ReadFile("report.md")
	if err != nil {
		return fmt.Errorf("failed to read report.md: %w", err)
	}
	message := string(data)
	owner, repo := a.ctx.RepoOwnerAndName()
	if err := CommitChangesAllowEmpty(a.ctx.IsWorkflowExecution(), owner, repo, message); err != nil {
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

func (a *Client) CommitMessages(base string) ([]string, error) {
	return CommitMessages(base)
}

func (a *Client) IsBranchSyncedWithRemote(branch string) error {
	return IsBranchSyncedWithRemote(branch)
}

func (a *Client) CommitOrchestrationRemoval(_ string) error {
	if err := Commit("chore: remove orchestration doc before PR"); err != nil {
		return err
	}
	return a.pushAfterCommit()
}

func (a *Client) CommitGeneratedArtifacts(slug string) error {
	if err := StageAll(); err != nil {
		return err
	}
	return Commit(fmt.Sprintf("chore: generate project for %s", slug))
}

func (a *Client) CommitProjectRemoval(path string) error {
	if err := CommitProjectRemoval(path); err != nil {
		return err
	}
	return a.pushAfterCommit()
}

// pushAfterCommit pushes the current branch after a pre-PR cleanup commit so
// the cleanup lands on the remote before the pull request is opened, matching
// how iteration commits are pushed by CommitChanges.
func (a *Client) pushAfterCommit() error {
	owner, repo := a.ctx.RepoOwnerAndName()
	return PullAndPush(a.ctx.IsWorkflowExecution(), owner, repo)
}
