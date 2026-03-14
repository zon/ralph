package git

import (
	"bytes"
	gocontext "context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/github"
)

// ErrWorkflowPermission is returned when a push is rejected because the GitHub App
// token lacks the `workflows` permission required to create or update files under
// .github/workflows/. Retrying will not help — the token must be granted the
// permission before the push can succeed.
var ErrWorkflowPermission = errors.New("push rejected: GitHub App token requires `workflows` permission to push workflow files")

// isWorkflowPermissionError checks whether git push output contains the GitHub
// rejection message for missing `workflows` App permission.
func isWorkflowPermissionError(output string) bool {
	return strings.Contains(output, "refusing to allow a GitHub App to create or update workflow") ||
		strings.Contains(output, "without `workflows` permission")
}

// configureAuth refreshes the GitHub App token and configures git HTTPS auth.
// Called before every network git operation when running inside a workflow container.
// Uses ctx.Repo ("owner/name") so that the correct installation is looked up even
// when the process CWD is not the target repository (e.g. when ralph is invoked via
// `go run` from a different directory in debug mode).
func configureAuth(ctx *context.Context) error {
	if !ctx.IsWorkflowExecution() {
		return nil
	}
	owner, repo := ctx.RepoOwnerAndName()
	return github.ConfigureGitAuth(gocontext.Background(), owner, repo, github.DefaultSecretsDir)
}

// IsGitRepository checks if the current directory is inside a git repository
func IsGitRepository(ctx *context.Context) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// FindRepoRoot returns the root directory of the git repository
func FindRepoRoot(ctx *context.Context) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to find repo root: %w (output: %s)", err, out.String())
	}

	repoRoot := strings.TrimSpace(out.String())
	if repoRoot == "" {
		return "", fmt.Errorf("failed to determine repo root")
	}

	return repoRoot, nil
}

// IsDetachedHead checks if the repository is in a detached HEAD state
func IsDetachedHead(ctx *context.Context) (bool, error) {
	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	err := cmd.Run()

	// Exit code 0 = on a branch (not detached)
	// Exit code 1 = detached HEAD
	isDetached := err != nil

	return isDetached, nil
}

// GetCurrentBranch returns the name of the current git branch
// Returns error if in detached HEAD state
func GetCurrentBranch(ctx *context.Context) (string, error) {

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w (output: %s)", err, out.String())
	}

	branch := strings.TrimSpace(out.String())
	if branch == "" {
		return "", fmt.Errorf("failed to determine current branch")
	}

	// Check for detached HEAD state
	if branch == "HEAD" {
		return "", fmt.Errorf("repository is in detached HEAD state, please checkout a branch first")
	}

	return branch, nil
}

// RemoteBranchExists checks if a branch exists on the remote using the already-fetched
// remote-tracking ref. Call Fetch first to ensure refs are up to date.
func RemoteBranchExists(ctx *context.Context, name string) bool {

	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "origin/"+name)
	return cmd.Run() == nil
}

// CheckoutOrCreateBranch checks out the named branch if it exists on the remote
// (after a prior Fetch), otherwise creates and checks out a new local branch.
func CheckoutOrCreateBranch(ctx *context.Context, name string) error {

	if RemoteBranchExists(ctx, name) {
		if err := CheckoutBranch(ctx, name); err != nil {
			return err
		}
		return nil
	}

	cmd := exec.Command("git", "checkout", "-b", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w (output: %s)", name, err, out.String())
	}
	return nil
}

// IsBranchSyncedWithRemote checks if the local branch is in sync with its remote counterpart.
// Returns an error if the remote branch doesn't exist or the local branch is ahead/behind.
func IsBranchSyncedWithRemote(ctx *context.Context, branch string) error {

	// Check that the remote branch exists
	remoteRef := fmt.Sprintf("origin/%s", branch)
	cmd := exec.Command("git", "rev-parse", "--verify", remoteRef)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' has not been pushed to remote - please push before running remotely", branch)
	}

	// Compare local and remote commit hashes
	localCmd := exec.Command("git", "rev-parse", branch)
	var localOut bytes.Buffer
	localCmd.Stdout = &localOut
	if err := localCmd.Run(); err != nil {
		return fmt.Errorf("failed to get local commit for branch '%s': %w", branch, err)
	}

	remoteCmd := exec.Command("git", "rev-parse", remoteRef)
	var remoteOut bytes.Buffer
	remoteCmd.Stdout = &remoteOut
	if err := remoteCmd.Run(); err != nil {
		return fmt.Errorf("failed to get remote commit for branch '%s': %w", branch, err)
	}

	localHash := strings.TrimSpace(localOut.String())
	remoteHash := strings.TrimSpace(remoteOut.String())

	if localHash != remoteHash {
		return fmt.Errorf("branch '%s' is not in sync with remote - please push your changes before running remotely", branch)
	}

	return nil
}

