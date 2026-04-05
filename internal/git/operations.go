package git

import (
	"fmt"
	"strings"
)

var ErrUncommittedChanges = fmt.Errorf("uncommitted changes detected in working tree")

func EnsureCleanWorkingTree() error {
	if HasUncommittedChanges() {
		return ErrUncommittedChanges
	}
	return nil
}

func SyncBranch(auth *AuthConfig, branch string) error {
	if err := Fetch(auth); err != nil {
		return fmt.Errorf("failed to fetch during sync: %w", err)
	}

	if err := PullRebase(auth); err != nil {
		return fmt.Errorf("failed to rebase during sync: %w", err)
	}

	if err := IsBranchSyncedWithRemote(branch); err != nil {
		return fmt.Errorf("branch not synced after rebase: %w", err)
	}

	return nil
}

func CommitWithVerification(message string) error {
	if err := Commit(message); err != nil {
		return err
	}

	commitHash, err := RevParse("HEAD")
	if err != nil {
		return fmt.Errorf("commit created but could not verify: %w", err)
	}

	if commitHash == "" {
		return fmt.Errorf("commit verification failed: empty commit hash")
	}

	return nil
}

func AtomicCommitWithFiles(message string, files []string) error {
	for _, file := range files {
		if err := StageFile(file); err != nil {
			return fmt.Errorf("failed to stage file %s: %w", file, err)
		}
	}

	if !HasStagedChanges() {
		return fmt.Errorf("no files staged after atomic commit operation")
	}

	if err := CommitWithVerification(message); err != nil {
		return fmt.Errorf("atomic commit failed: %w", err)
	}

	return nil
}

func SyncBranchWithVerification(auth *AuthConfig, branch string) (string, error) {
	if err := SyncBranch(auth, branch); err != nil {
		return "", err
	}

	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch after sync: %w", err)
	}

	remoteURL, err := Push(auth, currentBranch)
	if err != nil {
		return "", fmt.Errorf("failed to push after sync: %w", err)
	}

	return remoteURL, nil
}

func FetchAndRebase(auth *AuthConfig, branch string) error {
	if err := Fetch(auth); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	if err := PullRebase(auth); err != nil {
		return fmt.Errorf("failed to rebase: %w", err)
	}

	return nil
}

func ValidateCleanTreeAndCommit(message string, files []string) error {
	if err := EnsureCleanWorkingTree(); err != nil {
		return err
	}

	if err := AtomicCommitWithFiles(message, files); err != nil {
		return err
	}

	return nil
}

func GetStagedFiles() ([]string, error) {
	output, err := runGit("diff", "--cached", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	files := strings.Split(output, "\n")
	result := make([]string, 0, len(files))
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}

	return result, nil
}
