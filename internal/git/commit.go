package git

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNoChanges = errors.New("no changes to commit")

// workingTreeCommitMessage is used for the commit that sweeps up leftover
// working tree changes before a pull --rebase. It carries no completion trailer.
const workingTreeCommitMessage = "chore: commit working tree before pull"

// CommitWorkingTree stages and commits any uncommitted changes in the working
// tree. It is a no-op when the tree is clean or when nothing can be staged. It
// is used before pull --rebase, which refuses to run on a dirty tree.
func CommitWorkingTree(message string) error {
	if !HasUncommittedChanges() {
		return nil
	}
	if err := StageAll(); err != nil {
		return fmt.Errorf("failed to stage working tree: %w", err)
	}
	if !HasStagedChanges() {
		return nil
	}
	if err := Commit(message); err != nil {
		return fmt.Errorf("failed to commit working tree: %w", err)
	}
	return nil
}

// StageFile stages a specific file using git add
func StageFile(filePath string) error {
	_, err := runGit("add", filePath)
	if err != nil {
		return fmt.Errorf("failed to stage file '%s': %w", filePath, err)
	}
	return nil
}

// StageAll stages all changes using git add -A
func StageAll() error {
	_, err := runGit("add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

// HasStagedChanges checks if there are any staged changes ready to commit
func HasStagedChanges() bool {
	// git diff --cached --quiet: exit 0 = no staged changes, exit 1 = has staged changes
	_, err := runGit("diff", "--cached", "--quiet")
	return err != nil
}

// HasUncommittedChanges reports whether there are any uncommitted changes in the
// working tree or index (i.e. staged or unstaged modifications, additions, or deletions).
func HasUncommittedChanges() bool {
	out, err := runGit("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// Commit creates a git commit with the specified message
func Commit(message string) error {
	_, err := runGit("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// CommitProjectRemoval commits the staged project file deletion on its own with
// a message naming the project file, and no completion trailer.
func CommitProjectRemoval(path string) error {
	return Commit(fmt.Sprintf("chore: clean up completed project %s", path))
}

// CommitMessages returns the full commit messages of the commits on the
// current branch that are not on the base branch, in git log order (newest
// first). Each message is returned verbatim, including any trailing newline git
// stores with it.
func CommitMessages(base string) ([]string, error) {
	logRange := fmt.Sprintf("%s..HEAD", base)
	output, err := runGit("log", "-z", logRange, "--format=%B")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	var messages []string
	for _, m := range strings.Split(output, "\x00") {
		if m != "" {
			messages = append(messages, m)
		}
	}
	return messages, nil
}

// GetCommitLog retrieves commit log formatted exactly like the reference implementation.
// Returns a single string with commits formatted as "%h: %B" (hash: full message).
// Gets all commits since base..HEAD. If limit > 0, only the most recent limit commits are returned.
func GetCommitLog(base string, limit int) (string, error) {
	logRange := fmt.Sprintf("%s..HEAD", base)
	args := []string{"log", logRange, "--format=%h: %B"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	output, err := runGit(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get commit log: %w", err)
	}
	return output, nil
}

func performCommit(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("empty commit message: cannot proceed without a descriptive message")
	}

	if !HasStagedChanges() {
		return ErrNoChanges
	}

	if err := Commit(message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func CommitChanges(isWorkflow bool, owner, repo, message string) error {
	if err := StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	if err := performCommit(message); err != nil {
		return err
	}

	if err := PullAndPush(isWorkflow, owner, repo); err != nil {
		return err
	}

	return nil
}