// Fetch fetches from the remote, updating remote-tracking refs.
func Fetch(ctx *context.Context) error {

	if err := configureAuth(ctx); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	cmd := exec.Command("git", "fetch", "origin")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	return nil
}

// remoteBranchExists checks whether a branch exists on the remote.
func remoteBranchExists(branch string) bool {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", branch)
	return cmd.Run() == nil
}

// PullRebase pulls remote changes using rebase to avoid merge commits.
// This should be called before pushing to handle cases where the remote
// branch has advanced (e.g. from a previous run or another pod).
func PullRebase(ctx *context.Context) error {

	if err := configureAuth(ctx); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	branch, err := GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch for pull: %w", err)
	}

	if !remoteBranchExists(branch) {
		return nil
	}

	cmd := exec.Command("git", "pull", "--rebase", "origin", branch)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull --rebase: %w (output: %s)", err, out.String())
	}

	return nil
}

// CheckoutBranch switches to the specified git branch
func CheckoutBranch(ctx *context.Context, name string) error {

	cmd := exec.Command("git", "checkout", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w (output: %s)", name, err, out.String())
	}

	return nil
}

// HasCommits checks if the current branch has any commits
func HasCommits(ctx *context.Context) bool {

	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	return cmd.Run() == nil
}

// Push pushes the current branch or a specified branch to origin.
// If branch is empty, the current branch is pushed.
// Returns the remote URL on success.
func Push(ctx *context.Context, branch string) (string, error) {

	if err := configureAuth(ctx); err != nil {
		return "", fmt.Errorf("failed to configure git auth: %w", err)
	}

	// Determine branch to push
	branchToPush := branch
	if branchToPush == "" {
		var err error
		branchToPush, err = GetCurrentBranch(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Check if there are commits to push
	if !HasCommits(ctx) {
		return "", fmt.Errorf("no commits to push on branch '%s'", branchToPush)
	}

	// Push the branch with --set-upstream
	cmd := exec.Command("git", "push", "--set-upstream", "origin", branchToPush)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		if isWorkflowPermissionError(out.String()) {
			return "", fmt.Errorf("%w (output: %s)", ErrWorkflowPermission, out.String())
		}
		return "", fmt.Errorf("failed to push branch '%s': %w (output: %s)", branchToPush, err, out.String())
	}

	// Get the remote URL
	cmd = exec.Command("git", "config", "--get", "remote.origin.url")
	var urlOut bytes.Buffer
	cmd.Stdout = &urlOut
	cmd.Stderr = &urlOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w (output: %s)", err, urlOut.String())
	}

	remoteURL := strings.TrimSpace(urlOut.String())

	return remoteURL, nil
}

// PushBranch pushes the specified branch to origin and returns the remote URL
// Deprecated: Use Push instead
func PushBranch(ctx *context.Context, branch string) (string, error) {
	return Push(ctx, branch)
}

// PushCurrentBranch pushes the current branch to origin
// Deprecated: Use Push instead
func PushCurrentBranch(ctx *context.Context) error {
	_, err := Push(ctx, "")
	return err
}

// GetCommitLog retrieves commit log formatted exactly like the reference implementation.
// Returns a single string with commits formatted as "%h: %B" (hash: full message).
// Gets all commits since base..HEAD. If limit > 0, only the most recent limit commits are returned.
func GetCommitLog(ctx *context.Context, base string, limit int) (string, error) {

	// Use git log with format matching reference: %h: %B (hash: full message body)
	logRange := fmt.Sprintf("%s..HEAD", base)
	args := []string{"log", logRange, "--format=%h: %B"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit log: %w (output: %s)", err, out.String())
	}

	output := strings.TrimSpace(out.String())
	return output, nil
}

// GetDiffSince returns the diff between the base branch and HEAD
func GetDiffSince(ctx *context.Context, base string) (string, error) {

	// Use git diff to get changes between base and HEAD
	cmd := exec.Command("git", "diff", fmt.Sprintf("%s..HEAD", base))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get diff since '%s': %w (output: %s)", base, err, out.String())
	}

	diff := out.String()

	return diff, nil
}

// StageFile stages a specific file using git add
func StageFile(ctx *context.Context, filePath string) error {

	cmd := exec.Command("git", "add", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file '%s': %w (output: %s)", filePath, err, out.String())
	}

	return nil
}

// StageAll stages all changes using git add -A
func StageAll(ctx *context.Context) error {

	cmd := exec.Command("git", "add", "-A")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes: %w (output: %s)", err, out.String())
	}

	return nil
}

// HasFileChanges checks if a specific file has unstaged changes (i.e. differs from the index)
func HasFileChanges(ctx *context.Context, filePath string) bool {

	// git diff --quiet -- <file>: exit 0 = no changes, exit 1 = has changes
	cmd := exec.Command("git", "diff", "--quiet", "--", filePath)
	err := cmd.Run()
	return err != nil
}

// HasStagedChanges checks if there are any staged changes ready to commit
func HasStagedChanges(ctx *context.Context) bool {

	// Use git diff --cached --quiet to check for staged changes
	// Exit code 0 = no staged changes, exit code 1 = has staged changes
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	return err != nil // Non-zero exit = has changes
}

// HasUncommittedChanges reports whether there are any uncommitted changes in the
// working tree or index (i.e. staged or unstaged modifications, additions, or deletions).
func HasUncommittedChanges(ctx *context.Context) bool {

	// git status --porcelain: empty output = clean, non-empty = dirty
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(out.String()) != ""
}

// Commit creates a git commit with the specified message
func Commit(ctx *context.Context, message string) error {

	cmd := exec.Command("git", "commit", "-m", message)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w (output: %s)", err, out.String())
	}

	return nil
}

// CommitAllowEmpty creates a git commit even when there are no staged changes.
// Used when the AI ran but produced no file changes (e.g. all requirements already passing).
func CommitAllowEmpty(ctx *context.Context, message string) error {

	cmd := exec.Command("git", "commit", "--allow-empty", "-m", message)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit (allow-empty): %w (output: %s)", err, out.String())
	}

	return nil
}

// CommitChanges stages all changes and commits them with a descriptive message
// It generates the commit message based on changed files
// Returns error if there are no changes to commit
func CommitChanges(ctx *context.Context) error {

	// Stage all changes
	if err := StageAll(ctx); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are any staged changes
	if !HasStagedChanges(ctx) {
		return fmt.Errorf("no changes to commit")
	}

	// Get list of changed files to generate commit message
	message, err := generateCommitMessage(ctx)
	if err != nil {
		// Fallback to generic message if generation fails
		message = "Update project files"
	}

	// Commit the staged changes
	if err := Commit(ctx, message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// generateCommitMessage creates a descriptive commit message from changed files
func generateCommitMessage(ctx *context.Context) (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get staged files: %w (output: %s)", err, out.String())
	}

	stagedFiles := strings.TrimSpace(out.String())
	if stagedFiles == "" {
		return "Update files", nil
	}

	files := strings.Split(stagedFiles, "\n")
	fileCount := len(files)

	return buildCommitMessage(files, fileCount), nil
}

func buildCommitMessage(files []string, fileCount int) string {
	switch {
	case fileCount == 1:
		return fmt.Sprintf("Update %s", files[0])
	case fileCount <= 3:
		return fmt.Sprintf("Update %s", strings.Join(files, ", "))
	default:
		return summarizeCommitMessage(files, fileCount)
	}
}

func summarizeCommitMessage(files []string, fileCount int) string {
	categories := categorizeFiles(files)
	if len(categories) == 1 {
		for category := range categories {
			return fmt.Sprintf("Update %s files (%d files)", category, fileCount)
		}
	}
	return fmt.Sprintf("Update %d files across project", fileCount)
}

// categorizeFile categorizes a single file by directory or extension
func categorizeFile(file string) string {
	if strings.Contains(file, "/") {
		parts := strings.Split(file, "/")
		if len(parts) > 1 {
			return parts[0]
		}
	}
	if strings.Contains(file, ".") {
		parts := strings.Split(file, ".")
		return parts[len(parts)-1]
	}
	return "root"
}

// categorizeFiles groups files by directory or type
func categorizeFiles(files []string) map[string]int {
	categories := make(map[string]int)

	for _, file := range files {
		category := categorizeFile(file)
		categories[category]++
	}

	return categories
}

// DeleteFile removes a file from the filesystem and stages the deletion
func DeleteFile(ctx *context.Context, filePath string) error {

	// Remove the file from filesystem
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file '%s': %w", filePath, err)
	}

	// Stage the deletion
	cmd := exec.Command("git", "rm", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage deletion of '%s': %w (output: %s)", filePath, err, out.String())
	}

	return nil
}
